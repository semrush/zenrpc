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
		pwd = normalizePath(pwd)
		if fi, err := os.Stat(pwd + "/vendor"); err == nil && fi.IsDir() {
			vendors = append(vendors, pwd+"/vendor")
		}
	}

	for _, p := range strings.Split(os.Getenv("GOPATH"), ";") {
		vendors = append(vendors, normalizePath(p))
	}
}

func normalizePath(path string) string {
	// use lower case, as Windows file systems will almost always be case insensitive
	return strings.ToLower(strings.Replace(path, "\\", "/", -1))
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
