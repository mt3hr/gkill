package reps

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/safefetch"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/saintfish/chardet"
	"golang.org/x/image/draw"
)

type URLog struct {
	IsDeleted bool `json:"is_deleted"`

	ID string `json:"id"`

	RepName string `json:"rep_name"`

	RelatedTime time.Time `json:"related_time"`

	DataType string `json:"data_type"`

	CreateTime time.Time `json:"create_time"`

	CreateApp string `json:"create_app"`

	CreateDevice string `json:"create_device"`

	CreateUser string `json:"create_user"`

	UpdateTime time.Time `json:"update_time"`

	UpdateApp string `json:"update_app"`

	UpdateUser string `json:"update_user"`

	UpdateDevice string `json:"update_device"`

	URL string `json:"url"`

	Title string `json:"title"`

	Description string `json:"description"`

	FaviconImage string `json:"favicon_image"`

	ThumbnailImage string `json:"thumbnail_image"`
}

// FillURLogField .
// 0値なURLogの値を埋めます
func (u *URLog) FillURLogField(serverConfig *server_config.ServerConfig, applicationConfig *user_config.ApplicationConfig) error {
	ctx := context.Background()
	if u.URL == "" {
		err := fmt.Errorf("url value has not been set")
		return err
	}

	// id
	if u.ID == "" {
		u.ID = sqlite3impl.GenerateNewID()
	}

	// time
	if u.RelatedTime.IsZero() {
		u.RelatedTime = time.Now()
	}

	// favicon
	if u.FaviconImage == "" {
		err := u.fillFavicon()
		if err != nil {
			err = fmt.Errorf("failed to fill favicon: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	}

	enableProxy := false
	proxyURL := ""
	body, err := getBody(u.URL, serverConfig.URLogTimeout, serverConfig.URLogUserAgent, enableProxy, proxyURL)
	if err != nil {
		err = fmt.Errorf("failed to get body: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	} else {
		// title
		if u.Title == "" {
			err := u.fillTitle(body)
			if err != nil {
				err = fmt.Errorf("failed to fill title to urlog.: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			}
		}

		// description
		if u.Description == "" {
			err := u.fillDescription(body)
			if err != nil {
				err = fmt.Errorf("failed to fill description to urlog.: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			}
		}

		// image
		if u.ThumbnailImage == "" {
			err := u.fillImage(body)
			if err != nil {
				err = fmt.Errorf("failed to fill image to urlog.: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			}
		}
	}
	return nil
}

// faviconを取得してurlogに書き込む
func (u *URLog) fillFavicon() error {
	faviconBase64 := ""
	favicon, err := getFavicon(u.URL)
	if err != nil {
		err = fmt.Errorf("failed to getFavicon: %w", err)
		return err
	}
	defer func() {
		err := favicon.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	b, err := io.ReadAll(favicon)
	if err != nil {
		err = fmt.Errorf("failed to readFavicon: %w", err)
		return err
	}
	faviconBase64 = base64.StdEncoding.EncodeToString(b)
	u.FaviconImage = faviconBase64
	return nil
}

// titleを取得してurlogに書き込む
func (u *URLog) fillTitle(body []byte) error {
	title, err := getTitle(body)
	if err != nil {
		err = fmt.Errorf("failed to getTitle: %w", err)
		return err
	}
	u.Title = title
	return nil
}

// descriptionを取得してurlogに書き込む
func (u *URLog) fillDescription(body []byte) error {
	// description もしくは descriptionOG
	description, err := getDescriptionOG(body)
	if err != nil {
		description, err = getDescription(body)
		if err != nil {
			err = fmt.Errorf("failed to getDescription and getDescriptionOG: %w", err)
			return err
		}
	}
	u.Description = description
	return nil
}

// ページURLからページのfaviconを取得する。
// 取得先は固定ホスト google.com なのでSSRFではないが、無制限readを防ぐため上限付きで取る。
func getFavicon(urlstr string) (image io.ReadCloser, err error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		err = fmt.Errorf("failed parse url %s: %w", urlstr, err)
		return nil, err
	}
	b, err := safefetch.GetCapped(`https://www.google.com/s2/favicons?domain=`+u.Hostname(), 30*time.Second, "", false, safefetch.DefaultMaxImageBytes)
	if err != nil {
		err = fmt.Errorf("failed to get favicon by google api. hostname = %s: %w", u.Hostname(), err)
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// urlに対してhttpリクエストを飛ばし、bodyを取得する。
// 取得先は利用者が入力したURLなので SSRF 対策・サイズ上限つきの safefetch を通す。
// enableProxy=true のプロキシ経路は現状どこからも渡らない（互換のため残す）。
func getBody(targeturl string, timeout time.Duration, useragent string, enableProxy bool, proxyURL string) (body []byte, err error) {
	if enableProxy {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		client := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy),
			},
		}
		req, err := http.NewRequest("GET", targeturl, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create http request.: %w", err)
		}
		req.Header.Set("User-Agent", useragent)
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to http get request: %w", err)
		}
		defer func() {
			if cerr := res.Body.Close(); cerr != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", cerr)
			}
		}()
		return io.ReadAll(io.LimitReader(res.Body, safefetch.DefaultMaxBodyBytes+1))
	}
	return safefetch.GetCapped(targeturl, timeout, useragent, false, safefetch.DefaultMaxBodyBytes)
}

// imageを取得してurlogに書き込む
func (u *URLog) fillImage(body []byte) error {
	// image
	imgSrc, err := getImageOG(body)
	if err != nil {
		err = fmt.Errorf("failed to getImageOG: %w", err)
		// amazonの商品画像を読み込む
		var e error
		imgSrc, e = getAmazonImage(body)
		if imgSrc != nil {
			defer func() {
				err := imgSrc.Close()
				if err != nil {
					slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
				}
			}()
		}
		if e != nil {
			err = fmt.Errorf("error at get amazon image: %s: %w", e, err)
			return err
		}
	}
	defer func() {
		err := imgSrc.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	// imgSrc は safefetch で上限バイトまでに絞った画像なので全読みしても安全。
	// 復号前に DecodeConfig で寸法を検査し、画像爆弾（巨大寸法）を弾く。
	imgBytes, err := io.ReadAll(imgSrc)
	if err != nil {
		err = fmt.Errorf("failed to read image bytes: %w", err)
		return err
	}
	if err := safefetch.CheckImageDimensions(imgBytes, safefetch.DefaultMaxImagePixels); err != nil {
		return fmt.Errorf("failed to check image dimensions: %w", err)
	}
	img, imgType, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		err = fmt.Errorf("failed to decodeImage: %w", err)
		return err
	}

	// 画像が大きすぎればリサイズする
	resizedImg := resizeImage(img, 220)

	// bufにエンコードする
	var sb strings.Builder
	enc := base64.NewEncoder(base64.StdEncoding, &sb)
	switch imgType {
	case "jpeg", "jpg":
		err := jpeg.Encode(enc, resizedImg, &jpeg.Options{Quality: 100})
		if err != nil {
			err = fmt.Errorf("failed to image encode to %s: %w", imgType, err)
			return err
		}
	case "png":
		err := png.Encode(enc, resizedImg)
		if err != nil {
			err = fmt.Errorf("failed to image encode to %s: %w", imgType, err)
			return err
		}
	case "gif":
		err := gif.Encode(enc, resizedImg, nil)
		if err != nil {
			err = fmt.Errorf("failed to image encode to %s: %w", imgType, err)
			return err
		}
	default:
		err := fmt.Errorf("%s", imgType)
		err = fmt.Errorf("unknown image type %s: %w", imgType, err)
		return err
	}

	// base64にエンコードする
	u.ThumbnailImage = sb.String()
	return nil
}

// amazonのimageのURLを取得する
func getAmazonImage(body []byte) (io.ReadCloser, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		err = fmt.Errorf("error at get amazon image: %w", err)
		return nil, err
	}
	// src はフェッチしたHTMLの #landingImage 属性由来＝ブックマーク先ページが制御できるので
	// SSRF 対策つきの safefetch を通す。
	src := doc.Find("#landingImage").AttrOr("src", "")
	b, err := safefetch.GetCapped(src, 30*time.Second, "", false, safefetch.DefaultMaxImageBytes)
	if err != nil {
		err = fmt.Errorf("error at http get %s: %w", src, err)
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// 幅と高さで大きい方をmaxまで下げて、小さい方をその比率に合わせる
func resizeImage(img image.Image, max int) image.Image {
	var resizedImg draw.Image
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	if max < width || max < height {
		if width < height {
			h := max
			w := (width * h) / height
			height = h
			width = w
		} else {
			w := max
			h := (height * w) / width
			height = h
			width = w
		}
		resizedImg = image.NewRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	} else {
		return img
	}
	return resizedImg
}

// htmlBodyからページタイトルを取得する
func getTitle(body []byte) (title string, err error) {
	body, err = toUTF8(body)
	if err != nil {
		err = fmt.Errorf("failed to body to utf8: %w", err)
		return "", err
	}
	r := bytes.NewReader(body)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		err = fmt.Errorf("failed to goquery.NewDocumentFromReader: %w", err)
		return "", err
	}
	title = doc.Find("title").Text()
	return title, nil
}

// htmlBodyからページDescriptionの内容を取得する
func getDescription(body []byte) (description string, err error) {
	body, err = toUTF8(body)
	if err != nil {
		err = fmt.Errorf("failed to body to utf8: %w", err)
		return "", err
	}
	r := bytes.NewReader(body)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		err = fmt.Errorf("failed to goquery.NewDocumentFromReader: %w", err)
		return "", err
	}
	description = doc.Find(`meta[name="description"]`).AttrOr("content", "")
	if description == "" {
		err := fmt.Errorf("description not found from html body")
		return "", err
	}
	return description, nil
}

// htmlBodyからページDescriptionOGの内容を取得する:
func getDescriptionOG(body []byte) (descriptionOG string, err error) {
	body, err = toUTF8(body)
	if err != nil {
		err = fmt.Errorf("failed to body to utf8: %w", err)
		return "", err
	}
	r := bytes.NewReader(body)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		err = fmt.Errorf("failed to goquery.NewDocumentFromReader: %w", err)
		return "", err
	}
	descriptionOG = doc.Find(`meta[property="og:description"]`).AttrOr("content", "")
	if descriptionOG == "" {
		err := fmt.Errorf("descriptionOG not found from html body")
		return "", err
	}
	return descriptionOG, nil
}

// htmlBodyからImageOGの内容を取得する
func getImageOG(body []byte) (image io.ReadCloser, err error) {
	body, err = toUTF8(body)
	if err != nil {
		err = fmt.Errorf("failed to body to utf8: %w", err)
		return nil, err
	}
	r := bytes.NewReader(body)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		err = fmt.Errorf("failed to goquery.NewDocumentFromReader: %w", err)
		return nil, err
	}
	imageURL := doc.Find(`meta[property="og:image"]`).AttrOr("content", "")
	if imageURL == "" {
		return nil, fmt.Errorf("imageOG not found from html body")
	}
	// imageURL は og:image 由来＝ブックマーク先ページが制御できるので SSRF 対策つきで取る。
	b, err := safefetch.GetCapped(imageURL, 30*time.Second, "", false, safefetch.DefaultMaxImageBytes)
	if err != nil {
		err = fmt.Errorf("failed to get image %s: %w", imageURL, err)
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

var detector = chardet.NewHtmlDetector()

// 文字列をUTF8に統一する
func toUTF8(str []byte) (utf8str []byte, err error) {
	// 既に有効なUTF-8であればそのまま返す（chardetの誤検出による文字化けを防ぐ）
	if utf8.Valid(str) {
		return str, nil
	}

	detectorResult, err := detector.DetectBest([]byte(str))
	if err != nil {
		err = fmt.Errorf("failed to detect charset: %w", err)
		return nil, err
	}

	decoder := mahonia.NewDecoder(detectorResult.Charset)
	if decoder == nil {
		err := fmt.Errorf("not found %s decoder", detectorResult.Charset)
		return nil, err
	}
	s := string(str)
	utf8b := decoder.ConvertString(s)
	return []byte(utf8b), nil
}

// urlogImageColumnsSQL は FAVICON_IMAGE / THUMBNAIL_IMAGE の SELECT 句を返します。
//
// THUMBNAIL_IMAGE は base64 で埋め込まれており、実データでは1行あたり平均406KB・
// 最大10MBで、227行の合計が90MBあります。
// サムネイルを使わない呼び出しでは空文字を返す式に差し替えて、
// DBから読む段階で外します。
// 列の並びと数は変えないので、呼び出し側のScanはそのままで動きます。
//
// FAVICON_IMAGE は合計0.10MB・平均0.5KBしかないので常に取得します。
func urlogImageColumnsSQL(query *find.FindQuery) string {
	if query != nil && query.ExcludeURLogThumbnailImage {
		return "FAVICON_IMAGE,\n  '' AS THUMBNAIL_IMAGE"
	}
	return "FAVICON_IMAGE,\n  THUMBNAIL_IMAGE"
}
