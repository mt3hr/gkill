package dvnf_cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mt3hr/gkill/src/app/gkill/dvnf"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

func init() {
	moveFs := moveCommand.Flags()
	moveFs.StringArrayVarP(&moveOpt.ignore, "ignore", "i", gkill_options.IDFIgnore, "移動処理から除外するファイル名")
	moveFs.BoolVarP(&moveOpt.deleteDirectory, "delete_directory", "d", false, "移動後、中身のない移動元ディレクトリを削除する")
	moveFs.BoolVarP(&moveOpt.override, "override", "w", false, "移動先ファイルに上書きする")
	moveFs.BoolVarP(&moveOpt.file, "file", "f", false, "移動先をファイルとする")
	moveFs.BoolVarP(&moveOpt.ext, "ext", "e", true, "ドットが含まれた場合拡張子として分割する")
	moveFs.BoolVar(&moveOpt.robo, "robo", false, "移動元が存在しなかったらエラーを吐く")
	moveCommand.PreRun = func(cmd *cobra.Command, args []string) {
		moveOpt.src, moveOpt.target = args[0], args[1]
	}
}

var moveOpt = struct {
	src             string
	target          string
	ignore          []string
	deleteDirectory bool
	override        bool
	file            bool
	ext             bool
	robo            bool // 有効な場合、srcでマッチするものがなかったらパニクる
}{}

func runMove(_ *cobra.Command, _ []string) {
	var err error
	dvnfdir, childdir := splitDVNFPathnium(moveOpt.target)

	// createNewならcreateする
	opt := newDVNFOption(dvnfdir, moveOpt.ext)
	if rootOpt.createNew {
		_, err = dvnf.CreateNewDVNF(opt, true)
		if err != nil {
			err = fmt.Errorf("error at create new dvnf %s_%s: %w", opt.Device, opt.Directory, err)
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
	src := os.ExpandEnv(moveOpt.src)
	matches, err := glob(src)
	if err != nil && moveOpt.robo {
		err = fmt.Errorf("error at glob %s: %w", src, err)
		log.Fatal(err)
	}

	// srcが存在しなければ何もしない
	if len(matches) == 0 {
		return
	}

	// 移動する。親ディレクトリを作成しつつ
	for _, path := range matches {
		// 無視だったら無視
		ignore := false
		srcFileNameBase := filepath.Base(path)
		for _, i := range moveOpt.ignore {
			if i == srcFileNameBase {
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}

		fmt.Printf("move %s -> %s\n", path, target)
		err := move(path, target, moveOpt.ignore)
		if err != nil {
			err = fmt.Errorf("error at move from %s to %s: %w", path, target, err)
			log.Fatal(err)
		}
	}
}

func move(src, target string, ignores []string) error {
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
			err = move(childFilename, targetFilename, ignores)
			if err != nil {
				return err
			}
		}
		// DeleteDirectoryが有効なら中身が無いか確認して削除
		if moveOpt.deleteDirectory {
			childFiles, err := os.ReadDir(src)
			if err != nil {
				err = fmt.Errorf("error at read dir %s: %w", src, err)
				return err
			}
			if len(childFiles) == 0 {
				err := os.Remove(src)
				if err != nil {
					err = fmt.Errorf("error at delete %s: %w", src, err)
					return err
				}
			}
		}
		return nil
	}

	// ディレクトリでなければファイルなので、targetに名前を付ける
	targetFile := ""
	if moveOpt.file {
		targetFile = target
	} else {
		targetFile = filepath.Join(target, filepath.Base(srcFile.Name()))
	}

	// 上書きするならそのまま、しないのであればカッコをつけて衝突回避する
	if !moveOpt.override {
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
	err = os.Rename(src, targetFile)
	if err != nil {
		// 別ドライブへの移動だとなんか成功しません。
		// ので、移動先にあるはずのファイルが存在しなかった場合はコピーして削除で代用します。
		err = copyFile(src, targetFile)
		if err != nil {
			err = fmt.Errorf("error at copy file from %s to %s: %w", src, targetFile, err)
			return err
		}
		err = os.Remove(src)
		if err != nil {
			err = fmt.Errorf("error at remove %s: %w", src, err)
			return err
		}
	}
	return nil
}

// ファイル名に(n)がついていたら除去して返します。
// hogehoge.txt (1) (1) (1)とかにならないように。
// Windowsのファイル重複時Suffixに対応しています。？
func planeFileName(filename string) (fixedfilename string) {
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
