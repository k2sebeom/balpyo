package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/k2sebeom/balpyo/internal/watch"
	"github.com/k2sebeom/balpyo/render/geom"
)

// `balpyo watch`의 CLI 계약이다 (ADR 0046).
//
//  1. 주소를 **찍는다** — 저자가 그 줄을 복사해 브라우저에 붙인다. `--addr 127.0.0.1:0`이
//     커널이 고른 포트로 떠야 하고(테스트가 그 줄로 붙는다), 그 말은 찍는 값이 실제로 잡은
//     주소라는 뜻이다.
//  2. 서브되는 페이지에는 리로드 클라이언트가 실린다 — 디스크의 산출물은 그대로다.
//  3. ctx가 끝나면 (Ctrl+C) 조용히 돌아온다.

func TestWatchCommandServesTheDeckWithItsReloadClient(t *testing.T) {
	noBrowser(t)
	dir := newProject(t)
	stdout := &syncBuffer{}
	var stderr syncBuffer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runWatch(ctx, []string{"--project", dir, "--addr", "127.0.0.1:0"}, stdout, &stderr) }()

	url := waitForURL(t, stdout)
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("%s에 붙을 수 없다: %v", url, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), watch.ClientMarker) {
		t.Errorf("서브된 페이지에 리로드 클라이언트가 없다:\n%s", body)
	}
	// 디스크의 산출물은 build가 쓴 그대로다 — 저자가 본 것과 배포하는 것이 같아야 한다.
	disk, err := os.ReadFile(filepath.Join(dir, "dist", "index.html"))
	if err != nil {
		t.Fatalf("산출물이 쓰이지 않았다: %v", err)
	}
	if strings.Contains(string(disk), watch.ClientMarker) {
		t.Error("리로드 클라이언트가 디스크의 산출물에 들어갔다")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("watch가 에러로 끝났다: %v\nstderr: %s", err, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Error("Ctrl+C 뒤에 watch가 돌아오지 않았다")
	}
}

// 볼 페이지가 없는 조합은 시작하기 전에 멈춘다 — pptx만 감시하는 것은 `balpyo build`를 다시
// 돌리는 일이고, 그 경우 브라우저에 보여 줄 것이 없다.
func TestWatchCommandRejectsBadInput(t *testing.T) {
	noBrowser(t)
	dir := newProject(t)
	tests := []struct {
		name string
		args []string
	}{
		{"html 없는 타깃", []string{"--project", dir, "--target", "pptx"}},
		{"알 수 없는 타깃", []string{"--project", dir, "--target", "pdf"}},
		{"남는 인자", []string{"--project", dir, "slides"}},
		{"없는 프로젝트", []string{"--project", filepath.Join(t.TempDir(), "없다")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var stdout, stderr syncBuffer
			if err := runWatch(ctx, tc.args, &stdout, &stderr); err == nil {
				t.Error("에러여야 한다")
			}
		})
	}
}

// help는 watch를 설명해야 한다 — 서브커맨드가 있는데 사용법에 없으면 아무도 찾지 못한다.
func TestUsageMentionsWatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "balpyo watch") {
		t.Errorf("help가 watch를 설명하지 않는다:\n%s", stdout.String())
	}
}

// noBrowser는 브라우저를 띄우려는 순간 실패하게 만든다 — html 전용 감시는 기동 비용을 내지 않는다.
func noBrowser(t *testing.T) {
	t.Helper()
	t.Setenv(geom.EnvChromium, filepath.Join(t.TempDir(), "없는-크로미움"))
}

// urlPattern은 watch가 찍는 주소 줄에서 URL을 뽑는다.
var urlPattern = regexp.MustCompile(`http://127\.0\.0\.1:\d+`)

// waitForURL은 watch가 주소를 찍을 때까지 기다린다.
func waitForURL(t *testing.T, out *syncBuffer) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if match := urlPattern.FindString(out.String()); match != "" {
			return match
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("watch가 주소를 찍지 않았다:\n%s", out.String())
	return ""
}

// syncBuffer는 감시 루프가 쓰는 동안 테스트가 읽을 수 있는 버퍼다.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
