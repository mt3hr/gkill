package gkill_log

import (
	"io"
	"log"
)

var (
	Info     *log.Logger
	Debug    *log.Logger
	Trace    *log.Logger
	TraceSQL *log.Logger
)

func init() {
	Info = log.New(io.Discard, "INFO: ", log.LstdFlags)
	Debug = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
	Trace = log.New(io.Discard, "TRACE: ", log.LstdFlags)
	TraceSQL = log.New(io.Discard, "TRACE_SQL: ", log.LstdFlags)
}
