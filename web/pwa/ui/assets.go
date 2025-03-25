package ui

import (
	"embed"
	"io/fs"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/fsx"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/shared"
)

//go:embed "all:email"
//go:embed "all:master"
//go:embed "all:view"
var files embed.FS

var templateFiles = fsx.NewStack(fsx.RelDirFS(""), files)

//go:embed "all:public"
var publicFiles embed.FS

const publicDir = "public"

var AssetFiles = fsx.NewStack(
	fsx.RelDirFS(publicDir),
	errsx.Must(fs.Sub(publicFiles, publicDir)),
	shared.AssetFiles,
	api.AssetFilesV1.Mount(app.BasePath+"/js/api/v1/"),
)

var AssetTags = cache.New[string, string]()
