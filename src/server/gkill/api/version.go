package api

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

func GetVersion() (*GkillVersionData, error) {
	ctx := context.Background()
	assetsFileName := "embed/version.json"
	versionJSONFile, err := EmbedFS.Open(assetsFileName)
	if err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
	}
	defer func() {
		err := versionJSONFile.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	versionData := &GkillVersionData{}
	err = json.NewDecoder(versionJSONFile).Decode(versionData)
	return versionData, err
}
