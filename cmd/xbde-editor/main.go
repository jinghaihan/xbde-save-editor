package main

import (
	"fmt"
	"github.com/jinghaihan/xbde-save-editor/internal/app"
	"os"
)

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if e := app.Run(path); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
