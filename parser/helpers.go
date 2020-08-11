package parser

import (
	"fmt"
	"github.com/thoas/go-funk"
	"go/ast"
	"golang.org/x/tools/go/packages"
	"path"
	"strings"
)

func filterFile(filepath string) bool {
	if !strings.HasSuffix(filepath, goFileSuffix) ||
		strings.HasSuffix(filepath, GenerateFileSuffix) || strings.HasSuffix(filepath, testFileSuffix) {
		return false
	}
	return true
}

func GetDependencies(entryPoint string) ([]string, error) {
	dir := path.Dir(entryPoint)
	allGoFiles, err := getDependenciesFilenames(dir)
	if err != nil {
		return nil, err
	}
	return allGoFiles, nil
}

func getDependenciesFilenames(dir string) ([]string, error) {
	goFiles := []string{}
	pkgs, err := loadPackage(dir)
	if err != nil {
		return nil, err
	}
	for _, pack := range pkgs {
		goFiles = append(goFiles, goFilesFromPackage(pack)...)
		for _, childPack := range pack.Imports {
			goFiles = append(goFiles, goFilesFromPackage(childPack)...)
		}
	}
	return funk.UniqString(goFiles), nil
}

func GetDependenciesAstFiles(filename string) ([]*ast.File, error) {
	pkgs, err := loadPackageWithSyntax(path.Dir(filename))
	if err != nil {
		return nil, err
	}
	astFiles := []*ast.File{}
	done := map[string]bool{}
	for _, pkg := range pkgs {
		if _, ok := done[pkg.Name]; ok {
			continue
		}
		astFiles = append(astFiles, pkg.Syntax...)
		done[pkg.Name] = true
		for _, childPack := range pkg.Imports {
			if _, ok := done[childPack.Name]; ok {
				continue
			}
			astFiles = append(astFiles, childPack.Syntax...)
			done[childPack.Name] = true
		}
	}
	return astFiles, nil
}

func goFilesFromPackage(pkg *packages.Package) []string {
	files := []string{}
	files = append(files, pkg.GoFiles...)
	return funk.FilterString(files, filterFile)
}

func EntryPointPackageName(filename string) (string, error) {
	pkgs, err := loadPackage(path.Dir(filename))
	if err != nil {
		return "", err
	}
	for _, pack := range pkgs {
		return pack.Name, nil
	}
	return "", fmt.Errorf("package not found for entry point")
}

func loadPackage(path string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedImports | packages.NeedFiles | packages.NeedName,
	}, path)
}

func loadPackageWithSyntax(path string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedImports |
			packages.NeedFiles |
			packages.NeedName |
			packages.NeedSyntax,
	}, path)
}
