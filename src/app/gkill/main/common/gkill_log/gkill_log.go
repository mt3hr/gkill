package gkill_log

import (
	"io"
	"log"
	"os"
)

var (
	Info     *log.Logger
	Debug    *log.Logger
	Trace    *log.Logger
	TraceSQL *log.Logger
)

func init() {
	Info = log.New(os.Stdout, "INFO: ", log.LstdFlags)
	Debug = log.New(os.Stdout, "DEBUG: ", log.LstdFlags)
	Trace = log.New(os.Stdout, "TRACE: ", log.LstdFlags)
	TraceSQL = log.New(io.Discard, "TRACE_SQL: ", log.LstdFlags)
}
