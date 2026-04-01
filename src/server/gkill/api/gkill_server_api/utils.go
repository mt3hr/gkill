package gkill_server_api

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/twpayne/go-gpx"
)

func GenerateNewID() string {
	return uuid.New().String()
}

func (g *GkillServerAPI) resolveFileName(repDir string, filename string, behavior req_res.FileUploadConflictBehavior) (string, error) {
	// パストラバーサル対策: ファイル名をサニタイズしてrepDir外へのアクセスを禁止する
	cleanFilename := filepath.Clean(filename)
	if filepath.IsAbs(cleanFilename) || cleanFilename == ".." || strings.HasPrefix(cleanFilename, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid filename: path traversal detected")
	}
	fullFilename := filepath.Join(repDir, cleanFilename)
	_, err := os.Stat(fullFilename)
	if err != nil {
		return fullFilename, nil
	} else {
		switch string(behavior) {
		case string(req_res.Override):
			return fullFilename, nil
		case string(req_res.Rename):
			// カッコのついていないファイル名。例えば「hogehoge (1).txt」なら「hogehoge.txt」。
			planeFileName := g.planeFileName(fullFilename)
			ext := filepath.Ext(planeFileName)
			withoutExt := planeFileName[:len(planeFileName)-len(ext)]

			// ファイルが存在しない名前になるまでカッコ内の数字をインクリメントし続ける
			// targetFilenameは最終的な移動先ファイル名
			fullFilename = planeFileName
			for count := 1; ; count++ {
				if _, err := os.Stat(fullFilename); err != nil {
					break
				}
				fullFilename = os.Expand("${name} (${count})${ext}", func(str string) string {
					switch str {
					case "name":
						return withoutExt
					case "count":
						return strconv.Itoa(count)
					case "ext":
						return ext
					}
					return ""
				})
			}
			return fullFilename, nil
		case string(req_res.Merge):
			return fullFilename, nil
		}
	}
	err = fmt.Errorf("does not set file upload conflict behavior")
	return "", err
}

func (g *GkillServerAPI) generateGPXFileContent(gpsLogs []reps.GPSLog) (string, error) {
	gpxData := &gpx.GPX{}
	gpxData.Trk = []*gpx.TrkType{&gpx.TrkType{}}
	gpxData.Trk[0].TrkSeg = []*gpx.TrkSegType{&gpx.TrkSegType{}}
	trkPts := []*gpx.WptType{}
	for _, gpslog := range gpsLogs {
		trkPts = append(trkPts, &gpx.WptType{
			Time: gpslog.RelatedTime,
			Lat:  gpslog.Latitude,
			Lon:  gpslog.Longitude,
		})
	}
	gpxData.Trk[0].TrkSeg[0].TrkPt = trkPts

	buf := bytes.NewBufferString("")
	writer := bufio.NewWriter(buf)
	err := gpxData.Write(writer)
	if err != nil {
		err = fmt.Errorf("error at write gpx data: %w", err)
		return "", err
	}

	err = writer.Flush()
	if err != nil {
		err = fmt.Errorf("error at write gpx data flush: %w", err)
		return "", err
	}

	return buf.String(), nil
}

func (g *GkillServerAPI) initializeNewUserReps(ctx context.Context, account *account.Account) error {
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		return err
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(ctx, device)
	if err != nil {
		err = fmt.Errorf("error at get server config: %w", err)
		return err
	}

	userDataRootDirectory := filepath.Join(os.ExpandEnv(serverConfig.UserDataDirectory), account.UserID)
	if _, err := os.Stat(os.ExpandEnv(userDataRootDirectory)); err == nil {
		err := fmt.Errorf("error at initialize new user reps. user root directory already exist %s: %w", userDataRootDirectory, err)
		return err
	} else {
		err := os.MkdirAll(os.ExpandEnv(userDataRootDirectory), fs.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at initialize new user reps. error at create directory %s: %w", userDataRootDirectory, err)
			return err
		}
	}

	repositories := []*user_config.Repository{}

	repTypeFileNameMap := map[string]string{}
	repTypeFileNameMap["kmemo"] = "Kmemo.db"
	repTypeFileNameMap["kc"] = "KC.db"
	repTypeFileNameMap["urlog"] = "URLog.db"
	repTypeFileNameMap["timeis"] = "TimeIs.db"
	repTypeFileNameMap["mi"] = "Mi.db"
	repTypeFileNameMap["nlog"] = "Nlog.db"
	repTypeFileNameMap["lantana"] = "Lantana.db"
	repTypeFileNameMap["tag"] = "Tag.db"
	repTypeFileNameMap["text"] = "Text.db"
	repTypeFileNameMap["notification"] = "Notification.db"
	repTypeFileNameMap["rekyou"] = "ReKyou.db"

	for repType, repFileName := range repTypeFileNameMap {
		repFileFullName := filepath.Join(userDataRootDirectory, repFileName)
		repFile, err := os.Create(os.ExpandEnv(repFileFullName))
		if err != nil {
			err = fmt.Errorf("error at create rep file %s: %w", repFileFullName, err)
			return err
		}
		err = repFile.Close()
		if err != nil {
			err = fmt.Errorf("error at close rep file %s: %w", repFileFullName, err)
			return err
		}

		repository := &user_config.Repository{
			ID:                        GenerateNewID(),
			UserID:                    account.UserID,
			Device:                    device,
			Type:                      repType,
			File:                      repFileFullName,
			UseToWrite:                true,
			IsExecuteIDFWhenReload:    true,
			IsWatchTargetForUpdateRep: false,
			IsEnable:                  true,
		}
		repositories = append(repositories, repository)
	}

	repType, repFileName := "directory", "Files"
	repFileFullName := filepath.Join(userDataRootDirectory, repFileName)
	err = os.MkdirAll(os.ExpandEnv(repFileFullName), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository create directory reptype = %s repdirname = %s: %w", repType, repFileFullName, err)
		return err
	}
	repository := &user_config.Repository{
		ID:                        GenerateNewID(),
		UserID:                    account.UserID,
		Device:                    device,
		Type:                      repType,
		File:                      repFileFullName,
		UseToWrite:                true,
		IsExecuteIDFWhenReload:    true,
		IsWatchTargetForUpdateRep: false,
		IsEnable:                  true,
	}
	repositories = append(repositories, repository)

	repType, repFileName = "gpslog", "GPSLog"
	repFileFullName = filepath.Join(userDataRootDirectory, repFileName)
	err = os.MkdirAll(os.ExpandEnv(repFileFullName), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository create directory reptype = %s repdirname = %s: %w", repType, repFileFullName, err)
		return err
	}
	repository = &user_config.Repository{
		ID:                        GenerateNewID(),
		UserID:                    account.UserID,
		Device:                    device,
		Type:                      repType,
		File:                      repFileFullName,
		UseToWrite:                true,
		IsExecuteIDFWhenReload:    true,
		IsWatchTargetForUpdateRep: false,
		IsEnable:                  true,
	}
	repositories = append(repositories, repository)

	ok, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.DeleteWriteRepositories(ctx, account.UserID, repositories)
	if !ok || err != nil {
		err = fmt.Errorf("error at delete write repositories: %w", err)
		return err
	}

	return nil
}

// ファイル名に(n)がついていたら除去して返します。
// hogehoge.txt (1) (1) (1)とかにならないように。
// Windowsのファイル重複時Suffixに対応しています。？
func (g *GkillServerAPI) planeFileName(filename string) (fixedfilename string) {
	_ = "${name} (${count})${ext}" //このフォーマットが対象です。

	ext := filepath.Ext(filename)
	fnwithoutext := filename[:len(filename)-len(ext)]

	//それぞれLastIndex
	lindexP := strings.LastIndexAny(fnwithoutext, " (") //スペースがあります。
	lindexS := strings.LastIndexAny(fnwithoutext, ")")
	if lindexP != -1 && lindexS != -1 && //(と)が含まれていて、
		lindexS == len(fnwithoutext)-1 && //)が一番最後で、
		lindexP < lindexS { //)よりも(が前にあり、
		//その上括弧の間が数字であるとき、それは${count}でつけられたsuffixでありえる。
		num := fnwithoutext[lindexP+1 : lindexS] //スペース分+1
		_, err := strconv.Atoi(num)
		if err == nil {
			//${count}部分を除去して返す
			fnwithoutext = fnwithoutext[:len(fnwithoutext)-(len(num)+3)] //+3はカッコ2つとスペース分
			filename = fnwithoutext + ext
			return filename
		}
	}
	//${count}部分がなければそのまま返す
	return filename
}

func (g *GkillServerAPI) getTLSFileNames(device string) (certFileName string, pemFileName string, err error) {
	ctx := context.Background()
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		return "", "", err
	}
	return serverConfig.TLSCertFile, serverConfig.TLSKeyFile, nil
}

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

func httpGetBase64Data(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("error at new http get request: %w", err)
		return "", err
	}
	req.Header.Set("Referer", url)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("error at http get %s: %w", url, err)
		return "", err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("error at read all body %s: %w", url, err)
		return "", err
	}

	base64Data := base64.StdEncoding.EncodeToString(b)
	return base64Data, nil
}

func (g *GkillServerAPI) ifRedirectResetAdminAccountIsNotFound(w http.ResponseWriter, r *http.Request) bool {
	// GET 以外は対象外
	if r.Method != http.MethodGet {
		return false
	}

	// ブラウザの通常ナビゲーション(HTMLドキュメント)の時だけ
	if d := r.Header.Get("Sec-Fetch-Dest"); d != "" && d != "document" {
		return false
	}
	if m := r.Header.Get("Sec-Fetch-Mode"); m != "" && m != "navigate" && m != "nested-navigate" {
		return false
	}

	p := r.URL.Path
	if strings.HasPrefix(p, "/assets/") ||
		strings.HasSuffix(p, ".js") ||
		strings.HasSuffix(p, ".css") ||
		strings.HasSuffix(p, ".map") ||
		strings.HasSuffix(p, ".png") ||
		strings.HasSuffix(p, ".svg") ||
		strings.HasSuffix(p, ".ico") ||
		strings.HasSuffix(p, ".webmanifest") {
		return false
	}

	accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get all account config")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllAccountConfigError,
			ErrorMessage: api.GetLocalizer("").MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ACCOUNT_CONFIG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return false
	}

	if len(accounts) == 1 {
		if accounts[0].UserID != "admin" || accounts[0].PasswordSha256 != nil {
			return false
		}

		http.Redirect(w, r, fmt.Sprintf("/regist_first_account?reset_token=%s", *accounts[0].PasswordResetToken), http.StatusTemporaryRedirect)
		// http.Redirect(w, r, fmt.Sprintf("/set_new_password?reset_token=%s&user_id=%s", *accounts[0].PasswordResetToken, accounts[0].UserID), http.StatusTemporaryRedirect)
		return true
	}
	return false
}

func (g *GkillServerAPI) GetDevice() (string, error) {
	ctx := context.Background()
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		return "", err
	}

	var device *string
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			if device != nil {
				err = fmt.Errorf("invalid status. enable device count is not 1")
				return "", err
			}
			device = &serverConfig.Device
		}
	}
	if device == nil {
		err = fmt.Errorf("invalid status. enable device count is not 1")
		return "", err
	}
	g.device = *device
	return g.device, nil
}

func privateIPv4s() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, iface := range ifaces {
		// down / loopback は除外
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			ip4 := ip.To4()
			if ip4 == nil {
				continue // IPv4のみ
			}

			// 169.254.x.x (link-local) などは除外
			if ip4.IsLinkLocalUnicast() {
				continue
			}

			if isPrivateIPv4(ip4) {
				ips = append(ips, ip4)
			}
		}
	}
	return ips, nil
}

func isPrivateIPv4(ip net.IP) bool {
	// ip must be 4 bytes (To4済み想定)
	// RFC1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	switch {
	case ip[0] == 10:
		return true
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return true
	case ip[0] == 192 && ip[1] == 168:
		return true
	default:
		return false
	}
}

func globalIP(ctx context.Context) (net.IP, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	s := strings.TrimSpace(string(b))
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip response: %q", s)
	}
	return ip, nil
}
