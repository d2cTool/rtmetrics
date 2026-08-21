// Package pool — типизированная обёртка над sync.Pool для объектов с методом Reset.
package pool

import "sync"

// Resetter — объект, который можно вернуть в исходное состояние перед повторным использованием.
type Resetter interface {
	Reset()
}

// generate:reset
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт пул. newFunc вызывается, когда свободных объектов нет.
func New[T Resetter](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
	}
}

// Get возвращает объект из пула или создаёт новый через конструктор, переданный в New.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put сбрасывает состояние объекта и кладёт его обратно в пул.
func (p *Pool[T]) Put(value T) {
	value.Reset()
	p.pool.Put(value)
}
