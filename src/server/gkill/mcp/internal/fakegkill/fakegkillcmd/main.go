// fakegkillcmd は偽 gkill を単独プロセスで立てる（Node 実装からのゴールデン採取用）。
//
//	go run ./gkill/mcp/internal/fakegkill/fakegkillcmd -addr 127.0.0.1:19999
//
// 記録した要求は GET /__control/drain で取り出し、POST /__control/reset で状態を消す。
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/internal/fakegkill"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "listen address")
	flag.Parse()
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("listening %s\n", listener.Addr().String())
	if err := http.Serve(listener, fakegkill.New()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
