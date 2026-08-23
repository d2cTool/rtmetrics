// Package pool — типизированная обёртка над sync.Pool для объектов с методом Reset.
package pool

import "sync"

// Resetter — объект, который можно вернуть в исходное состояние перед повторным использованием.
type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт пул. newFunc вызывается, когда свободных объектов нет.
// Если newFunc == nil, Get при пустом пуле возвращает нулевое значение T.
func New[T Resetter](newFunc func() T) *Pool[T] {
	p := &Pool[T]{}
	if newFunc != nil {
		p.pool.New = func() any {
			return newFunc()
		}
	}
	return p
}

// Get возвращает объект из пула. Если пул пуст, вызывает newFunc
// или возвращает нулевое T, когда конструктор не задан.
func (p *Pool[T]) Get() T {
	v := p.pool.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// Put сбрасывает состояние объекта и кладёт его обратно в пул.
func (p *Pool[T]) Put(value T) {
	value.Reset()
	p.pool.Put(value)
}
