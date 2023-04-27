package sqlite

import "embed"

//go:embed "migrations"
var migrations embed.FS
