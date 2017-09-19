// +build !windows

package parser

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var vendors []string

func init() {
	if pwd, err := os.Getwd(); err != nil {
		fmt.Printf("Warning: %s\n", err)
	} else {
		if fi, err := os.Stat(pwd + "/vendor"); err == nil && fi.IsDir() {
			vendors = append(vendors, pwd+"/vendor")
		}
	}

	vendors = append(vendors, strings.Split(os.Getenv("GOPATH"), ":")...)
}

func tryFindPath(fname, dir string) string {
	if len(fname) > 0 && fname[0] == '.' {
		// path is relative
		return path.Clean(path.Join(dir, fname))
	}

	for _, p := range vendors {
		filepath := fname
		if !strings.HasPrefix(fname, p) {
			filepath = path.Join(p, "src") + "/" + fname
		}

		if fi, err := os.Stat(filepath); err == nil && fi.IsDir() {
			return filepath
		}
	}

	return ""
}
