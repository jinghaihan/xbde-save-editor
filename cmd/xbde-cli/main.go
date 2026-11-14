package main

import (
	"github.com/jinghaihan/xbde-save-editor/internal/cli"
	"os"
)

func main() { os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
