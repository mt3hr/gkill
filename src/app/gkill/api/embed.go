package api

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	//go:embed embed
	EmbedFS embed.FS // htmlファイル郡

	localizers map[string]*i18n.Localizer = map[string]*i18n.Localizer{}
)

func GetLocalizer(localeName string) *i18n.Localizer {
	localizer, exist := localizers[localeName]
	if !exist {
		return localizers["ja"] // nil回避 デフォはja
	}
	return localizer
}

func init() {
	jsonFileNames := []string{}
	locales, err := EmbedFS.ReadDir("embed/i18n/locales")
	if err != nil {
		gkill_log.Error.Println(err.Error())
		panic(err)
	}
	for _, locale := range locales {
		info, err := locale.Info()
		if err != nil {
			gkill_log.Error.Println(err.Error())
			panic(err)
		}
		jsonFileNames = append(jsonFileNames, info.Name())
	}

	bundle := i18n.NewBundle(language.English)
	for _, jsonFileName := range jsonFileNames {
		base := filepath.Base(jsonFileName)
		fullPath := filepath.ToSlash(filepath.Join("embed/i18n/locales", base))
		localeName := strings.ReplaceAll(jsonFileName, ".json", "")

		jsonFile, err := EmbedFS.ReadFile(fullPath)
		if err != nil {
			gkill_log.Error.Println(err.Error())
			panic(err)
		}
		bundle.MustParseMessageFileBytes(jsonFile, base)

		localizers[localeName] = i18n.NewLocalizer(bundle, localeName)
	}
}
