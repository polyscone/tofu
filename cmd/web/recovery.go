package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/polyscone/tofu/internal/human"
	"github.com/polyscone/tofu/internal/size"
	"github.com/polyscone/tofu/internal/uuid"
	"github.com/polyscone/tofu/repo/sqlite"
	"github.com/polyscone/tofu/web/handler"
)

type RecoveryService struct {
	dataDir   string
	sqliteDBs map[string]*sqlite.DB
}

func NewRecoveryService(dataDir string) *RecoveryService {
	return &RecoveryService{
		dataDir:   dataDir,
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
	zw := zip.NewWriter(w)
	defer zw.Close()

	id, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("new v4 UUID: %w", err)
	}

	tmp := filepath.Join(r.dataDir, "backup_"+id.String())
	if err := os.MkdirAll(tmp, 0755); err != nil {
		return fmt.Errorf("make temp backup directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	if opts.Database {
		for name, srcDB := range r.sqliteDBs {
			p := filepath.Join(tmp, name)
			dstDB, err := sqlite.Open(ctx, sqlite.KindFile, p, nil, nil)
			if err != nil {
				return fmt.Errorf("open SQLite database: %w", err)
			}
			defer dstDB.Close()

			if err := r.sqliteOnlineBackup(ctx, dstDB, srcDB); err != nil {
				return fmt.Errorf("SQLite online backup: %w", err)
			}

			// We explicitly close the destination connection here to give
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
		}
	}

	return nil
}

func (r *RecoveryService) Restore(ctx context.Context, zr *zip.Reader, opts handler.RestoreOptions) error {
	id, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("new v4 UUID: %w", err)
	}

	tmp := filepath.Join(r.dataDir, "restore_"+id.String())
	if err := os.MkdirAll(tmp, 0755); err != nil {
		return fmt.Errorf("make temp backup directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	for _, zf := range zr.File {
		err := func() error {
			var dir string
			switch {
			case strings.HasSuffix(zf.Name, ".sqlite"):
				if !opts.Database {
					return nil
				}

				dir = tmp

			default:
				return nil
			}

			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("make restore directory: %w", err)
			}

			// We only use the base name of the zipped file here with predefined
			// directory names to help protect against zip slip vulnerabilities
			name := path.Base(zf.Name)
			p := filepath.Join(dir, name)
			dst, err := os.Create(p)
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			defer dst.Close()

			src, err := zf.Open()
			if err != nil {
				return fmt.Errorf("zip open: %w", err)
			}
			defer src.Close()

			// We limit the amount of data to copy to a max size per file here
			// to help protect against zip bombs being uploaded
			const maxFileSize = 50 * size.Megabyte
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

			switch {
			case strings.HasSuffix(name, ".sqlite"):
				dstDB, ok := r.sqliteDBs[name]
				if !ok {
					return nil
				}

				srcDB, err := sqlite.Open(ctx, sqlite.KindFile, p, nil, nil)
				if err != nil {
					return fmt.Errorf("open SQLite database: %w", err)
				}
				defer srcDB.Close()

				if err := r.sqliteOnlineBackup(ctx, dstDB, srcDB); err != nil {
					return fmt.Errorf("SQLite online backup: %w", err)
				}

				return nil
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
