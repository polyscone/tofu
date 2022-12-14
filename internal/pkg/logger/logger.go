package logger

import (
	"log"
	"os"
)

var (
	Info  = log.New(New(os.Stdout, OutputStyle), "", 0)
	Error = log.New(New(os.Stderr, OutputStyle), "", 0)
)
