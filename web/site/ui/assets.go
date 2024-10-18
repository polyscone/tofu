package ui

import (
	"embed"
	"io/fs"
	"path"
	"strings"

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

var componentFiles = fsx.NewRestricted(templateFiles, func(name string) bool {
	if !strings.HasPrefix(name, "component/") {
		return false
	}

	ext := path.Ext(name)
	_, ok := componentFilesExtWhitelist[ext]

	return ok
})

//go:embed "all:public"
var publicFiles embed.FS

const publicDir = "public"

var AssetFiles = fsx.NewStack(
	componentFiles,
	fsx.RelDirFS(publicDir),
	errsx.Must(fs.Sub(publicFiles, publicDir)),
	shared.AssetFiles,
)

var AssetTags = cache.New[string, string]()
