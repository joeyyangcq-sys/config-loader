package conf

import (
	provider "config-loader/conf/provider"
	"os"
	"testing"
)

func TestLoad_OK(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.yaml")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	defer tmp.Close()
	_, _ = tmp.WriteString("welcome:\n  title: 't'\n  messages: ['m']\n  tail: ''\nserver:\n  bind: ':9090'\n")
	var opts Options
	if err := Load(tmp.Name(), &opts); err != nil {
		t.Fatalf("load: %v", err)
	}
	if opts.Server.Bind != ":9090" || opts.Welcome.Title != "t" {
		t.Fatalf("bad opts: %+v", opts)
	}
}

func TestLoad_ErrEmptyPath(t *testing.T) {
	var opts Options
	if err := Load("", &opts); err == nil {
		t.Fatalf("want error")
	}
}

func TestLoadFromProvider_File(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.yaml")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	defer tmp.Close()
	_, _ = tmp.WriteString("welcome:\n  title: 'x'\n  messages: []\n  tail: ''\nserver:\n  bind: ':8081'\n")
	p := provider.NewFile(tmp.Name())
	var opts Options
	if err := LoadFromProvider(p, &opts); err != nil {
		t.Fatalf("load from provider: %v", err)
	}
	if opts.Welcome.Title != "x" || opts.Server.Bind != ":8081" {
		t.Fatalf("bad opts: %+v", opts)
	}
}

func TestLoadOptionsFromProvider_Defaults(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.yaml")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	defer tmp.Close()
	_, _ = tmp.WriteString("welcome:\n  title: 'd'\n  messages: []\n  tail: ''\n")
	p := provider.NewFile(tmp.Name())
	opts, err := LoadOptionsFromProvider(p)
	if err != nil {
		t.Fatalf("load opts: %v", err)
	}
	if opts.Server.Bind != ":8080" || opts.Welcome.Title != "d" {
		t.Fatalf("bad opts: %+v", opts)
	}
}

func TestLoadDir(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(d+"/a.yaml", []byte("x: 1\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(d+"/b.yml", []byte("y: 2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := LoadDir(d)
	if err != nil {
		t.Fatalf("load dir: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("bad size: %d", len(m))
	}
	if m["a.yaml"]["x"].(int) != 1 || m["b.yml"]["y"].(int) != 2 {
		t.Fatalf("bad content: %+v", m)
	}
}

type multiProv struct{}

func (multiProv) Open() ([]provider.Content, error) {
	return []provider.Content{
		{ID: "a", Group: "g", Payload: "welcome:\n  title: 'A'\n  messages: ['m1']\n  tail: ''\nserver:\n  bind: ':1001'\n"},
		{ID: "b", Group: "g", Payload: "welcome:\n  title: 'B'\n  messages: ['m2']\n  tail: 'X'\n"},
	}, nil
}
func (multiProv) Watch(func() error) error { return nil }

func TestLoadFromProvider_MergeAll(t *testing.T) {
	var opts Options
	if err := LoadFromProvider(multiProv{}, &opts); err != nil {
		t.Fatalf("load from provider: %v", err)
	}
	if opts.Welcome.Title != "B" || opts.Welcome.Tail != "X" || opts.Server.Bind != ":1001" || len(opts.Welcome.Messages) != 1 || opts.Welcome.Messages[0] != "m2" {
		t.Fatalf("bad opts: %+v", opts)
	}
}

func TestLoadOptionsFromProvider_MergeAll(t *testing.T) {
	opts, err := LoadOptionsFromProvider(multiProv{})
	if err != nil {
		t.Fatalf("load opts: %v", err)
	}
	if opts.Welcome.Title != "B" || opts.Welcome.Tail != "X" || opts.Server.Bind != ":1001" || len(opts.Welcome.Messages) != 1 || opts.Welcome.Messages[0] != "m2" {
		t.Fatalf("bad opts: %+v", opts)
	}
}
