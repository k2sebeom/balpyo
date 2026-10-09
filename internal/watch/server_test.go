package watch

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 감시 서버의 계약 셋이다.
//
//  1. **산출물은 하나다.** 리로드 클라이언트는 응답 바이트에만 들어가고 디스크의
//     `index.html`은 `balpyo build`가 쓴 그대로다 — 저자가 보며 쓴 페이지와 배포하는 페이지가
//     같아야 하고, 그러지 않으면 골든 9개도 watch 하나로 움직인다.
//  2. **실패한 빌드가 화면을 지우지 않는다.** 마지막 성공본이 남고 그 위에 진단이 얹힌다 —
//     방금 고친 자리를 볼 수 없으면 저자는 비교 대상을 잃는다.
//  3. **진단이 보는 자리에 있다.** 상태는 JSON 한 곳에서 나오고, SSE가 같은 값을 밀어
//     새로고침 없이 갱신된다 (D8의 절반이 "경고가 다른 창에 있다"였다).

// 루트 응답에는 클라이언트가 **주입**되고, 디스크의 파일은 움직이지 않는다.
func TestServerInjectsTheClientWithoutTouchingTheOutput(t *testing.T) {
	dir := t.TempDir()
	const page = "<!DOCTYPE html><html><body><div class=\"balpyo-deck\"></div></body></html>"
	writeFile(t, dir, web.IndexFile, page)
	srv := newTestServer(t, dir)

	body := get(t, srv, "/")
	if !strings.Contains(body, ClientMarker) {
		t.Errorf("리로드 클라이언트가 주입되지 않았다:\n%s", body)
	}
	// 주입 자리는 `</body>` **앞**이다 — 덱 런타임이 다 실린 뒤에 붙어야 한다.
	if i, j := strings.Index(body, ClientMarker), strings.LastIndex(body, "</body>"); i < 0 || i > j {
		t.Errorf("클라이언트가 </body> 앞에 없다 (marker=%d, body=%d)", i, j)
	}
	if !strings.Contains(body, `class="balpyo-deck"`) {
		t.Error("산출물의 내용이 주입 중에 잘렸다")
	}
	// 디스크는 그대로다 — 이 한 줄이 "산출물은 하나다"의 전부다.
	if got := readFile(t, dir, web.IndexFile); got != page {
		t.Errorf("디스크의 index.html이 움직였다:\n%s", got)
	}
	// 같은 페이지를 다시 요청해도 클라이언트가 **한 번만** 들어간다 (주입이 누적되지 않는다).
	if n := strings.Count(get(t, srv, "/"), ClientMarker); n != 1 {
		t.Errorf("클라이언트가 %d번 주입되었다, want 1", n)
	}
}

// HTML이 아닌 산출물은 **그대로** 나간다 — 주입은 문서에만 한다.
func TestServerServesAssetsByteForByte(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, web.IndexFile, "<html><body></body></html>")
	const png = "\x89PNG\r\n\x1a\n not really"
	writeFile(t, dir, "assets/dot.png", png)
	srv := newTestServer(t, dir)

	if got := get(t, srv, "/assets/dot.png"); got != png {
		t.Errorf("에셋이 변형되어 나갔다: %q", got)
	}
}

// 첫 빌드가 실패해 산출물이 없어도 **읽을 수 있는 페이지**가 나온다.
//
// 404를 내면 저자는 브라우저의 기본 에러 화면을 보고 원인을 터미널에서 찾아야 한다 — 진단을
// 보는 자리로 가져오는 것이 이 명령의 목적이므로, 산출물이 없는 상태도 그 자리에서 말한다.
// 그리고 이 자리표시자에도 클라이언트가 있어야 빌드가 성공한 순간 스스로 새로고침한다.
func TestServerServesAPlaceholderBeforeTheFirstSuccess(t *testing.T) {
	srv := newTestServer(t, t.TempDir())
	srv.Publish(Status{Generation: 1, Error: "슬라이드가 없다: slides/ 아래에 .slide 파일이 필요하다"})

	res := request(t, srv, "/")
	if res.code != http.StatusOK {
		t.Errorf("자리표시자 응답 코드 = %d, want 200", res.code)
	}
	if !strings.Contains(res.body, ClientMarker) {
		t.Error("자리표시자에 클라이언트가 없다 — 빌드가 성공해도 화면이 그대로다")
	}
}

// 상태는 JSON 한 곳에서 나온다 — SSE로 미는 값과 새로 붙은 클라이언트가 읽는 값이 같아야 한다.
func TestServerStatusCarriesDiagnostics(t *testing.T) {
	srv := newTestServer(t, t.TempDir())
	srv.Publish(Status{
		Generation: 3,
		OK:         false,
		Error:      "빌드 실패: 에러 1건",
		Diagnostics: []Diagnostic{{
			Severity: soml.SeverityError.String(),
			File:     "slides/001_a.slide",
			Line:     4,
			Column:   9,
			Message:  `알 수 없는 태그 <h7>`,
		}},
	})

	var got Status
	decode(t, get(t, srv, StatusPath), &got)
	if got.Generation != 3 || got.OK {
		t.Errorf("상태 = %+v, want generation 3 · 실패", got)
	}
	if len(got.Diagnostics) != 1 {
		t.Fatalf("진단 %d건, want 1", len(got.Diagnostics))
	}
	diag := got.Diagnostics[0]
	if diag.File != "slides/001_a.slide" || diag.Line != 4 || diag.Column != 9 {
		t.Errorf("진단의 자리 = %s:%d:%d, want slides/001_a.slide:4:9", diag.File, diag.Line, diag.Column)
	}
	if !strings.Contains(diag.Message, "<h7>") {
		t.Errorf("진단 메시지 = %q", diag.Message)
	}
}

// SSE는 **붙는 즉시** 현재 상태를 주고, 그 뒤 빌드마다 민다.
//
// 붙자마자 현재 상태를 주는 이유: 성공 빌드가 브라우저를 새로고침시키면 새 클라이언트가
// 붙는데, 그 클라이언트가 다음 빌드까지 아무것도 모르면 남아 있는 경고가 화면에서 사라진다.
func TestServerEventsPushEveryBuild(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, web.IndexFile, "<html><body></body></html>")
	srv := newTestServer(t, dir)
	srv.Publish(Status{Generation: 1, OK: true, Slides: 2})

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	res, err := getWithTimeout(ts.URL + EventsPath)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	reader := bufio.NewReader(res.Body)

	var first Status
	decode(t, nextEvent(t, reader), &first)
	if first.Generation != 1 || !first.OK {
		t.Errorf("붙은 직후 받은 상태 = %+v, want generation 1 · 성공", first)
	}

	srv.Publish(Status{Generation: 2, OK: false, Error: "빌드 실패: 에러 1건"})
	var second Status
	decode(t, nextEvent(t, reader), &second)
	if second.Generation != 2 || second.OK {
		t.Errorf("민 상태 = %+v, want generation 2 · 실패", second)
	}
}

// 출력 디렉토리 밖의 파일은 서브하지 않는다.
func TestServerStaysInsideTheOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, web.IndexFile, "<html><body></body></html>")
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("비밀"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, dir)

	for _, path := range []string{"/../secret.txt", "/%2e%2e/secret.txt"} {
		res := request(t, srv, path)
		if strings.Contains(res.body, "비밀") {
			t.Errorf("%s가 출력 디렉토리 밖을 서브했다", path)
		}
	}
}

// newTestServer는 산출물 디렉토리 하나를 서브하는 서버다.
func newTestServer(t *testing.T, dir string) *Server {
	t.Helper()
	return NewServer(dir)
}

type response struct {
	code int
	body string
}

func request(t *testing.T, srv *Server, path string) response {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return response{code: rec.Code, body: rec.Body.String()}
}

func get(t *testing.T, srv *Server, path string) string {
	t.Helper()
	res := request(t, srv, path)
	if res.code != http.StatusOK {
		t.Fatalf("GET %s = %d\n%s", path, res.code, res.body)
	}
	return res.body
}

func getWithTimeout(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Get(url)
}

// nextEvent는 SSE 스트림에서 `data:` 한 줄을 읽는다 (이벤트 하나 = JSON 한 줄).
func nextEvent(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("SSE 스트림을 읽을 수 없다: %v", err)
		}
		if data, ok := strings.CutPrefix(strings.TrimRight(line, "\r\n"), "data: "); ok {
			return data
		}
	}
}

func decode(t *testing.T, body string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), into); err != nil {
		t.Fatalf("상태 JSON을 읽을 수 없다 (%q): %v", body, err)
	}
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
