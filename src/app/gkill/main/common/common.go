package common

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"time"

	_ "time/tzdata"

	"net/http"
	_ "net/http/pprof"

	"github.com/mt3hr/gkill/src/app/gkill/api"
)

var (
	gkillServerAPI *api.GkillServerAPI
)

func init() {
	if "" == os.Getenv("HOME") {
		os.Setenv("HOME", os.Getenv("HOMEPATH"))
	}
	fixTimezone()

	go func() {
		http.ListenAndServe("localhost:6060", nil) // pprof用
	}()
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
	interceptCh := make(chan os.Signal)
	signal.Notify(interceptCh, os.Interrupt)
	go func() {
		<-interceptCh
		gkillServerAPI.Close()
		os.Exit(0)
	}()

	err = gkillServerAPI.Serve()
	if err != nil {
		return err
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
