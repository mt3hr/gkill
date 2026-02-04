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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
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
			slog.Log(ctx, gkill_log.Debug, "error", err)
		}
	}

	enableProxy := false
	proxyURL := ""
	body, err := getBody(u.URL, serverConfig.URLogTimeout, serverConfig.URLogUserAgent, enableProxy, proxyURL)
	if err != nil {
		err = fmt.Errorf("failed to get body: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", err)
	} else {
		// title
		if u.Title == "" {
			err := u.fillTitle(body)
			if err != nil {
				err = fmt.Errorf("failed to fill title to urlog.: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", err)
			}
		}

		// description
		if u.Description == "" {
			err := u.fillDescription(body)
			if err != nil {
				err = fmt.Errorf("failed to fill description to urlog.: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", err)
			}
		}

		// image
		if u.ThumbnailImage == "" {
			err := u.fillImage(body)
			if err != nil {
				err = fmt.Errorf("failed to fill image to urlog.: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", err)
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
	defer favicon.Close()
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

// ページURLからページのfaviconを取得する
func getFavicon(urlstr string) (image io.ReadCloser, err error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		err = fmt.Errorf("failed parse url %s: %w", urlstr, err)
		return nil, err
	}
	res, err := http.Get(`http://www.google.com/s2/favicons?domain=` + u.Hostname())
	if err != nil {
		err = fmt.Errorf("failed to get favicon by google api. hostname = %s: %w", u.Hostname(), err)
		return nil, err
	}
	image = res.Body
	return image, nil
}

// urlに対してhttpリクエストを飛ばし、bodyを取得する
func getBody(targeturl string, timeout time.Duration, useragent string, enableProxy bool, proxyURL string) (body []byte, err error) {
	var client *http.Client
	if enableProxy {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}

		client = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy),
			},
		}
	} else {
		client = &http.Client{Timeout: timeout}
	}
	req, err := http.NewRequest("GET", targeturl, nil)
	if err != nil {
		err = fmt.Errorf("failed to create http request.: %w", err)

		return nil, err
	}
	req.Header.Set("User-Agent", useragent)

	res, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to http get request: %w", err)
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// imageを取得してurlogに書き込む
func (u *URLog) fillImage(body []byte) error {
	// image
	imageBase64 := ""
	imgSrc, err := getImageOG(body)
	if err != nil {
		err = fmt.Errorf("failed to getImageOG: %w", err)
		// amazonの商品画像を読み込む
		var e error
		imgSrc, e = getAmazonImage(body)
		if imgSrc != nil {
			defer imgSrc.Close()
		}
		if e != nil {
			err = fmt.Errorf("error at get amazon image: %s: %w", e, err)
			return err
		}
	}
	defer imgSrc.Close()
	img, imgType, err := image.Decode(imgSrc)
	if err != nil {
		err = fmt.Errorf("failed to decodeImage: %w", err)
		return err
	}

	// 画像が大きすぎればリサイズする
	resizedImg := resizeImage(img, 220)

	// bufにエンコードする
	buf := bytes.NewBuffer([]byte{})
	switch imgType {
	case "jpeg", "jpg":
		err := jpeg.Encode(buf, resizedImg, &jpeg.Options{Quality: 100})
		if err != nil {
			err = fmt.Errorf("failed to image encode to %s: %w", imgType, err)
			return err
		}
	case "png":
		err := png.Encode(buf, resizedImg)
		if err != nil {
			err = fmt.Errorf("failed to image encode to %s: %w", imgType, err)
			return err
		}
	case "gif":
		err := gif.Encode(buf, resizedImg, nil)
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
	b, err := io.ReadAll(buf)
	if err != nil {
		// bufを読めないのはおかしい
		err = fmt.Errorf("failed to read buf: %w", err)
		if err != nil {
			panic(err)
		}
	} else {
		imageBase64 = base64.StdEncoding.EncodeToString(b)
	}
	u.ThumbnailImage = imageBase64
	return nil
}

// amazonのimageのURLを取得する
func getAmazonImage(body []byte) (io.ReadCloser, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		err = fmt.Errorf("error at get amazon image: %w", err)
		return nil, err
	}
	src := doc.Find("#landingImage").AttrOr("src", "")
	res, err := http.Get(src)
	if err != nil {
		err = fmt.Errorf("error at http get %s: %w", src, err)
		return nil, err
	}
	return res.Body, nil
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
	res, err := http.Get(imageURL)
	if err != nil {
		err = fmt.Errorf("failed to get image %s: %w", imageURL, err)
		return nil, err
	}
	image = res.Body
	return image, nil
}

var detector = chardet.NewHtmlDetector()

// 文字列をUTF8に統一する
func toUTF8(str []byte) (utf8str []byte, err error) {
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
