package watch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ClientMarker는 주입된 리로드 클라이언트의 표시다. HTML 주석 하나이고, 주입되는 덩어리에
// **정확히 한 번** 나타난다 — 주입이 누적되지 않았음을 이 문자열을 세어 확인한다.
//
// 클래스 이름이 아니라 주석인 이유: 산출물의 CSS·JS와 절대 겹치지 않고, 저자가 개발자 도구에서
// 페이지를 볼 때 "여기부터는 내 산출물이 아니다"가 눈에 보인다.
const ClientMarker = "<!--balpyo-watch-client-->"

// OverlayID는 오버레이 요소의 id다. 산출물의 CSS와 겹치지 않게 접두사를 둔다.
const OverlayID = "balpyo-watch-overlay"

// injectClient는 문서의 마지막 `</body>` 앞에 리로드 클라이언트를 끼운다.
//
// 디스크의 파일을 고치지 않고 **응답 바이트에만** 넣는 이유: 산출물은 하나여야 한다.
// 빌드가 쓴 `index.html`에 개발용 스크립트가 섞이면 저자가 보며 쓴 페이지와 배포하는 페이지가
// 달라지고, 골든 파일도 watch 하나로 움직인다.
//
// 이미 주입된 문서는 그대로 돌려준다 — 산출물이 어쩌다 이 표시를 담고 있어도(예: watch 응답을
// 저장해 둔 파일) 클라이언트가 두 번 붙지 않는다.
func injectClient(page string, status Status) []byte {
	if strings.Contains(page, ClientMarker) {
		return []byte(page)
	}
	snippet := clientSnippet(status)
	if i := strings.LastIndex(page, "</body>"); i >= 0 {
		return []byte(page[:i] + snippet + page[i:])
	}
	// `</body>`가 없는 문서(조각 HTML)에도 클라이언트는 실려야 한다 — 끝에 붙인다.
	return []byte(page + snippet)
}

// clientSnippet은 주입되는 덩어리다: 표시 · 오버레이 스타일 · 클라이언트 스크립트.
func clientSnippet(status Status) string {
	boot, err := json.Marshal(struct {
		Generation int    `json:"generation"`
		Events     string `json:"events"`
		Status     string `json:"status"`
	}{Generation: status.Generation, Events: EventsPath, Status: StatusPath})
	if err != nil {
		// 정수와 상수 문자열뿐이므로 여기에 오지 않는다. 와도 페이지는 살아야 한다.
		boot = []byte(`{"generation":0,"events":"` + EventsPath + `","status":"` + StatusPath + `"}`)
	}
	return fmt.Sprintf("\n%s\n<style>%s</style>\n<script>%s</script>\n",
		ClientMarker, overlayCSS, fmt.Sprintf(clientJS, OverlayID, boot))
}

// overlayCSS는 오버레이의 모양이다. 슬라이드 **위에** 뜨고, 슬라이드를 밀지 않는다 —
// 저자가 보는 것은 진단이 아니라 방금 고친 슬라이드이고, 진단은 그 옆의 쪽지다.
//
// 값을 테마 토큰에서 읽지 않는 이유: 이 오버레이는 덱의 일부가 아니라 개발 도구다. 테마가
// 무엇이든(어두운 테마에서도) 같게 읽혀야 하고, 빌드가 **실패한** 상태에서도 떠야 한다 —
// 그때 페이지의 토큰은 마지막 성공본의 것이다.
const overlayCSS = `
#` + OverlayID + `{position:fixed;left:16px;bottom:16px;z-index:2147483647;
max-width:min(560px,calc(100vw - 32px));font:13px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;
color:#f8fafc;background:#0f172ae6;border-radius:8px;box-shadow:0 8px 24px #0f172a59;
backdrop-filter:blur(2px);overflow:hidden}
#` + OverlayID + `[hidden]{display:none}
#` + OverlayID + ` .bw-head{display:flex;align-items:center;gap:8px;padding:8px 12px;cursor:pointer;
user-select:none;font-weight:600}
#` + OverlayID + ` .bw-dot{width:8px;height:8px;border-radius:50%;background:#facc15;flex:none}
#` + OverlayID + `[data-state="error"] .bw-dot{background:#f87171}
#` + OverlayID + `[data-state="offline"] .bw-dot{background:#94a3b8}
#` + OverlayID + ` .bw-time{margin-left:auto;font-weight:400;color:#94a3b8}
#` + OverlayID + ` .bw-list{margin:0;padding:0 12px 10px;list-style:none;max-height:40vh;overflow:auto}
#` + OverlayID + ` .bw-list[hidden]{display:none}
#` + OverlayID + ` .bw-item{padding:4px 0;border-top:1px solid #33415580}
#` + OverlayID + ` .bw-where{color:#93c5fd}
#` + OverlayID + ` .bw-item[data-severity="warning"] .bw-where{color:#fcd34d}
`

// clientJS는 리로드 클라이언트다. %s 둘은 오버레이 id와 부트스트랩 JSON이다.
//
// EventSource를 쓰는 이유: 서버→브라우저 한 방향이면 되고, 끊기면 브라우저가 스스로
// 다시 붙는다 — watch를 재시작해도 저자가 탭을 만지지 않는다. WebSocket은 그 재연결을 우리가
// 써야 한다.
//
// 성공 빌드에 **통째로 새로고침**하는 이유: HMR은 상태를 보존하는 대신 "화면이 파일과 같다"는
// 보장을 잃는다. 이 도구에서 저자가 가장 자주 묻는 질문이 그 보장이다.
const clientJS = `
(function () {
  var ID = %q;
  var boot = %s;

  function overlay() {
    var node = document.getElementById(ID);
    if (node) return node;
    node = document.createElement("div");
    node.id = ID;
    node.hidden = true;
    node.innerHTML =
      '<div class="bw-head"><span class="bw-dot"></span>' +
      '<span class="bw-title"></span><span class="bw-time"></span></div>' +
      '<ul class="bw-list" hidden></ul>';
    node.querySelector(".bw-head").addEventListener("click", function () {
      var list = node.querySelector(".bw-list");
      if (list.children.length) list.hidden = !list.hidden;
    });
    document.body.appendChild(node);
    return node;
  }

  function show(state, title, time, items) {
    var node = overlay();
    node.dataset.state = state;
    node.querySelector(".bw-title").textContent = title;
    node.querySelector(".bw-time").textContent = time || "";
    var list = node.querySelector(".bw-list");
    list.textContent = "";
    (items || []).forEach(function (diag) {
      var item = document.createElement("li");
      item.className = "bw-item";
      item.dataset.severity = diag.severity || "error";
      var where = diag.file || "";
      if (where && diag.line) where += ":" + diag.line + (diag.column ? ":" + diag.column : "");
      if (where) {
        var span = document.createElement("span");
        span.className = "bw-where";
        span.textContent = where + " ";
        item.appendChild(span);
      }
      item.appendChild(document.createTextNode(diag.message || ""));
      list.appendChild(item);
    });
    // 에러는 펼친 채로 띄운다 — 실패한 빌드에서 저자가 한 번 더 클릭할 이유가 없다.
    list.hidden = list.children.length === 0 || state === "ok";
    node.hidden = false;
  }

  function hide() {
    var node = document.getElementById(ID);
    if (node) node.hidden = true;
  }

  function render(status) {
    var diags = status.diagnostics || [];
    var warnings = diags.filter(function (d) { return d.severity === "warning"; });
    if (!status.ok) {
      show("error", status.error || "build failed", status.builtAt, diags);
      return;
    }
    if (warnings.length) {
      show("ok", warnings.length + (warnings.length === 1 ? " warning" : " warnings"), status.builtAt, warnings);
      return;
    }
    hide();
  }

  function connect() {
    var source = new EventSource(boot.events);
    source.onmessage = function (event) {
      var status = JSON.parse(event.data);
      // 자기가 실린 세대와 다른 **성공** 빌드만 새로고침한다. 붙는 즉시 받는 현재 상태는
      // 같은 세대이므로 새로고침 고리가 돌지 않는다.
      if (status.ok && status.generation !== boot.generation) {
        location.reload();
        return;
      }
      render(status);
    };
    source.onerror = function () {
      // EventSource가 스스로 다시 붙는다 — 여기서는 화면에만 말한다.
      if (source.readyState === EventSource.CONNECTING) {
        show("offline", "lost connection to the watch server — reconnecting", "", []);
      }
    };
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", connect);
  } else {
    connect();
  }
})();
`

// placeholderPage는 산출물이 아직 없을 때 나가는 페이지다. 첫 빌드가 실패한 상태이고, 이유는
// 오버레이가 말한다 — 이 페이지는 그 오버레이를 실을 자리이자 "서버는 살아 있다"는 표시다.
func placeholderPage() string {
	return `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="utf-8">
<title>balpyo watch</title>
<style>
html,body{height:100%;margin:0}
body{display:grid;place-items:center;background:#0f172a;color:#94a3b8;
font:15px/1.6 ui-sans-serif,system-ui,-apple-system,"Apple SD Gothic Neo",sans-serif}
</style>
</head>
<body>
<p>No output yet — this page will update itself once a build succeeds.</p>
</body>
</html>
`
}
