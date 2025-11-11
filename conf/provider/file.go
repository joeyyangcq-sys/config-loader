package provider

import (
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileProvider 从本地文件读取一个配置文档，支持 fsnotify 热更新。
type FileProvider struct {
	Path string
}

func NewFile(path string) *FileProvider {
	return &FileProvider{Path: path}
}

func (p *FileProvider) Open() ([]Content, error) {
	b, err := os.ReadFile(p.Path)
	if err != nil {
		return nil, err
	}
	return []Content{{ID: p.Path, Group: "file", Payload: string(b)}}, nil
}

func (p *FileProvider) Watch(onChange func() error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(p.Path); err != nil {
		_ = watcher.Close()
		return err
	}
	// 简单的事件监听与轻微防抖
	go func() {
		defer watcher.Close()
		var last time.Time
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					// 防抖，避免编辑器触发多次写入事件
					if time.Since(last) > 200*time.Millisecond {
						_ = onChange()
						last = time.Now()
					}
				}
				// 若文件被移除后重建，尝试重新添加监听
				if ev.Op&fsnotify.Remove != 0 {
					time.Sleep(200 * time.Millisecond)
					_ = watcher.Add(p.Path)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// 忽略错误，保持监听
			}
		}
	}()
	return nil
}
