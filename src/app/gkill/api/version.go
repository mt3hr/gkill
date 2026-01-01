package api

import (
	"encoding/json"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

func GetVersion() (*GkillVersionData, error) {
	assetsFileName := "embed/version.json"
	versionJSONFile, err := EmbedFS.Open(assetsFileName)
	if err != nil {
		gkill_log.Error.Println(err.Error())
	}
	defer versionJSONFile.Close()

	versionData := &GkillVersionData{}
	err = json.NewDecoder(versionJSONFile).Decode(versionData)
	return versionData, err
}
