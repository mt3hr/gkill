package main

import (
	"log"
	"os"

	"github.com/mt3hr/gkill/src/app/gkill/main/common"
	"github.com/spf13/cobra"
)

func main() {
	if err := appCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.MousetrapHelpText = "" // Windowsでマウスから起動しても怒られないようにする
}

var (
	appCmd = &cobra.Command{
		Use: "gkill_server",
		Run: func(_ *cobra.Command, _ []string) {
			var err error

			err = common.InitGkillServerAPI()
			if err != nil {
				log.Fatal(err)
			}

			err = common.LaunchGkillServerAPI()
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		},
	}
)
