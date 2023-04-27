package web

import "github.com/polyscone/tofu/internal/adapter/web/repo/sqlite"

var (
	NewSQLiteSessionRepo = sqlite.NewSessionRepo
	NewSQLiteTokenRepo   = sqlite.NewTokenRepo
)
