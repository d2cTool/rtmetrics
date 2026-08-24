package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ResolveConfigPath возвращает путь к JSON-конфигу: CONFIG перекрывает флаг -c/-config.
func ResolveConfigPath(flagPath string) string {
	if v, ok := os.LookupEnv("CONFIG"); ok && v != "" {
		return v
	}
	return flagPath
}

// LoadJSON читает dest из JSON-файла path.
func LoadJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

// VisitedFlags возвращает имена флагов, явно переданных в fs.
func VisitedFlags(fs *flag.FlagSet) map[string]struct{} {
	set := make(map[string]struct{})
	fs.Visit(func(f *flag.Flag) {
		set[f.Name] = struct{}{}
	})
	return set
}

// FlagPassed сообщает, задан ли хотя бы один из names через командную строку.
func FlagPassed(visited map[string]struct{}, names ...string) bool {
	for _, name := range names {
		if _, ok := visited[name]; ok {
			return true
		}
	}
	return false
}

// ParseIntervalSeconds разбирает интервал из JSON: "1s", "300ms" или целое число секунд.
func ParseIntervalSeconds(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return int(d / time.Second), nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return n, nil
}
