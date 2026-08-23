// Command reset генерирует методы Reset() для структур с комментарием // generate:reset.
//
// Запуск из корня модуля:
//
//	go run ./cmd/reset
//
// Для каждого пакета методы пишутся в reset.gen.go того же пакета.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		exitErr(err)
	}
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "reset: %v\n", err)
	os.Exit(1)
}

func run() error {
	root, err := findModuleRoot()
	if err != nil {
		return err
	}
	return processRoot(root)
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func processRoot(root string) error {
	dirs := make(map[string]struct{})
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == generatedFile {
			return nil
		}
		dirs[filepath.Dir(path)] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}

	for dir := range dirs {
		if err := processPackage(dir); err != nil {
			return err
		}
	}
	return nil
}

func skipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "vendor", "testdata", "node_modules":
		return true
	default:
		return false
	}
}

func processPackage(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	structs, pkgName, err := parsePackage(dir, entries)
	if err != nil {
		return err
	}

	out := filepath.Join(dir, generatedFile)
	if len(structs) == 0 {
		_ = os.Remove(out)
		return nil
	}

	src, err := generateFile(pkgName, structs)
	if err != nil {
		return fmt.Errorf("%s: %w", dir, err)
	}
	return os.WriteFile(out, src, 0o644)
}
