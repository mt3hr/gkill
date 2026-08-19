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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/twpayne/go-gpx"
)

func GenerateNewID() string {
	return uuid.New().String()
}

// reForbiddenChars はWindows/Android不正文字の正規表現です。
// クライアント側 sanitize_filename と同じ文字セットです。
var reForbiddenChars = regexp.MustCompile(`[\\/:*?"<>|{}]`)

// reControlChars は制御文字 (0x00-0x1f, 0x7f) の正規表現です。
var reControlChars = regexp.MustCompile("[\x00-\x1f\x7f]")

// sanitizeFilename はOS不正文字・制御文字を除去してファイル名を安全にします。
// クライアント側の sanitize_filename と同じロジックです。
func sanitizeFilename(name string) string {
	name = reForbiddenChars.ReplaceAllString(name, "")
	name = reControlChars.ReplaceAllString(name, "")
	name = strings.TrimSpace(name)
	if name == "" {
		return "file"
	}
	return name
}

func (g *GkillServerAPI) resolveFileName(repDir string, filename string, behavior req_res.FileUploadConflictBehavior) (string, error) {
	// OS不正文字・制御文字を除去する (クライアント側 sanitize_filename と同じ処理)
	filename = sanitizeFilename(filename)
	// パストラバーサル対策: repDir外へのアクセスを禁止する
	fullFilename, ok := reps.SecureJoin(repDir, filename)
	if !ok {
		return "", fmt.Errorf("invalid filename: path traversal detected")
	}
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
	repTypeFileNameMap["mirekyou"] = "MiReKyou.db"

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
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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

// maxHTTPGetBodyBytes は httpGetBase64Data が取得するレスポンスボディの上限サイズです。
const maxHTTPGetBodyBytes = 10 * 1024 * 1024

// isDisallowedFetchIP はSSRF対策として、ユーザ指定URLの取得先にできないIPか判定します。
// loopback・プライベート・リンクローカル・マルチキャスト・未指定アドレスを拒否します。
func isDisallowedFetchIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast()
}

// ssrfSafeHTTPClient はユーザ指定URLの取得に使うHTTPクライアントです。
// Dialer.Controlで実際の接続先IPを検証するため、DNSリバインディングやリダイレクトで
// 内部アドレスへ誘導されても接続段階で拒否されます。
var ssrfSafeHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: func(network, address string, c syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				if isDisallowedFetchIP(net.ParseIP(host)) {
					return fmt.Errorf("blocked request to disallowed address: %s", address)
				}
				return nil
			},
		}).DialContext,
	},
}

func httpGetBase64Data(urlString string) (string, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		err = fmt.Errorf("error at parse url %s: %w", urlString, err)
		return "", err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		err = fmt.Errorf("unsupported url scheme %q at http get %s", parsedURL.Scheme, urlString)
		return "", err
	}

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		err = fmt.Errorf("error at new http get request: %w", err)
		return "", err
	}
	req.Header.Set("Referer", urlString)

	res, err := ssrfSafeHTTPClient.Do(req)
	if err != nil {
		err = fmt.Errorf("error at http get %s: %w", urlString, err)
		return "", err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	b, err := io.ReadAll(io.LimitReader(res.Body, maxHTTPGetBodyBytes+1))
	if err != nil {
		err = fmt.Errorf("error at read all body %s: %w", urlString, err)
		return "", err
	}
	if len(b) > maxHTTPGetBodyBytes {
		err = fmt.Errorf("response body too large at http get %s", urlString)
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

	// 静的アセットはリダイレクトしない。
	// ビルド成果物は基本 /assets/ 配下 (content hash付き) に出るので prefix 判定で拾えるが、
	// Sec-Fetch-* ヘッダを送らない古いWebView経由だと上の早期returnを通らずここまで来るため、
	// 拡張子でも拾えるようにしている。
	p := r.URL.Path
	if strings.HasPrefix(p, "/assets/") ||
		strings.HasSuffix(p, ".js") ||
		strings.HasSuffix(p, ".css") ||
		strings.HasSuffix(p, ".map") ||
		strings.HasSuffix(p, ".png") ||
		strings.HasSuffix(p, ".webp") ||
		strings.HasSuffix(p, ".avif") ||
		strings.HasSuffix(p, ".svg") ||
		strings.HasSuffix(p, ".ico") ||
		strings.HasSuffix(p, ".woff2") ||
		strings.HasSuffix(p, ".webmanifest") {
		return false
	}

	// この関数は bool を返すだけでレスポンスを組み立てないので、
	// GkillError を作っても載せる先が無い。エラーコードはログに残す
	accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get all account config: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err), "error_code", message.GetAllAccountConfigError)
		return false
	}

	if len(accounts) == 1 {
		if accounts[0].UserID != "admin" || accounts[0].PasswordHash != nil || accounts[0].PasswordResetToken == nil {
			return false
		}

		// リセットトークンは admin を丸ごと取れてしまう秘密なので、
		// 同一マシンから直接来たリクエストにしか載せない。
		// 外から見えている状態だと `curl -sI http://host:9999/` だけで奪われてしまう。
		// ループバック以外からの初回セットアップは、起動時に標準出力へ出しているURLを使う
		// (PrintStartedMessage を参照)。
		if !isTrustedLocalRequest(r) {
			return false
		}

		http.Redirect(w, r, fmt.Sprintf("/register_first_account?reset_token=%s", url.QueryEscape(*accounts[0].PasswordResetToken)), http.StatusTemporaryRedirect)
		return true
	}
	return false
}

// GetDevice はこのプロセスが動いているデバイス名を返します。
//
// 結果は1回だけ求めてキャッシュします。取得元の GetAllServerConfigs は
// SERVER_CONFIG への相関サブクエリを18本使う重いSQLで、毎回
// 約250行のSQL文字列を組み立てて PrepareContext している。
// これが認証付きAPI1本につき最低3回、/files/ の画像配信では1枚ごとに走っていた。
//
// 設定更新時は handle_update_server_configs がサーバを落として
// GkillServerAPI ごと作り直すので、キャッシュの明示的な無効化は要らない。
//
// 以前は結果を g.device に代入していたが、直後に読むだけで他から参照されておらず、
// かつ全HTTPハンドラのgoroutineから排他無しで書かれていた(データ競合)。
func (g *GkillServerAPI) GetDevice() (string, error) {
	g.deviceOnce.Do(func() {
		g.device, g.deviceErr = g.findEnabledDevice()
	})
	return g.device, g.deviceErr
}

func (g *GkillServerAPI) findEnabledDevice() (string, error) {
	ctx := context.Background()
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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
	return *device, nil
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

// withUserContentSecurityHeaders は、利用者のファイルをそのまま返す経路
// (/files/ と /zip_cache/) にセキュリティヘッダを付ける。
//
// これらの経路は取り込んだファイルやZIPの展開物を、拡張子から決めた
// Content-Type で同一オリジンから配信する。展開物に拡張子の許可リストは無いので、
// .html や .svg が含まれていればブラウザはそれをHTMLとして解釈する。
// セッションクッキーはクライアント側のJSが document.cookie で書いており
// HttpOnly を付けられないため、同一オリジンでスクリプトが動くと
// そのまま読み出せてしまう。
//
//   - nosniff: 拡張子から決めた Content-Type をブラウザが推測で上書きしないようにする
//   - CSP sandbox: allow-scripts を付けていないので、これらの経路から返った
//     HTML/SVG の中のスクリプトは実行されない。
//     sandbox はドキュメントとして読み込まれたときにだけ効くので、
//     <img> や <video> のサブリソースとしての表示には影響しない。
//
// 例外として .pdf には sandbox を付けない。sandbox 下のドキュメントは
// opaque origin になり、Chrome が内蔵PDFビューワを無効化して表示ではなく
// ダウンロードに落ちるため。.pdf は nosniff と組み合わせて Content-Type が
// application/pdf に固定されるので、HTML/SVG と違い同一オリジンで
// スクリプトが動く経路にはならない（中身がHTMLでもPDFのパース失敗になるだけ）。
func withUserContentSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// r.URL.Path はデコード済みなので %2Epdf のような表記もここで .pdf に揃う
		if !strings.HasSuffix(strings.ToLower(r.URL.Path), ".pdf") {
			w.Header().Set("Content-Security-Policy", "sandbox")
		}
		next(w, r)
	}
}
