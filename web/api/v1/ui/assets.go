package ui

import (
	"embed"
	"io/fs"

	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/fsx"
	"github.com/polyscone/tofu/web/shared"
)

//go:embed "all:public"
var publicFilesV1 embed.FS

const publicDirV1 = "public"

var AssetFilesV1 = fsx.NewStack(
	fsx.RelDirFS(publicDirV1),
	errsx.Must(fs.Sub(publicFilesV1, publicDirV1)),
	shared.AssetFiles,
)
