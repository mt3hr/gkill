package dvnf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// Option .
// dvnfを取得、作成するときのオプション。
type Option struct {
	Directory  string
	Name       string
	Device     string
	TimeLength int
	Extension  string
}

// GetOrCreateLatestDVNFDir .
// 最新のDVNFディレクトリを取得します。なければ作成してから取得します。
func GetOrCreateLatestDVNFDir(opt *Option) (string, error) {
	dvnf, err := GetLatestDVNF(opt)
	// すでに存在する場合はディレクトリであるならば返す
	if err == nil && dvnf != "" {
		file, err := os.Stat(dvnf)
		if err != nil {
			err = fmt.Errorf("error at get %s stat: %w", dvnf, err)
			return "", err
		}
		if file.IsDir() {
			err = fmt.Errorf("%s is not directory", dvnf)
			return dvnf, nil
		}
	}

	// そうでなければ新たにディレクトリを作って返す
	return CreateNewDVNF(opt, true)
}

// GetOrCreateLatestDVNFFile .
// 最新のDVNFファイルを取得します。なければ作成してから取得します。
func GetOrCreateLatestDVNFFile(opt *Option) (string, error) {
	dvnf, err := GetLatestDVNF(opt)
	// すでに存在する場合はディレクトリでなければ返す
	if err == nil && dvnf != "" {
		file, err := os.Stat(dvnf)
		if err != nil {
			err = fmt.Errorf("error at get %s stat: %w", dvnf, err)
			return "", err
		}
		if !file.IsDir() {
			return dvnf, nil
		}
	}

	// そうでなければ新たにファイルを作って返す
	return CreateNewDVNF(opt, false)
}

// GetLatestDVNF .
// 存在する最新のDVNFを取得します。
// 当てはまるものが存在しない場合は空文字を返します。
func GetLatestDVNF(opt *Option) (dvnf string, err error) {
	dvnfs, err := GetDVNFs(opt)
	if err != nil {
		err = fmt.Errorf("error at get dvnfs %s_%s: %w", opt.Device, opt.Directory, err)
		return "", err
	}
	if len(dvnfs) == 0 {
		return "", nil
	}
	SortDVNFs(dvnfs)
	return dvnfs[0], nil
}

// CreateNewDVNF .
// dvnfを新たに作成します。
func CreateNewDVNF(opt *Option, isDir bool) (string, error) {
	dvnf, err := NewDVNF(opt)
	if err != nil {
		err = fmt.Errorf("error at new dvnf %s_%s: %w", opt.Device, opt.Directory, err)
		return "", err
	}
	// ディレクトリだったら作って返す
	if isDir {
		err := os.MkdirAll(dvnf, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create directory %s: %w", dvnf, err)
			return "", err
		}
		return dvnf, nil
	}

	// ファイルだったら親ディレクトリを作ってから返す
	parentDir := filepath.Dir(dvnf)
	err = os.MkdirAll(parentDir, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at create directory %s: %w", parentDir, err)
		return "", err
	}
	file, err := os.Create(dvnf)
	if err != nil {
		err = fmt.Errorf("error at create file %s: %w", dvnf, err)
		return "", err
	}
	file.Close()
	return dvnf, nil
}

// SortDVNFs .
// 日時降順でソートします。
func SortDVNFs(dvnfs []string) {
	sort.Sort(sort.Reverse(sort.StringSlice(dvnfs)))
}

// GetDVNFs .
// マッチするDVNFを取得します。
// マッチするものが存在しなかった場合は空のスライスを返します。
// ソートはされません。必要であればSortDVNFsに渡してください。
func GetDVNFs(opt *Option) ([]string, error) {
	err := isValidOption(opt)
	if err != nil {
		return nil, err
	}

	// dvnfの正規雨表現パターンを作成する
	pattern := ""
	if opt.Name != "" {
		if pattern != "" {
			pattern += "_"
		}
		pattern += opt.Name
	}
	if opt.Device != "" {
		if pattern != "" {
			pattern += "_"
		}
		pattern += opt.Device
	}
	if opt.TimeLength != 0 {
		if pattern != "" {
			pattern += "_"
		}
		pattern += (`\d{` + strconv.Itoa(opt.TimeLength) + "}")
	}
	if opt.Extension != "" {
		pattern += opt.Extension
	}
	pattern = "^" + pattern + "$"

	regex, err := regexp.Compile(pattern)
	if err != nil {
		err = fmt.Errorf("error at compile regexp %s: %w", pattern, err)
		return nil, err
	}

	// 親ディレクトリが存在しなかったら無いので
	dir, err := os.Stat(opt.Directory)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		err = fmt.Errorf("error at get %s stat: %w", opt.Directory, err)
		return nil, err
	}
	if !dir.IsDir() {
		return nil, fmt.Errorf("%s is directory. want not directory file", opt.Directory)
	}

	// ディレクトリ内でパターンに一致するものを取得する
	files, err := os.ReadDir(opt.Directory)
	if err != nil {
		err = fmt.Errorf("error at read directory %s: %w", opt.Directory, err)
		return nil, err
	}
	matchFiles := []string{}
	for _, file := range files {
		if regex.MatchString(file.Name()) {
			matchFiles = append(matchFiles, file.Name())
		}
	}

	// なければそのまま返す
	if len(matchFiles) == 0 {
		return []string{}, nil
	}

	// あればDirとつなげて返す
	dvnfs := []string{}
	for _, matchFile := range matchFiles {
		dvnfFile := filepath.Join(opt.Directory, matchFile)
		dvnfs = append(dvnfs, dvnfFile)
	}
	return dvnfs, nil
}

// NewDVNF .
// 現在時刻のDVNFを取得します。
// ファイルやディレクトリの作成はされません。
func NewDVNF(opt *Option) (string, error) {
	err := isValidOption(opt)
	if err != nil {
		return "", err
	}

	dvnf := ""
	if opt.Name != "" {
		if dvnf != "" {
			dvnf += "_"
		}
		dvnf += opt.Name
	}
	if opt.Device != "" {
		if dvnf != "" {
			dvnf += "_"
		}
		dvnf += opt.Device
	}
	if opt.TimeLength != 0 {
		if dvnf != "" {
			dvnf += "_"
		}
	}
	switch opt.TimeLength {
	case 8:
		dvnf += time.Now().Format("20060102")
	case 6:
		dvnf += time.Now().Format("200601")
	case 4:
		dvnf += time.Now().Format("2006")
	case 0:
	}
	if opt.Extension != "" {
		dvnf += opt.Extension
	}
	if opt.Directory != "" {
		dvnf = filepath.Join(opt.Directory, dvnf)
	}

	return dvnf, nil
}

// オプションの数値が正しいかどうかを判断します。
// 正しくない場合はErrorを返します。
func isValidOption(opt *Option) error {
	switch opt.TimeLength {
	case 0, 4, 6, 8:
	default:
		err := fmt.Errorf("invalid timelength of dvnf option.fix to 0, 4, 6, 8")
		return err
	}

	if (opt.Directory == "" && opt.Name == "") &&
		(opt.Device == "" && opt.TimeLength == 0) &&
		opt.Extension == "" {
		err := fmt.Errorf("dvnf option is empty")
		return err
	}
	return nil
}
