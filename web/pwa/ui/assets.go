package ui

import (
	"embed"
	"io/fs"

	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/fsx"
	"github.com/polyscone/tofu/web/shared"
)

//go:embed "all:template"
var files embed.FS

const templateDir = "template"

var templateFiles = fsx.NewStack(
	fsx.RelDirFS(templateDir),
	errsx.Must(fs.Sub(files, templateDir)),
)

//go:embed "all:public"
var publicFiles embed.FS

const publicDir = "public"

var AssetFiles = fsx.NewStack(
	fsx.RelDirFS(publicDir),
	errsx.Must(fs.Sub(publicFiles, publicDir)),
	shared.AssetFiles,
)

var AssetTags = cache.New[string, string]()
