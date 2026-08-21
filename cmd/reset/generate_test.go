package main

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exampleSrc = `package example

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}
`

func TestGenerateFromExample(t *testing.T) {
	got, err := generateFromSource("example.go", exampleSrc)
	require.NoError(t, err)
	src := string(got)

	assert.Contains(t, src, "package example")
	assert.Contains(t, src, "func (r *ResetableStruct) Reset()")
	assert.Contains(t, src, "if r == nil")
	assert.Contains(t, src, "r.i = 0")
	assert.Contains(t, src, `r.str = ""`)
	assert.Contains(t, src, "if r.strP != nil")
	assert.Contains(t, src, `*r.strP = ""`)
	assert.Contains(t, src, "r.s = r.s[:0]")
	assert.Contains(t, src, "clear(r.m)")
	assert.Contains(t, src, "if r.child != nil")
	assert.Contains(t, src, "resetter.Reset()")
}

func TestSkipsStructsWithoutComment(t *testing.T) {
	got, err := generateFromSource("skip.go", `package p

type Plain struct {
	N int
}
`)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGenerateGenericStruct(t *testing.T) {
	got, err := generateFromSource("generic.go", `package p

// generate:reset
type Box[T any] struct {
	n int
	s []T
}
`)
	require.NoError(t, err)
	src := string(got)
	assert.Contains(t, src, "func (b *Box[T]) Reset()")
	assert.Contains(t, src, "b.n = 0")
	assert.Contains(t, src, "b.s = b.s[:0]")
}

func TestHasGenerateResetOnTypeSpec(t *testing.T) {
	src := `package p

type (
	// generate:reset
	Marked struct {
		OK bool
	}

	Unmarked struct {
		OK bool
	}
)
`
	f, err := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.ParseComments)
	require.NoError(t, err)
	structs := structsFromFile(f)
	require.Len(t, structs, 1)
	assert.Equal(t, "Marked", structs[0].name)
}

func TestProcessRootWritesResetGen(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "example")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testreset\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "example.go"), []byte(exampleSrc), 0o644))

	require.NoError(t, processRoot(root))

	genPath := filepath.Join(pkgDir, generatedFile)
	got, err := os.ReadFile(genPath)
	require.NoError(t, err, "должен появиться reset.gen.go")
	assert.Contains(t, string(got), "func (r *ResetableStruct) Reset()")
	assert.Contains(t, string(got), "r.s = r.s[:0]")
	assert.Contains(t, string(got), "clear(r.m)")

	cmd := exec.Command("go", "build", "./example")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestProcessRootRemovesStaleFile(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "empty")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testreset\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "empty.go"), []byte("package empty\n\ntype Foo struct{ N int }\n"), 0o644))
	stale := filepath.Join(pkgDir, generatedFile)
	require.NoError(t, os.WriteFile(stale, []byte("package empty\n"), 0o644))

	require.NoError(t, processRoot(root))
	_, err := os.Stat(stale)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
