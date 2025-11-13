package provider

import "testing"

func TestNacos_WatchEnsureError(t *testing.T) {
    p := NewNacos([]string{}, "", "DEFAULT_GROUP", "x")
    if err := p.Watch(func() error { return nil }); err == nil {
        t.Fatalf("want error")
    }
}

