package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	_ "time/tzdata"

	"net/http"
	_ "net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/app/gkill/api"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	dvnf_cmd "github.com/mt3hr/gkill/src/app/gkill/dvnf/cmd"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
	"github.com/spf13/cobra"
)

var (
	AppName = "gkill_server"

	gkillServerAPI *api.GkillServerAPI

	IDFCmd = &cobra.Command{
		Use: "idf",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Usage()
				return
			}

			targetDirs := args

			autoIDF := false
			router := mux.NewRouter()
			idfIgnore := gkill_options.IDFIgnore

			for _, filenamePattern := range targetDirs {
				filenamePattern = os.ExpandEnv(filenamePattern)
				matchFiles, _ := zglob.Glob(filenamePattern)
				for _, filename := range matchFiles {
					if _, err := os.Stat(filename); os.IsNotExist(err) {
						fmt.Printf("Directory not found. skip idf: %s\n", filename)
						continue
					}
					parentDir := filepath.Join(filename, ".gkill")
					err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
						fmt.Printf("%s\n", err)
						fmt.Printf("skip idf: %s\n", filename)
						continue
					}
					idDBFilename := filepath.Join(parentDir, "gkill_id.db")
					idfKyouRep, err := reps.NewIDFDirRep(context.TODO(), filename, idDBFilename, router, &autoIDF, &idfIgnore, nil)
					if err != nil {
						err = fmt.Errorf("error at new idf dir rep: %w", err)
						gkill_log.Debug.Println(err.Error())
						fmt.Printf("skip idf: %s\n", filename)
						continue
					}
					defer idfKyouRep.Close(context.TODO())
					idfKyouRep.IDF(context.TODO())
				}
			}
		},
	}
	DVNFCmd = dvnf_cmd.DVNFCmd

	VersionCommand = &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			version, err := GetVersion()
			if err != nil {
				println(err.Error())
				return
			}
			fmt.Printf("%s: %s\n", AppName, version)
		},
	}
)

func init() {
	if os.Getenv("HOME") == "" {
		os.Setenv("HOME", os.Getenv("HOMEPATH"))
	}
	fixTimezone()

	done := threads.AllocateThread()
	defer done()
	go func() {
		http.ListenAndServe("localhost:6060", nil) // pprof用
	}()

	IDFCmd.PersistentFlags().StringArrayVarP(&gkill_options.IDFIgnore, "ignore", "i", gkill_options.IDFIgnore, "ignore files")
}

func InitGkillOptions() {
	os.Setenv("GKILL_HOME", filepath.Clean(os.ExpandEnv(gkill_options.GkillHomeDir)))
	gkill_options.LibDir = fmt.Sprintf("%s/lib/base_directory", gkill_options.GkillHomeDir)
	gkill_options.CacheDir = fmt.Sprintf("%s/caches", gkill_options.GkillHomeDir)
	gkill_options.LogDir = fmt.Sprintf("%s/logs", gkill_options.GkillHomeDir)
	gkill_options.ConfigDir = fmt.Sprintf("%s/configs", gkill_options.GkillHomeDir)
	gkill_options.TLSCertFileDefault = fmt.Sprintf("%s/tls/cert.cer", gkill_options.GkillHomeDir)
	gkill_options.TLSKeyFileDefault = fmt.Sprintf("%s/tls/key.pem", gkill_options.GkillHomeDir)
	gkill_options.DataDirectoryDefault = fmt.Sprintf("%s/datas", gkill_options.GkillHomeDir)
}

func fixTimezone() {
	if runtime.GOOS == "android" {
		out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
		if err != nil {
			return
		}
		z, err := time.LoadLocation(strings.TrimSpace(string(out)))
		if err != nil {
			return
		}
		time.Local = z
	}
}

func InitGkillServerAPI() error {
	var err error

	gkillServerAPI, err = api.NewGkillServerAPI()
	if err != nil {
		return err
	}
	return nil
}

func LaunchGkillServerAPI() error {
	var err error
	defer gkillServerAPI.Close()
	interceptCh := make(chan os.Signal, 1)
	signal.Notify(interceptCh, os.Interrupt)
	done := threads.AllocateThread()
	go func() {
		defer done()
		<-interceptCh
		gkillServerAPI.Close()
		os.Exit(0)
	}()

	for err == nil {
		err = gkillServerAPI.Serve()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
				// サーバが正常に閉じられた場合はスルーして立ち上げ直す
			} else {
				gkill_log.Error.Printf("error at gkill server api serve: %v", err)
				return err
			}
		}
		err = InitGkillServerAPI()
		if err != nil {
			return err
		}
	}

	return nil
}

func GetGkillServerAPI() *api.GkillServerAPI {
	return gkillServerAPI
}

func Openbrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func GetVersion() (string, error) {
	assetsDirName := "embed/html/assets"
	files, err := api.HTMLFS.ReadDir(assetsDirName)
	if err != nil {
		return "", err
	}

	var indexJSFile fs.DirEntry
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "index") && strings.HasSuffix(file.Name(), ".js") {
			indexJSFile = file
		}
	}
	if indexJSFile == nil {
		return "", fmt.Errorf("not found gkill page js file.")
	}

	jsFile, err := api.HTMLFS.Open(assetsDirName + "/" + indexJSFile.Name())
	if err != nil {
		return "", fmt.Errorf("error at open gkill page js file.")
	}
	defer jsFile.Close()

	jsFileContentBytes, err := io.ReadAll(jsFile)
	if err != nil {
		return "", fmt.Errorf("error at read gkill page js file: %w", err)
	}
	jsFileContent := string(jsFileContentBytes)

	regex, err := regexp.Compile(`const name = "gkill";[\s\S\n]*?const version = "(.*)";`)
	if err != nil {
		return "", fmt.Errorf("error at regex compile: %w", err)
	}
	versionStrings := regex.FindStringSubmatch(jsFileContent)

	if len(versionStrings) != 2 {
		return "", fmt.Errorf("failed extruct version info: version match is not 2")
	}
	return "v" + versionStrings[1], nil
}
