package common

import (
	"flag"
	"time"
)

// Bindings связывает флаги с полями конфига. Apply копирует значение только
// если флаг явно передан: нули (false, 0, "") остаются валидным перекрытием.
//
// viper.BindEnv только мапит ключ на переменную; порядок всё равно
// Set > flag > env, то есть ADDRESS не перекроет явно переданный -a.
// mergo по нулям пропускает restore=false / interval=0 / "" либо затирает
// незаданные поля. Поэтому флаги накладываются через Visit, env — через env.Parse.
type Bindings struct {
	apply map[string]func()
}

// NewBindings создаёт пустую таблицу привязок флагов.
func NewBindings() *Bindings {
	return &Bindings{apply: make(map[string]func())}
}

func (b *Bindings) add(name string, apply func()) {
	if b.apply == nil {
		b.apply = make(map[string]func())
	}
	b.apply[name] = apply
}

// String регистрирует строковый флаг и отложенную запись в dest.
func (b *Bindings) String(fs *flag.FlagSet, name, value, usage string, dest *string) {
	v := value
	fs.StringVar(&v, name, value, usage)
	b.add(name, func() { *dest = v })
}

// Int регистрирует целочисленный флаг и отложенную запись в dest.
func (b *Bindings) Int(fs *flag.FlagSet, name string, value int, usage string, dest *int) {
	v := value
	fs.IntVar(&v, name, value, usage)
	b.add(name, func() { *dest = v })
}

// Bool регистрирует булев флаг и отложенную запись в dest.
func (b *Bindings) Bool(fs *flag.FlagSet, name string, value bool, usage string, dest *bool) {
	v := value
	fs.BoolVar(&v, name, value, usage)
	b.add(name, func() { *dest = v })
}

// Duration регистрирует duration-флаг и отложенную запись в dest.
func (b *Bindings) Duration(fs *flag.FlagSet, name string, value time.Duration, usage string, dest *time.Duration) {
	v := value
	fs.DurationVar(&v, name, value, usage)
	b.add(name, func() { *dest = v })
}

// Apply переносит в dest только флаги, которые реально были в командной строке.
func (b *Bindings) Apply(fs *flag.FlagSet) {
	ApplyVisited(fs, b.apply)
}

// ApplyVisited вызывает apply[name] для каждого явно переданного флага.
func ApplyVisited(fs *flag.FlagSet, apply map[string]func()) {
	if len(apply) == 0 {
		return
	}
	fs.Visit(func(f *flag.Flag) {
		if fn, ok := apply[f.Name]; ok {
			fn()
		}
	})
}
