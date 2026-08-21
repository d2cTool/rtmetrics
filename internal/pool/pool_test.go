package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type item struct {
	value  int
	resets int
}

func (i *item) Reset() {
	i.value = 0
	i.resets++
}

func TestNewReturnsPool(t *testing.T) {
	t.Parallel()

	p := New(func() *item { return &item{} })
	require.NotNil(t, p)
}

func TestGetCreatesViaConstructor(t *testing.T) {
	t.Parallel()

	p := New(func() *item {
		return &item{value: 42}
	})

	got := p.Get()
	require.NotNil(t, got)
	assert.Equal(t, 42, got.value)
	assert.Equal(t, 0, got.resets)
}

func TestPutResetsBeforeReuse(t *testing.T) {
	t.Parallel()

	p := New(func() *item { return &item{} })

	first := p.Get()
	first.value = 7
	p.Put(first)

	assert.Equal(t, 0, first.value, "после Put объект должен быть сброшен")
	assert.Equal(t, 1, first.resets)
}

func TestGetPutConcurrent(t *testing.T) {
	t.Parallel()

	p := New(func() *item { return &item{} })

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 64 {
				it := p.Get()
				it.value++
				p.Put(it)
			}
		}()
	}
	wg.Wait()
}
