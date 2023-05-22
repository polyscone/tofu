package repo

import (
	"github.com/polyscone/tofu/internal/repo/account"
	"github.com/polyscone/tofu/internal/repo/web"
)

var (
	NewSQLiteAccountUserRepo      = account.NewSQLiteUserRepo
	NewSQLiteAccountUserQueryRepo = account.NewSQLiteUserQueryRepo

	NewSQLiteWebTokenRepo   = web.NewSQLiteTokenRepo
	NewSQLiteWebSessionRepo = web.NewSQLiteSessionRepo
)
