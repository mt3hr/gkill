package dvnf_cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/app/gkill/dvnf"
	"github.com/spf13/cobra"
)

func init() {
	getFs := getCommand.Flags()
	getFs.BoolVarP(&getOpt.all, "all", "a", false, "有効にした場合、最新だけでなくマッチするすべてのdvnfを取得します")
	getFs.BoolVarP(&getOpt.createSubDir, "create_sub_directory", "s", false, "例えばhoge/fuga/piyoを渡されたときに、fuga/piyoも作成します。")
	getFs.BoolVarP(&getOpt.ext, "ext", "e", true, "ドットが含まれた場合拡張子として分割する")
	getCommand.PreRun = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			getOpt.dvnfName = ""
		} else if args[0] != "" {
			getOpt.dvnfName = args[0]
		} else {
			pipeArg, err := pipeInFormatted()
			if err != nil {
				log.Fatal(err)
			}
			pipeArg = removeNewlineCodes(pipeArg)
			if pipeArg != "" {
				getOpt.dvnfName = pipeArg
			}
		}
	}
}

var getOpt = struct {
	dvnfName     string
	all          bool
	createSubDir bool
	ext          bool
}{}

func printDVNFRootDir(cfg *Config) {
	fmt.Println(cfg.Directory)
}

// childdirはなければ空文字です
func splitDVNFPathnium(dvnfPathnium string) (dvnfdir, childdir string) {
	dvnfPathnium = filepath.Clean(dvnfPathnium)
	list := strings.Split(dvnfPathnium, string(filepath.Separator))
	dvnfdir = list[0]
	if len(list) >= 2 {
		childdir = filepath.Join(list[1:]...)
	}
	return dvnfdir, childdir
}

// dvnfdirをnameとしたotpを作成します。
// それ以外の情報はcfgを参照します。
func newDVNFOption(dvnfdir string, useExt bool) *dvnf.Option {
	if !useExt {
		return &dvnf.Option{
			Directory:  config.Directory,
			Name:       dvnfdir,
			Device:     config.Device,
			TimeLength: config.TimeLength,
		}
	}

	split := filepath.SplitList(dvnfdir)
	zero := split[0]

	ext := filepath.Ext(zero)
	withoutExt := zero[:len(zero)-len(ext)]

	split[0] = withoutExt
	dvnfdir = filepath.Join(split...)
	return &dvnf.Option{
		Directory:  config.Directory,
		Name:       dvnfdir,
		Device:     config.Device,
		TimeLength: config.TimeLength,
		Extension:  ext,
	}
}

func runGet(_ *cobra.Command, _ []string) {
	var err error

	// 引数がなければrootを返す
	if getOpt.dvnfName == "" {
		printDVNFRootDir(config)
		return
	}

	// 引数があればExpandDVNF
	dvnfdir, childdir := splitDVNFPathnium(getOpt.dvnfName)

	// childがあろうとなかろうと、createNewならば作る
	opt := newDVNFOption(dvnfdir, getOpt.ext)
	if rootOpt.createNew {
		dvnfdir, err = dvnf.CreateNewDVNF(opt, true)
		if err != nil {
			err = fmt.Errorf("failed to create new dvnf %s_%s: %w", opt.Device, opt.Directory, err)
			log.Fatal(err)
		}
	}

	// optなdvnfがそもそも1つ以上存在するかどうか
	dvnfdir, err = dvnf.GetLatestDVNF(opt)
	if err != nil {
		err = fmt.Errorf("failed to get latest dvnf %s_%s: %w", opt.Device, opt.Directory, err)
		log.Fatal(err)
	}
	exist := dvnfdir != ""
	// まだ存在しなかったら現在のdvnfをはめる
	if !exist {
		dvnfdir, err = dvnf.NewDVNF(opt)
		if err != nil {
			err = fmt.Errorf("failed to new dvnf %s_%s: %w", opt.Device, opt.Directory, err)
			log.Fatal(err)
		}
	}
	// autoCreateなら作る
	if rootOpt.autoCreate && !exist {
		err = os.MkdirAll(dvnfdir, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("failed to create directory %s: %w", dvnfdir, err)
			log.Fatal(err)
		}
	}

	if !getOpt.all {
		// allじゃなければさっきとった最新のものをprintln
		// createSubDirならつくってから
		dir := filepath.Join(dvnfdir, childdir)
		if getOpt.createSubDir {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				err = fmt.Errorf("failed to create directory %s: %w", dir, err)
				log.Fatal(err)
			}
		}
		fmt.Println(dir)
		return
	}
	dvnfs, err := dvnf.GetDVNFs(opt)
	if err != nil {
		err = fmt.Errorf("failed to get dvnfs %s_%s: %w", opt.Device, opt.Directory, err)
		log.Fatal(err)
	}
	// 存在しなければdvnfdirをprintlnして終了
	if len(dvnfs) == 0 {
		dir := filepath.Join(dvnfdir, childdir)
		fmt.Println(dir)
		return
	}
	dvnf.SortDVNFs(dvnfs)
	// あるならば全部printlnして終了
	for _, dvnfdir := range dvnfs {
		dir := filepath.Join(dvnfdir, childdir)
		if getOpt.createSubDir {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				err = fmt.Errorf("failed to create directory %s: %w", dir, err)
				log.Fatal(err)
			}
		}
		fmt.Println(dir)
	}
	return
}
