package ui

import (
	"embed"
	"io/fs"

	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/fsx"
	"github.com/polyscone/tofu/web/shared"
)

//go:embed "all:public"
var publicFilesV1 embed.FS

const publicDir = "public"

var AssetFiles = fsx.NewStack(
	fsx.RelDirFS(publicDir),
	errsx.Must(fs.Sub(publicFilesV1, publicDir)),
	shared.AssetFiles,
)

var AssetTags = cache.New[string, string]()
