package config

import (
	"embed"

	"github.com/polyscone/tofu/internal/i18n"
)

//go:embed "all:locale"
var locales embed.FS

func LoadI18nLocales() error {
	return i18n.LoadJSONFiles(locales)
}
