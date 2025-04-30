package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/polyscone/tofu/internal/human"
	"github.com/polyscone/tofu/internal/size"
	"github.com/polyscone/tofu/internal/uuid"
	"github.com/polyscone/tofu/repo/sqlite"
	"github.com/polyscone/tofu/web/handler"
)

type RecoveryService struct {
	mu        sync.Mutex
	dataDir   string
	logger    *slog.Logger
	sqliteDBs map[string]*sqlite.DB
}

func NewRecoveryService(dataDir string, logger *slog.Logger) *RecoveryService {
	return &RecoveryService{
		dataDir:   dataDir,
		logger:    logger,
		sqliteDBs: make(map[string]*sqlite.DB),
	}
}

func (r *RecoveryService) sqliteOnlineBackup(ctx context.Context, dst, src *sqlite.DB) error {
	srcConn, err := src.Conn(ctx)
	if err != nil {
		return fmt.Errorf("src conn: %w", err)
	}
	defer srcConn.Close()

	dstConn, err := dst.Conn(ctx)
	if err != nil {
		return fmt.Errorf("dst conn: %w", err)
	}
	defer dstConn.Close()

	err = srcConn.Raw(func(dc any) error {
		srcRaw, ok := dc.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("src conn raw: could not assert %T", srcRaw)
		}

		err := dstConn.Raw(func(dc any) error {
			dstRaw, ok := dc.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("dst conn raw: could not assert %T", dstRaw)
			}

			backup, err := dstRaw.Backup("main", srcRaw, "main")
			if err != nil {
				return fmt.Errorf("backup: %w", err)
			}

			for {
				done, err := backup.Step(-1)
				if err != nil {
					if sqlite.IsBusyOrLocked(err) {
						time.Sleep(100 * time.Millisecond)

						continue
					}

					return fmt.Errorf("step: %w", err)
				}

				if done {
					break
				}

				time.Sleep(5 * time.Millisecond)
			}

			if err := backup.Finish(); err != nil {
				return fmt.Errorf("finish: %w", err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("dst conn: raw: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("src conn: raw: %w", err)
	}

	return nil
}

func (r *RecoveryService) Backup(ctx context.Context, w io.Writer, opts handler.BackupOptions) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	zw := zip.NewWriter(w)
	defer zw.Close()

	tmpID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("new v4 UUID: %w", err)
	}

	tmp := filepath.Join(r.dataDir, "backup."+tmpID.String())
	if err := os.MkdirAll(tmp, 0755); err != nil {
		return fmt.Errorf("make temp backup directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	if opts.Database {
		for name, srcDB := range r.sqliteDBs {
			err := func() error {
				p := filepath.Join(tmp, name)
				dstDB, err := sqlite.Open(ctx, sqlite.KindFile, p, nil, nil)
				if err != nil {
					return fmt.Errorf("open SQLite database: %w", err)
				}
				defer dstDB.Close()

				if err := r.sqliteOnlineBackup(ctx, dstDB, srcDB); err != nil {
					return fmt.Errorf("SQLite online backup: %w", err)
				}

				// We explicitly close the destination connection pool here to give
				// SQLite the chance to do things like writing the WAL into the
				// main database file etc. if needed
				if err := dstDB.Close(); err != nil {
					return fmt.Errorf("close: %w", err)
				}

				f, err := os.Open(p)
				if err != nil {
					return fmt.Errorf("dst open: %w", err)
				}
				defer f.Close()

				zf, err := zw.Create(name)
				if err != nil {
					return fmt.Errorf("zip create: %w", err)
				}

				if _, err := io.Copy(zf, f); err != nil {
					return fmt.Errorf("zip copy: %w", err)
				}

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *RecoveryService) Restore(ctx context.Context, zr *zip.Reader, opts handler.RestoreOptions) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("new v4 UUID: %w", err)
	}

	tmpName := "restore." + tmpID.String()
	tmp := filepath.Join(r.dataDir, tmpName)
	if err := os.MkdirAll(tmp, 0755); err != nil {
		return fmt.Errorf("make temp backup directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	allowedExts := map[string]struct{}{
		".bmp":    {},
		".gif":    {},
		".jpeg":   {},
		".jpg":    {},
		".png":    {},
		".sqlite": {},
	}

	for _, zf := range zr.File {
		err := func() error {
			ext := strings.ToLower(path.Ext(zf.Name))
			if _, ok := allowedExts[ext]; !ok {
				return nil
			}

			_, allowed := r.sqliteDBs[zf.Name]
			allowed = allowed || path.Dir(zf.Name) == "photos"
			allowed = allowed || path.Dir(zf.Name) == "proof"
			if !allowed {
				return nil
			}

			// Protect against zip slip directory traversal by checking the path prefix
			p := filepath.Join(tmp, zf.Name)
			if !strings.HasPrefix(p, filepath.Clean(tmp)+string(os.PathSeparator)) {
				return fmt.Errorf("zip illegal file path: %v", p)
			}

			if zf.FileInfo().IsDir() {
				if err := os.MkdirAll(p, zf.Mode()); err != nil {
					return fmt.Errorf("make destination directory: %w", err)
				}

				return nil
			}

			src, err := zf.Open()
			if err != nil {
				return fmt.Errorf("zip open: %w", err)
			}
			defer src.Close()

			if err := os.MkdirAll(filepath.Dir(p), zf.Mode()); err != nil {
				return fmt.Errorf("make destination file directory: %w", err)
			}

			dst, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zf.Mode())
			if err != nil {
				return err
			}
			defer dst.Close()

			// Protect against zip bombs by copying up to a max size
			const maxFileSize = 100 * size.Megabyte
			n, err := io.CopyN(dst, src, maxFileSize)
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("zip copyn: %w", err)
			}
			if zf.UncompressedSize64 != uint64(n) {
				return fmt.Errorf(
					"zipped file %q too large (%v): max file size %v",
					zf.Name,
					human.SizeSI(int64(zf.UncompressedSize64)),
					human.SizeSI(maxFileSize),
				)
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	var tmpSQLiteDBs map[string]*sqlite.DB
	if opts.Database {
		tmpSQLiteDBs = make(map[string]*sqlite.DB)
		for name, srcDB := range r.sqliteDBs {
			err := func() error {
				dstName := tmpName + name
				dstDB, err := sqlite.Open(ctx, sqlite.KindMemory, dstName, nil, nil)
				if err != nil {
					return fmt.Errorf("open SQLite database: %w", err)
				}

				if err := r.sqliteOnlineBackup(ctx, dstDB, srcDB); err != nil {
					return fmt.Errorf("SQLite online backup: %w", err)
				}

				tmpSQLiteDBs[name] = dstDB

				return nil
			}()
			if err != nil {
				return err
			}
		}

		for _, db := range tmpSQLiteDBs {
			defer db.Close()
		}
	}

	// If success doesn't get set to true we'll assume failure and move
	// all of the old data back to its original location
	var success bool
	defer func() {
		if success {
			return
		}

		if opts.Database {
			for name, srcDB := range tmpSQLiteDBs {
				dstDB, ok := r.sqliteDBs[name]
				if !ok {
					continue
				}

				if err := r.sqliteOnlineBackup(ctx, dstDB, srcDB); err != nil {
					r.logger.Error("restore SQLite online backup", "error", err)
				}
			}
		}
	}()

	// Attempt online backups of SQLite databases
	if opts.Database {
		for name, dstDB := range r.sqliteDBs {
			err := func() error {
				p := filepath.Join(tmp, name)
				srcDB, err := sqlite.Open(ctx, sqlite.KindFile, p, nil, nil)
				if err != nil {
					return fmt.Errorf("open SQLite database: %w", err)
				}
				defer srcDB.Close()

				if err := r.sqliteOnlineBackup(ctx, dstDB, srcDB); err != nil {
					return fmt.Errorf("SQLite online backup: %w", err)
				}

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	success = true

	return nil
}
