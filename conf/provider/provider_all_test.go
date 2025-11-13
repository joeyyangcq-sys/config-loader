package provider

import (
    "bytes"
    "context"
    "encoding/base64"
    clientv3 "go.etcd.io/etcd/client/v3"
    "net/http"
    "net/url"
    "os"
    "os/exec"
    "testing"
    "time"
)

func runCompose(args ...string) error {
    c1 := exec.Command("docker", append([]string{"compose"}, args...)...)
    c1.Dir = "../../"
    if err := c1.Run(); err == nil {
        return nil
    }
    c2 := exec.Command("docker-compose", args...)
    c2.Dir = "../../"
    return c2.Run()
}

func ensureCompose(t *testing.T) {
    c1 := exec.Command("docker", "compose", "version")
    if err := c1.Run(); err == nil {
        return
    }
    c2 := exec.Command("docker-compose", "version")
    if err := c2.Run(); err == nil {
        return
    }
    t.Skip("docker compose not available")
}

func startEtcd(t *testing.T) {
    ensureCompose(t)
    if err := runCompose("ps"); err != nil {
        t.Skip("compose not available")
    }
    if err := runCompose("up", "-d", "etcd"); err != nil {
        t.Fatalf("compose up etcd: %v", err)
    }
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        resp, err := http.Get("http://127.0.0.1:2379/version")
        if err == nil && resp.StatusCode == 200 {
            return
        }
        time.Sleep(2 * time.Second)
    }
    t.Fatalf("etcd not ready")
}

func stopEtcd(t *testing.T) { _ = runCompose("rm", "-f", "-s", "-v", "etcd") }

func putEtcd(key, val string) error {
    k := base64.StdEncoding.EncodeToString([]byte(key))
    v := base64.StdEncoding.EncodeToString([]byte(val))
    body := []byte("{\"key\":\"" + k + "\",\"value\":\"" + v + "\"}")
    resp, err := http.Post("http://127.0.0.1:2379/v3/kv/put", "application/json", bytes.NewReader(body))
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        return context.DeadlineExceeded
    }
    return nil
}

func startNacos(t *testing.T) {
    ensureCompose(t)
    if err := runCompose("ps"); err != nil {
        t.Skip("compose not available")
    }
    if err := runCompose("up", "-d", "nacos"); err != nil {
        t.Fatalf("compose up nacos: %v", err)
    }
    deadline := time.Now().Add(90 * time.Second)
    for time.Now().Before(deadline) {
        resp, err := http.Get("http://127.0.0.1:8848/nacos/")
        if err == nil && (resp.StatusCode == 200 || resp.StatusCode == 302 || resp.StatusCode == 401) {
            return
        }
        time.Sleep(2 * time.Second)
    }
    t.Fatalf("nacos not ready")
}

func stopNacos(t *testing.T) { _ = runCompose("rm", "-f", "-s", "-v", "nacos") }

func publishNacos(dataId, group, content string) error {
    form := url.Values{}
    form.Set("dataId", dataId)
    form.Set("group", group)
    form.Set("content", content)
    resp, err := http.Post("http://127.0.0.1:8848/nacos/v1/cs/configs", "application/x-www-form-urlencoded", bytes.NewReader([]byte(form.Encode())))
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        return err
    }
    return nil
}

func TestFile_Open(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "f-*.yaml")
    if err != nil {
        t.Fatalf("tmp: %v", err)
    }
    defer tmp.Close()
    _, _ = tmp.WriteString("a: 1\n")
    p := NewFile(tmp.Name())
    cs, err := p.Open()
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    if len(cs) != 1 || cs[0].Payload == "" || cs[0].Group != "file" {
        t.Fatalf("bad content: %+v", cs)
    }
    if cs[0].ID != tmp.Name() {
        t.Fatalf("bad id: %s", cs[0].ID)
    }
}

func TestFile_Watch(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "f-*.yaml")
    if err != nil {
        t.Fatalf("tmp: %v", err)
    }
    defer tmp.Close()
    p := NewFile(tmp.Name())
    ch := make(chan struct{}, 1)
    if err := p.Watch(func() error { ch <- struct{}{}; return nil }); err != nil {
        t.Fatalf("watch: %v", err)
    }
    _, _ = tmp.WriteString("b: 2\n")
    select {
    case <-ch:
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout")
    }
}

func TestFile_WatchRename(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "f-*.yaml")
    if err != nil {
        t.Fatalf("tmp: %v", err)
    }
    defer tmp.Close()
    p := NewFile(tmp.Name())
    ch := make(chan struct{}, 1)
    if err := p.Watch(func() error { ch <- struct{}{}; return nil }); err != nil {
        t.Fatalf("watch: %v", err)
    }
    newPath := tmp.Name() + ".renamed"
    if err := os.Rename(tmp.Name(), newPath); err != nil {
        t.Fatalf("rename: %v", err)
    }
    _, _ = os.Create(tmp.Name())
    select {
    case <-ch:
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout")
    }
}

func TestFile_WatchAddError(t *testing.T) {
    p := NewFile("/not-exist-abc.yaml")
    if err := p.Watch(func() error { return nil }); err == nil {
        t.Fatalf("want error")
    }
}

func TestEtcd_OpenAndWatch(t *testing.T) {
    startEtcd(t)
    defer stopEtcd(t)
    key := "/config-loader/test.yaml"
    val := "a: 1\n"
    if err := putEtcd(key, val); err != nil {
        t.Fatalf("seed etcd: %v", err)
    }
    p := NewEtcd([]string{"127.0.0.1:2379"}, key, "", "")
    cs, err := p.Open()
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    if len(cs) != 1 || cs[0].Payload != val {
        t.Fatalf("bad content: %+v", cs)
    }
    ch := make(chan struct{}, 1)
    if err := p.Watch(func() error { ch <- struct{}{}; return nil }); err != nil {
        t.Fatalf("watch: %v", err)
    }
    if err := putEtcd(key, "a: 2\n"); err != nil {
        t.Fatalf("update etcd: %v", err)
    }
    select {
    case <-ch:
    case <-time.After(5 * time.Second):
        t.Fatalf("timeout")
    }
}

func TestEtcd_Open_NoKV(t *testing.T) {
    startEtcd(t)
    defer stopEtcd(t)
    p := NewEtcd([]string{"127.0.0.1:2379"}, "/config-loader/not-exist", "", "")
    cs, err := p.Open()
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    if len(cs) != 0 {
        t.Fatalf("want empty, got %d", len(cs))
    }
}

func TestEtcd_EnsureClientCached(t *testing.T) {
    p := &EtcdProvider{cli: &clientv3.Client{}}
    if err := p.ensureClient(); err != nil {
        t.Fatalf("ensure: %v", err)
    }
}

func TestEtcd_EnsureClientWithAuth(t *testing.T) {
    p := NewEtcd([]string{"127.0.0.1:1"}, "/x", "user", "pass")
    _ = p.ensureClient()
}

func TestEtcd_OpenEnsureError(t *testing.T) {
    p := NewEtcd([]string{"127.0.0.1:1"}, "/x", "", "")
    if _, err := p.Open(); err == nil {
        t.Fatalf("want error")
    }
}

func TestEtcd_WatchStart(t *testing.T) {
    p := NewEtcd([]string{"127.0.0.1:1"}, "/x", "", "")
    if err := p.Watch(func() error { return nil }); err != nil {
        t.Fatalf("watch: %v", err)
    }
}

func TestNacos_OpenAndWatch(t *testing.T) {
    startNacos(t)
    defer stopNacos(t)
    dataId := "test.yaml"
    group := "DEFAULT_GROUP"
    val := "a: 1\n"
    if err := publishNacos(dataId, group, val); err != nil {
        t.Fatalf("seed nacos: %v", err)
    }
    p := NewNacos([]string{"127.0.0.1:8848"}, "", group, dataId)
    cs, err := p.Open()
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    if len(cs) != 1 || cs[0].Payload == "" {
        t.Fatalf("bad content: %+v", cs)
    }
    ch := make(chan struct{}, 1)
    if err := p.Watch(func() error { ch <- struct{}{}; return nil }); err != nil {
        t.Fatalf("watch: %v", err)
    }
    if err := publishNacos(dataId, group, "a: 2\n"); err != nil {
        t.Fatalf("update nacos: %v", err)
    }
    select {
    case <-ch:
    case <-time.After(10 * time.Second):
        t.Fatalf("timeout")
    }
}

func TestNacos_EnsureClient(t *testing.T) {
    p := NewNacos([]string{"127.0.0.1:8848", "bad"}, "", "DEFAULT_GROUP", "x")
    if err := p.ensureClient(); err != nil {
        t.Fatalf("ensure: %v", err)
    }
}

func TestNacos_EnsureClientError(t *testing.T) {
    p := NewNacos([]string{}, "", "DEFAULT_GROUP", "x")
    if err := p.ensureClient(); err == nil {
        t.Fatalf("want error")
    }
}

func TestSplitHostPort(t *testing.T) {
    h, p := splitHostPort("127.0.0.1:8848")
    if h != "127.0.0.1" || p != 8848 {
        t.Fatalf("bad: %s %d", h, p)
    }
    h, p = splitHostPort("bad")
    if h != "bad" || p != 0 {
        t.Fatalf("bad: %s %d", h, p)
    }
    h, p = splitHostPort("")
    if h != "" || p != 0 {
        t.Fatalf("bad: %s %d", h, p)
    }
}

type emptyProvider struct{}

func (emptyProvider) Open() ([]Content, error) { return []Content{}, nil }
func (emptyProvider) Watch(func() error) error { return nil }

func TestManager_LoadEmpty(t *testing.T) {
    m := NewManager(emptyProvider{})
    if err := m.Load(); err == nil {
        t.Fatalf("want error")
    }
}

func TestManager_CurrentEmpty(t *testing.T) {
    m := NewManager(emptyProvider{})
    cur := m.Current()
    if cur.Doc == nil || len(cur.Doc) != 0 {
        t.Fatalf("bad cur: %+v", cur)
    }
}

type badProvider struct{}

func (badProvider) Open() ([]Content, error) { return []Content{{ID: "x", Group: "g", Payload: "1"}}, nil }
func (badProvider) Watch(func() error) error { return nil }

func TestManager_LoadParseError(t *testing.T) {
    m := NewManager(badProvider{})
    if err := m.Load(); err == nil {
        t.Fatalf("want error")
    }
}

type goodProvider struct{}

func (goodProvider) Open() ([]Content, error) { return []Content{{ID: "x", Group: "g", Payload: "a: 1\n"}}, nil }
func (goodProvider) Watch(func() error) error { return nil }

func TestManager_Load_OnUpdate(t *testing.T) {
    m := NewManager(goodProvider{})
    ch := make(chan struct{}, 1)
    m.SetOnUpdate(func(Generic) { ch <- struct{}{} })
    if err := m.Load(); err != nil {
        t.Fatalf("load: %v", err)
    }
    select {
    case <-ch:
    default:
        t.Fatalf("no update")
    }
}

func TestManager_LoadLookup(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "m-*.yaml")
    if err != nil {
        t.Fatalf("tmp: %v", err)
    }
    defer tmp.Close()
    _, _ = tmp.WriteString("welcome:\n  title: 't'\nserver:\n  bind: ':1111'\n")
    p := NewFile(tmp.Name())
    m := NewManager(p)
    if err := m.Load(); err != nil {
        t.Fatalf("load: %v", err)
    }
    v, ok := m.Lookup("welcome.title")
    if !ok || v.(string) != "t" {
        t.Fatalf("lookup: %v %v", ok, v)
    }
    if _, ok := m.Lookup("welcome.notfound"); ok {
        t.Fatalf("want not found")
    }
}

func TestManager_Watch(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "m-*.yaml")
    if err != nil {
        t.Fatalf("tmp: %v", err)
    }
    defer tmp.Close()
    _, _ = tmp.WriteString("a: 1\n")
    p := NewFile(tmp.Name())
    m := NewManager(p)
    if err := m.Load(); err != nil {
        t.Fatalf("load: %v", err)
    }
    ch := make(chan struct{}, 1)
    m.SetOnUpdate(func(Generic) { ch <- struct{}{} })
    if err := m.Watch(); err != nil {
        t.Fatalf("watch: %v", err)
    }
    _ = os.WriteFile(tmp.Name(), []byte("a: 2\n"), 0644)
    select {
    case <-ch:
    case <-time.After(3 * time.Second):
        t.Fatalf("timeout")
    }
}

type errProvider struct{}

func (errProvider) Open() ([]Content, error) { return []Content{{ID: "x", Group: "g", Payload: "a: 1\n"}}, nil }
func (errProvider) Watch(func() error) error { return &watchErr{} }

type watchErr struct{}

func (*watchErr) Error() string { return "watch error" }

func TestManager_WatchError(t *testing.T) {
    m := NewManager(errProvider{})
    if err := m.Watch(); err == nil {
        t.Fatalf("want error")
    }
}

