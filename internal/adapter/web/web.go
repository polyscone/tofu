package web

import "github.com/polyscone/tofu/internal/adapter/web/internal/repo/sqlite"

var (
	NewSQLiteSessionRepo = sqlite.NewSessionRepo
	NewSQLiteTokenRepo   = sqlite.NewTokenRepo
)
