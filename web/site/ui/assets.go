package ui

import (
	"embed"
	"io/fs"
	"path"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/fsx"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/shared"
)

//go:embed "all:component"
//go:embed "all:email"
//go:embed "all:master"
//go:embed "all:partial"
//go:embed "all:view"
var files embed.FS

var templateFiles = fsx.NewStack(fsx.RelDirFS(""), files)

var componentFilesExtWhitelist = map[string]struct{}{
	".bmp":  {},
	".css":  {},
	".gif":  {},
	".jpeg": {},
	".jpg":  {},
	".js":   {},
	".json": {},
	".png":  {},
	".txt":  {},
	".webp": {},
}

var componentFiles = fsx.NewRestricted(templateFiles, fsx.RestrictedConfig{
	AllowOpen: func(name string) (bool, error) {
		if !strings.HasPrefix(name, "component/") {
			return false, nil
		}

		ext := path.Ext(name)
		_, ok := componentFilesExtWhitelist[ext]

		return ok, nil
	},
})

//go:embed "all:public"
var publicFiles embed.FS

const publicDir = "public"

var AssetFiles = fsx.NewStack(
	componentFiles,
	fsx.RelDirFS(publicDir),
	errsx.Must(fs.Sub(publicFiles, publicDir)),
	shared.AssetFiles,
	api.AssetFilesV1.Mount(app.BasePath+"/js/api/v1/"),
)

var AssetTags = cache.New[string, string]()
