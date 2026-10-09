package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 감시 서버가 소유하는 경로다. 접두사를 두는 이유: 산출물에 같은 이름의 파일이 있어도
// 겹치지 않고, 저자가 주소창에서 "이건 내 산출물이 아니다"를 알아본다.
const (
	// BasePath는 서버가 소유하는 경로 접두사다.
	BasePath = "/_balpyo/"
	// StatusPath는 마지막 빌드 상태를 JSON으로 돌려준다 (새로 붙은 클라이언트의 첫 화면).
	StatusPath = BasePath + "status"
	// EventsPath는 빌드마다 같은 JSON을 미는 SSE 스트림이다.
	EventsPath = BasePath + "events"
)

// keepAlive는 SSE 스트림에 주석 한 줄을 보내는 간격이다. 중간의 프록시·브라우저가 조용한
// 연결을 끊으면 리로드가 조용히 멈추므로(그때 저자는 빌드가 안 돈다고 생각한다) 살아 있음을
// 주기적으로 말한다.
const keepAlive = 20 * time.Second

// Diagnostic은 진단 하나를 브라우저가 읽을 모양으로 옮긴 것이다.
//
// soml.Diagnostic을 그대로 직렬화하지 않는 이유: 그 타입은 값 층의 것이고 JSON 태그를 붙이면
// 그 모양이 곧 외부 계약이 된다. 여기서 한 번 옮기면 진단 타입이 바뀌어도 오버레이의 계약은
// 이 구조체에 남는다.
type Diagnostic struct {
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
}

// Status는 마지막 빌드 하나의 상태다 — 터미널에 찍히는 것과 오버레이에 뜨는 것이 같은 값이다.
type Status struct {
	// Generation은 감시가 시작된 뒤의 빌드 번호다 (1부터). 클라이언트는 자기가 실린 세대와
	// 다른 **성공** 빌드를 보면 새로고침한다 — 같은 세대의 이벤트(붙는 즉시 받는 현재 상태)로
	// 새로고침 고리가 돌지 않게 하는 기준이 이 번호다.
	Generation int `json:"generation"`
	// OK는 산출물이 쓰였는지다.
	OK bool `json:"ok"`
	// Error는 빌드가 실패한 이유 한 줄이다 (진단 에러의 요약 또는 설정·파일 에러).
	Error string `json:"error,omitempty"`
	// Diagnostics는 경고까지 포함한 전부다 — 성공한 빌드의 경고도 화면에 남는다.
	Diagnostics []Diagnostic `json:"diagnostics"`
	// Slides는 성공한 빌드의 슬라이드 수다.
	Slides int `json:"slides"`
	// BuiltAt은 빌드가 끝난 시각이다 (`15:04:05`) — 화면이 언제 것인지 알려준다.
	BuiltAt string `json:"builtAt,omitempty"`
}

// DiagnosticsOf는 값 층의 진단을 오버레이 모양으로 옮긴다.
func DiagnosticsOf(diags soml.Diagnostics) []Diagnostic {
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		item := Diagnostic{Severity: d.Severity.String(), File: d.File, Message: d.Message}
		if d.Span.Start.IsValid() {
			item.Line = d.Span.Start.Line
			item.Column = d.Span.Start.Col
		}
		out = append(out, item)
	}
	return out
}

// Server는 산출물 디렉토리를 서브하고 빌드 상태를 밀어 준다.
//
// 산출물을 **디스크에서** 읽는다 (메모리 사본이 아니다): 저자가 watch로 확인한 것이 그대로
// 배포되는 파일이어야 하고, 그 사실이 코드에서 보여야 한다.
type Server struct {
	dir string

	mu      sync.Mutex
	status  Status
	clients map[chan Status]struct{}
}

// NewServer는 산출물 디렉토리 하나를 서브하는 서버를 만든다.
func NewServer(dir string) *Server {
	return &Server{dir: dir, clients: map[chan Status]struct{}{}}
}

// Publish는 빌드 상태를 갈아 끼우고 붙어 있는 클라이언트에 민다.
func (s *Server) Publish(status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	for ch := range s.clients {
		// 낡은 상태는 버린다 — 클라이언트가 볼 값은 **마지막 것**이고, 밀다가 막히면
		// 빌드 루프가 브라우저 탭 하나에 매달린다.
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- status:
		default:
		}
	}
}

// Status는 마지막 빌드 상태다.
func (s *Server) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Handler는 서버의 라우팅이다.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(StatusPath, s.handleStatus)
	mux.HandleFunc(EventsPath, s.handleEvents)
	mux.HandleFunc("/", s.handleFile)
	return mux
}

// Listen은 주소를 잡는다. 서브보다 먼저 잡는 이유는 **실제 주소**를 호출자가 알아야 한다는
// 것이다 — 포트 0(테스트)에서는 커널이 고른 번호이고, 사람에게 찍어 줄 URL도 이 값이다.
func (s *Server) Listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("%s를 열 수 없다: %w", addr, err)
	}
	return ln, nil
}

// Serve는 ctx가 끝날 때까지 서브한다.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		// 감시가 끝나면 열린 SSE 스트림이 남아 있으므로 Shutdown(연결이 끝나기를 기다린다)
		// 대신 Close다 — 저자가 Ctrl+C를 눌렀을 때 터미널이 돌아오지 않는 것이 더 나쁘다.
		_ = srv.Close()
	}()
	err := srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	noStore(w)
	if err := json.NewEncoder(w).Encode(s.Status()); err != nil {
		return
	}
}

// handleEvents는 SSE 스트림이다. 붙는 즉시 현재 상태를 한 번 보낸다 — 성공 빌드가
// 브라우저를 새로고침시키면 새 클라이언트가 붙는데, 그가 다음 빌드까지 아무것도 모르면
// 남아 있는 경고가 화면에서 사라진다.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "이 서버는 스트리밍을 할 수 없다", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	noStore(w)

	ch := make(chan Status, 1)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	current := s.status
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	if !writeEvent(w, flusher, current) {
		return
	}
	ticker := time.NewTicker(keepAlive)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case status := <-ch:
			if !writeEvent(w, flusher, status) {
				return
			}
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// writeEvent는 상태 하나를 SSE 이벤트로 쓴다. 이벤트 하나가 JSON 한 줄인 이유: 여러 줄
// 페이로드는 `data:` 줄마다 쪼개야 하고, 그 조립을 클라이언트에 요구할 이유가 없다.
func writeEvent(w http.ResponseWriter, flusher http.Flusher, status Status) bool {
	payload, err := json.Marshal(status)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

// handleFile은 산출물 하나를 서브한다. 문서에만 리로드 클라이언트를 주입한다.
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "GET만 받는다", http.StatusMethodNotAllowed)
		return
	}
	clean := path.Clean(r.URL.Path)
	if clean == "/" || clean == "/"+web.IndexFile {
		s.serveIndex(w)
		return
	}
	rel := strings.TrimPrefix(clean, "/")
	// 출력 디렉토리 밖은 서브하지 않는다. path.Clean이 `..`을 이미 접었으므로 남은 `..`는
	// 루트를 벗어나려는 시도다.
	if rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join(s.dir, filepath.FromSlash(rel))
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		// 디렉토리 목록을 내지 않는다 — 산출물 디렉토리는 구경거리가 아니고, 목록은
		// `index.html`이 있는 자리를 헷갈리게 만든다.
		http.NotFound(w, r)
		return
	}
	noStore(w)
	http.ServeFile(w, r, full)
}

// serveIndex는 산출물 문서에 클라이언트를 주입해 보낸다. 산출물이 아직 없으면(첫 빌드가
// 실패했다) 자리표시자를 같은 방식으로 보낸다 — 그 페이지도 클라이언트를 가지므로 빌드가
// 성공한 순간 스스로 새로고침한다.
func (s *Server) serveIndex(w http.ResponseWriter) {
	status := s.Status()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	noStore(w)
	data, err := os.ReadFile(filepath.Join(s.dir, web.IndexFile))
	if err != nil {
		_, _ = w.Write([]byte(injectClient(placeholderPage(), status)))
		return
	}
	_, _ = w.Write(injectClient(string(data), status))
}

// noStore는 캐시를 끈다. 저작 루프에서 캐시된 산출물은 "고쳤는데 화면이 그대로다"가 되고,
// 그 증상은 감시기가 고장 난 것과 구별되지 않는다.
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
}
