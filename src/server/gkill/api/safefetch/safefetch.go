// Package safefetch は、利用者が入力したURLや、そのページのHTMLが指す
// og:image / #landingImage といった攻撃者制御になりうるURLを、
// SSRF・無制限read・画像爆弾から守って取得するための共通ヘルパです。
//
// 以前は gkill_server_api/utils.go の中にブックマークレット画像取得専用として
// 閉じていました。URLog のタイトル/画像取得（dao/reps/ur_log.go）にも同じ防御が
// 要るので、共有パッケージへ切り出しています。
package safefetch

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	// image.DecodeConfig / image.Decode 用にデコーダを登録する。
	// 呼び出し側のプロセスに依存せず、このパッケージ単体で寸法検査できるようにする。
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

const (
	// DefaultMaxBodyBytes は HTML 本文取得の上限（10MB）。
	DefaultMaxBodyBytes int64 = 10 * 1024 * 1024
	// DefaultMaxImageBytes は画像取得の上限（10MB）。
	DefaultMaxImageBytes int64 = 10 * 1024 * 1024
	// DefaultMaxImagePixels は画像爆弾対策の総ピクセル上限（8192×8192）。
	DefaultMaxImagePixels int64 = 8192 * 8192
)

// IsDisallowedFetchIP は取得先にできないIPかを判定します。
// allowPrivate=false: loopback / プライベート / リンクローカル / マルチキャスト / 未指定を全拒否。
// allowPrivate=true: loopback / プライベートは許可するが、リンクローカル（169.254.169.254 の
// クラウドメタデータ含む）/ マルチキャスト / 未指定は常に拒否する。
func IsDisallowedFetchIP(ip net.IP, allowPrivate bool) bool {
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if allowPrivate {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

// newClient は Dialer.Control で実際の接続先IPを検証するHTTPクライアントを返します。
// これにより DNSリバインディングやリダイレクトで内部アドレスへ誘導されても接続段階で拒否されます。
func newClient(timeout time.Duration, allowPrivate bool) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
				Control: func(_ string, address string, _ syscall.RawConn) error {
					host, _, err := net.SplitHostPort(address)
					if err != nil {
						return err
					}
					if IsDisallowedFetchIP(net.ParseIP(host), allowPrivate) {
						return fmt.Errorf("blocked request to disallowed address: %s", address)
					}
					return nil
				},
			}).DialContext,
		},
	}
}

// GetCapped は http/https のURLを取得し、maxBytes で本文を打ち切ります。
// スキーム検査・接続先IP検査・タイムアウトを適用します。
func GetCapped(urlString string, timeout time.Duration, userAgent string, allowPrivate bool, maxBytes int64) ([]byte, error) {
	parsed, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("error at parse url %s: %w", urlString, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme %q at http get %s", parsed.Scheme, urlString)
	}

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return nil, fmt.Errorf("error at new http get request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	req.Header.Set("Referer", urlString)

	res, err := newClient(timeout, allowPrivate).Do(req)
	if err != nil {
		return nil, fmt.Errorf("error at http get %s: %w", urlString, err)
	}
	defer func() { _ = res.Body.Close() }()

	// gzip はトランスポートが透過展開するので、これは展開後バイトへの上限になる。
	b, err := io.ReadAll(io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("error at read all body %s: %w", urlString, err)
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("response body too large at http get %s", urlString)
	}
	return b, nil
}

// GetBase64Data は http/https のURLを SSRF 安全に取得し、base64 で返します。
// ブックマークレットの画像/favicon 取得用（private 拒否・10MB上限）。
func GetBase64Data(urlString string) (string, error) {
	b, err := GetCapped(urlString, 30*time.Second, "", false, DefaultMaxBodyBytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// CheckImageDimensions は image.DecodeConfig で復号前に寸法を検査し、
// 総ピクセルが maxPixels を超える画像（画像爆弾）を拒否します。
func CheckImageDimensions(data []byte, maxPixels int64) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("error at decode image config: %w", err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return fmt.Errorf("image too large: %dx%d exceeds %d pixels", cfg.Width, cfg.Height, maxPixels)
	}
	return nil
}
