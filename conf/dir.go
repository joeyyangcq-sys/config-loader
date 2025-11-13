package conf

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadDir(dir string) (map[string]map[string]any, error) {
	out := make(map[string]map[string]any)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if !strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(lower, ".yml") {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		var m map[string]any
		if e = yaml.Unmarshal(b, &m); e != nil {
			return e
		}
		out[d.Name()] = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
