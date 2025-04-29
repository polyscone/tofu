package system

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterRecoveryHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /admin/system/recovery", recoveryDashboardGet(h), "system.recovery")

	mux.HandleFunc("GET /admin/system/recovery/backup", recoveryBackupGet(h), "system.recovery.backup")

	mux.HandleFunc("GET /admin/system/recovery/restore", recoveryRestoreGet(h))
	mux.HandleFunc("POST /admin/system/recovery/restore", recoveryRestorePost(h), "system.recovery.restore.post")
}

func recoveryDashboardGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.System.CanBackup() || p.System.CanRestore() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		h.HTML.View(w, r, http.StatusOK, "system/recovery", nil)
	}
}

func recoveryBackupGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.System.CanBackup() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		ctx := r.Context()

		var buf bytes.Buffer
		opts := handler.BackupOptions{Database: true}
		if err := h.Recovery.Backup(ctx, &buf, opts); err != nil {
			h.HTML.ErrorView(w, r, "backup", err, "error", nil)

			return
		}

		datetime := time.Now().UTC().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("system_backup_%v_utc.zip", datetime)

		w.Header().Set("content-type", "application/zip")
		w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=%q", filename))

		buf.WriteTo(w)
	}
}

func recoveryRestoreGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.Path("recovery.dashboard"), http.StatusSeeOther)
	}
}

func recoveryRestorePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.System.CanRestore() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		ctx := r.Context()

		f, header, err := r.FormFile("restore-archive")
		if err != nil {
			h.HTML.ErrorView(w, r, "restore form file", err, "error", nil)

			return
		}
		defer f.Close()

		zr, err := zip.NewReader(f, header.Size)
		if err != nil {
			h.HTML.ErrorView(w, r, "restore new zip reader", err, "error", nil)

			return
		}

		opts := handler.RestoreOptions{Database: true}
		if err := h.Recovery.Restore(ctx, zr, opts); err != nil {
			h.HTML.ErrorView(w, r, "restore", err, "error", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.admin.recovery.flash.system_restored"))

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "error", nil)

			return
		}

		h.Session.Destroy(r.Context())

		http.Redirect(w, r, h.Path("recovery.dashboard"), http.StatusSeeOther)
	}
}
