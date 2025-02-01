package dvnf_cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mt3hr/gkill/src/app/gkill/dvnf"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

func init() {
	copyFs := copyCommand.Flags()
	copyFs.StringArrayVarP(&copyOpt.ignore, "ignore", "i", gkill_options.IDFIgnore, "コピー処理から除外するファイル名")
	copyFs.BoolVarP(&copyOpt.override, "override", "w", true, "コピー先ファイルに上書きする")
	copyFs.BoolVar(&copyOpt.fast, "fast", true, "最終更新時刻が更新された場合にのみコピーする")
	copyFs.BoolVarP(&copyOpt.file, "file", "f", false, "コピー先をファイルとする")
	copyFs.BoolVarP(&copyOpt.ext, "ext", "e", true, "ドットが含まれた場合拡張子として分割する")
	copyFs.BoolVar(&copyOpt.copyLastMod, "copy_lastmod", true, "ファイルの最終更新時刻情報をコピーする")
	copyFs.BoolVar(&copyOpt.robo, "robo", false, "移動元が存在しなかったらエラーを吐く")
	copyCommand.PreRun = func(cmd *cobra.Command, args []string) {
		copyOpt.src, copyOpt.target = args[0], args[1]
	}
}

var copyOpt = &struct {
	src         string
	target      string
	ignore      []string
	override    bool
	fast        bool
	file        bool
	ext         bool
	copyLastMod bool
	robo        bool // 有効な場合、srcでマッチするものがなかったらパニクる
}{}

func runCopy(_ *cobra.Command, _ []string) {
	var err error
	dvnfdir, childdir := splitDVNFPathnium(copyOpt.target)

	// createNewならcreateする
	opt := newDVNFOption(dvnfdir, copyOpt.ext)
	if rootOpt.createNew {
		dvnfdir, err = dvnf.CreateNewDVNF(opt, true)
		if err != nil {
			err = fmt.Errorf("error create new dvnf %s_%s: %w", opt.Device, opt.Directory, err)
			log.Fatal(err)
		}
	}

	// 最新のdvnfを取得する。なければ今のdvnfを。
	dvnfdir, err = dvnf.GetLatestDVNF(opt)
	if err != nil {
		err = fmt.Errorf("error at get latest dvnf %s_%s: %w", opt.Device, opt.Directory, err)
		log.Fatal(err)
	}
	if dvnfdir == "" {
		dvnfdir, err = dvnf.NewDVNF(opt)
		if err != nil {
			err = fmt.Errorf("error at new dvnf %s_%s: %w", opt.Device, opt.Directory, err)
			log.Fatal(err)
		}
	}

	// targetを決める
	target := filepath.Join(dvnfdir, childdir)
	target, err = filepath.Abs(target)
	if err != nil {
		err = fmt.Errorf("error at target to absolute path %s: %w", target, err)
		log.Fatal(err)
	}

	// srcにマッチするものをすべて取得する。globなど
	src := os.ExpandEnv(copyOpt.src)
	matches, err := glob(src)
	if err != nil && copyOpt.robo {
		err = fmt.Errorf("error at glob %s: %w", src, err)
		log.Fatal(err)
	}

	// srcが存在しなければ何もしない
	if len(matches) == 0 {
		return
	}

	// コピーする。親ディレクトリを作成しつつ
	for _, path := range matches {
		// 無視だったら無視
		ignore := false
		srcFileNameBase := filepath.Base(path)
		for _, i := range copyOpt.ignore {
			if i == srcFileNameBase {
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}

		// 最終更新時刻が同じで無視なら無視
		if copyOpt.fast {
			srcStat, err := os.Stat(path)
			if err != nil {
				err = fmt.Errorf("error at get stat %s: %w", path, err)
				log.Fatal(err)
			}

			var targetStat os.FileInfo
			if copyOpt.file {
				targetStat, err = os.Stat(target)
			} else {
				targetStat, err = os.Stat(filepath.Join(target, srcFileNameBase))
			}
			if err == nil {
				duration := srcStat.ModTime().Sub(targetStat.ModTime())
				if int64(0) == int64(duration) {
					continue
				}
			}
		}

		fmt.Printf("copy %s -> %s\n", path, target)
		err := copy(path, target, copyOpt.ignore)
		if err != nil {
			err = fmt.Errorf("error at copy from %s to %s: %w", path, target, err)
			log.Fatal(err)
		}
	}
}

func copy(src, target string, ignores []string) error {
	srcFile, err := os.Stat(src)
	if err != nil {
		err = fmt.Errorf("error at get stat %s: %w", src, err)
		return err
	}

	// 無視だったら無視
	ignore := false
	srcFileNameBase := filepath.Base(srcFile.Name())
	for _, i := range ignores {
		if i == srcFileNameBase {
			ignore = true
			break
		}
	}
	if ignore {
		return nil
	}

	// ディレクトリだったら再帰的に
	if srcFile.IsDir() {
		childFiles, err := os.ReadDir(src)
		if err != nil {
			err = fmt.Errorf("error at read directory %s: %w", src, err)
			return err
		}
		for _, childFile := range childFiles {
			childFilename := filepath.Join(src, filepath.Base(childFile.Name()))
			targetFilename := filepath.Join(target, srcFile.Name())
			err = copy(childFilename, targetFilename, ignores)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// ディレクトリでなければファイルなので、targetに名前を付ける
	targetFile := ""
	if copyOpt.file {
		targetFile = target
	} else {
		targetFile = filepath.Join(target, filepath.Base(srcFile.Name()))
	}

	// 上書きするならそのまま、しないのであればカッコをつけて衝突回避する
	if !copyOpt.override {
		// カッコのついていないファイル名。例えば「hogehoge (1).txt」なら「hogehoge.txt」。
		planeFileName := planeFileName(targetFile)
		ext := filepath.Ext(planeFileName)
		withoutExt := planeFileName[:len(planeFileName)-len(ext)]

		// ファイルが存在しない名前になるまでカッコ内の数字をインクリメントし続ける
		// targetFilenameは最終的な移動先ファイル名
		targetFile = planeFileName
		for count := 0; ; count++ {
			if _, err := os.Stat(targetFile); err != nil {
				break
			}
			// 初回は無視。count:=0としないほうがfor文としてきれいに収まる
			if count == 0 {
				continue
			}
			targetFile = os.Expand("${name} (${count})${ext}", func(str string) string {
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
	}

	// 親ディレクトリが存在しなければ作る
	parentDir := filepath.Dir(targetFile)
	if parentDir != "" {
		_, err := os.Stat(parentDir)
		if os.IsNotExist(err) {
			err = os.MkdirAll(parentDir, os.ModePerm)
			if err != nil {
				err = fmt.Errorf("error at create directory %s: %w", parentDir, err)
				return err
			}
		}
	}
	err = copyFile(src, targetFile)
	return err
}

func copyFile(src, target string) error {
	// ファイルの内容をコピーする
	srcFile, err := os.OpenFile(src, os.O_RDONLY, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", src, err)
		return err
	}
	defer srcFile.Close()
	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at open file %s: %w", target, err)
		return err
	}
	defer targetFile.Close()
	_, err = io.Copy(targetFile, srcFile)
	if err != nil {
		err = fmt.Errorf("error at copy data from %s to %s: %w", src, target, err)
		return err
	}
	targetFile.Close() // Closeしてから出ないとchtimesが適用されないケースがあったため

	// 最終更新時刻をコピー
	if copyOpt.copyLastMod {
		srcInfo, err := os.Stat(src)
		if err != nil {
			err = fmt.Errorf("error at get stat %s: %w", src, err)
			return err
		}
		os.Chtimes(target, srcInfo.ModTime(), srcInfo.ModTime())
	}
	return err
}
