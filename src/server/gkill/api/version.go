package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

func GetVersion() (*GkillVersionData, error) {
	ctx := context.Background()
	assetsFileName := "embed/version.json"
	versionJSONFile, err := EmbedFS.Open(assetsFileName)
	if err != nil {
		// ログするだけで進むと、この下の defer が nil に対して Close() を呼んで
		// nil pointer dereference で落ちる。version.json は prepare_install が
		// 生成するもので、それを踏まないビルドでは存在しない。
		// 実際 CI で /api/get_application_config がこれで panic した。
		err = fmt.Errorf("error at open %s: %w", assetsFileName, err)
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return nil, err
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
