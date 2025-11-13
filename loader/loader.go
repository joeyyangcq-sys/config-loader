package loader

import (
    conf "config-loader/conf"
    provider "config-loader/conf/provider"
    "sync/atomic"
)

type Loader struct {
    p        provider.Provider
    cur      atomic.Value
    onUpdate func(conf.Options)
}

func New(p provider.Provider) *Loader { return &Loader{p: p} }

func (l *Loader) Load() (conf.Options, error) {
    opts, err := conf.LoadOptionsFromProvider(l.p)
    if err != nil {
        return opts, err
    }
    l.cur.Store(opts)
    if l.onUpdate != nil {
        l.onUpdate(opts)
    }
    return opts, nil
}

func (l *Loader) Current() conf.Options {
    v := l.cur.Load()
    if v == nil {
        var o conf.Options
        return o
    }
    return v.(conf.Options)
}

func (l *Loader) SetOnUpdate(fn func(conf.Options)) { l.onUpdate = fn }

func (l *Loader) Watch() error {
    return l.p.Watch(func() error { _, _ = l.Load(); return nil })
}

func NewFile(path string) *Loader { return New(provider.NewFile(path)) }

func NewEtcd(endpoints []string, key, user, pass string) *Loader {
    return New(provider.NewEtcd(endpoints, key, user, pass))
}

func NewNacos(serverAddrs []string, namespaceID, group, dataID string) *Loader {
    return New(provider.NewNacos(serverAddrs, namespaceID, group, dataID))
}