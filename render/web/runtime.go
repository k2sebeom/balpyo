package web

import "fmt"

// 이 파일은 뷰어 런타임이다 — 산출물 index.html을 발표 가능한 웹앱으로 만드는 인라인
// JS/CSS. 슬라이드를 한 장씩 화면에 맞춰 보여주고, 하단 가운데 컨트롤 바로 넘기며,
// 발표자 노트를 패널로 열어 본다.
//
// # 문서는 두 가지 뷰를 갖는다
//
// **문서 뷰**(기본, JS 없음)는 모든 슬라이드를 논리 스테이지 크기 그대로 세로로 쌓는다.
// 지오메트리 패스와 스크린샷 하네스가 재는 배치가 이것이다 — 슬라이드 rect가 논리
// 스테이지와 1:1이어야 좌표가 그대로 PPTX로 간다 (geom.newGeometry가 확인한다).
//
// **발표자 뷰**는 런타임 JS가 로드 즉시 <html>에 ViewAttr(=PresenterView)를 켜서 들어간다.
// 발표자 뷰의 CSS는 전부 이 속성에 걸려 있으므로(viewerCSS), 속성 하나가 뷰 전환의
// 전부다: 지우면 문서 뷰로 돌아온다. 지오메트리 패스가 페이지를 열 때 정확히 그렇게 한다
// (geom.Session.Open이 DocumentViewExpr를 평가한다) — 측정과 발표가 **같은 파일**을 쓰면서도
// 측정은 스케일·겹침이 없는 배치를 본다. `balpyo server`의 발표자 뷰도 같은 속성 계약을 쓴다.
//
// 발표자 뷰가 숨기는 슬라이드는 display:none이 아니라 visibility:hidden이다 — 레이아웃이
// 남아 있어야 발표자 뷰가 켜진 채로도 rect가 0으로 무너지지 않고, 뷰 전환이 줄바꿈을
// 다시 계산하게 만들지 않는다. 화면 맞춤은 transform: scale이므로 레이아웃 크기
// (offsetWidth)는 논리 스테이지 그대로 남고, 런타임이 그 값으로 배율을 계산한다.

// ViewAttr는 <html>에 실리는 뷰 모드 속성이다. 없으면 문서 뷰다.
const ViewAttr = "data-balpyo-view"

// PresenterView는 발표자 뷰의 ViewAttr 값이다.
const PresenterView = "presenter"

// 뷰어 크롬의 클래스다. 런타임 JS가 만드는 요소이므로 문서 골든에는 나오지 않지만,
// 발표자 뷰를 조작하는 테스트(internal/testkit)가 같은 문자열로 요소를 집는다 — 식별자
// 계약(IDAttr 등)과 같은 이유로 여기서 export한다.
const (
	// ViewerBarClass는 하단 가운데 컨트롤 바다.
	ViewerBarClass = "balpyo-viewer-bar"
	// ViewerPrevClass·ViewerNextClass는 이전/다음 버튼이다.
	ViewerPrevClass = "balpyo-viewer-prev"
	ViewerNextClass = "balpyo-viewer-next"
	// ViewerCounterClass는 "3 / 12" 카운터다.
	ViewerCounterClass = "balpyo-viewer-counter"
	// ViewerNotesToggleClass는 노트 패널 토글 버튼이다.
	ViewerNotesToggleClass = "balpyo-viewer-notes-toggle"
	// ViewerNotesClass는 노트 패널이다 (숨은 슬라이드 노트의 내용을 보여준다).
	ViewerNotesClass = "balpyo-viewer-notes"
	// viewerActiveClass는 지금 보이는 슬라이드의 표식이다 (런타임이 토글한다).
	viewerActiveClass = "balpyo-viewer-active"
	// viewerEmptyClass는 노트가 없는 슬라이드에서 패널이 갖는 표식이다.
	viewerEmptyClass = "balpyo-viewer-empty"
)

// PresenterViewExpr는 발표자 뷰를 켜는 JS 표현식이다 — 런타임이 로드 시 하는 일과 같다.
// 발표자 뷰의 동작을 검증하는 테스트가 Session.Open이 되돌린 문서 뷰에서 다시 켤 때 쓴다.
func PresenterViewExpr() string {
	return fmt.Sprintf("document.documentElement.setAttribute(%q, %q)", ViewAttr, PresenterView)
}

// DocumentViewExpr는 문서 뷰로 되돌리는 JS 표현식이다 — 지오메트리 패스가 재기 전에
// 평가한다 (geom.Session.Open). 발표자 뷰 CSS는 전부 ViewAttr에 걸려 있으므로 속성을
// 지우는 것으로 충분하고, balpyo 산출물이 아닌 페이지에서도 아무 일도 하지 않아 안전하다.
func DocumentViewExpr() string {
	return fmt.Sprintf("document.documentElement.removeAttribute(%q)", ViewAttr)
}

// viewerCSS는 발표자 뷰와 뷰어 크롬의 CSS다 — baseCSS가 balpyo.base 층 안에 싣는다.
//
// 발표자 뷰 규칙은 전부 `html[data-balpyo-view="presenter"]` 아래에 있다. 게이트 밖에
// 남는 것은 (a) 문서 뷰에서 크롬을 숨기는 규칙과 (b) 게이트된 규칙이 이미 노출을 제어하는
// 크롬 내부 장식뿐이다 — 게이트 없는 규칙이 슬라이드 배치를 건드리면 측정이 그만큼 틀어진다.
//
// 크롬의 색·폰트는 테마 토큰이 아니다 (baseCSS의 뷰어 바탕색과 같은 판단): 슬라이드 밖의
// 색이라 덱의 색이 아니고, 대응하는 PPTX 개념도 없다. 폰트도 system-ui로 두어 테마 폰트
// 이름이 크롬에 새지 않게 한다.
func viewerCSS() string {
	return `
  /* ── 발표자 뷰 ── 런타임 JS가 <html>에 data-balpyo-view="presenter"를 켠다.
     속성이 없으면(JS 꺼짐, 또는 지오메트리 패스가 지운 상태) 위의 세로 스택 그대로다. */
  .balpyo-viewer-bar, .balpyo-viewer-notes { display: none; }

  html[data-balpyo-view="presenter"] { height: 100%; }
  html[data-balpyo-view="presenter"] body { height: 100%; overflow: hidden; }
  html[data-balpyo-view="presenter"] .balpyo-deck {
    position: fixed;
    inset: 0;
    padding: 0;
    display: block;
  }

  /* 슬라이드는 전부 화면 가운데 겹쳐 두고 활성 슬라이드만 보인다. display가 아니라
     visibility인 이유와 transform 스케일인 이유는 파일 머리 주석에 있다. */
  html[data-balpyo-view="presenter"] .balpyo-slide {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%) scale(var(--balpyo-view-scale, 1));
    visibility: hidden;
  }
  html[data-balpyo-view="presenter"] .balpyo-slide.balpyo-viewer-active {
    visibility: visible;
  }

  html[data-balpyo-view="presenter"] .balpyo-viewer-bar {
    display: flex;
    align-items: center;
    gap: 2px;
    position: fixed;
    left: 50%;
    bottom: 16px;
    transform: translateX(-50%);
    z-index: 10;
    padding: 4px;
    border-radius: 999px;
    background: rgba(20, 20, 22, 0.88);
    color: #ddd;
    font: 14px/1 system-ui, sans-serif;
  }
  .balpyo-viewer-bar button {
    appearance: none;
    border: 0;
    background: transparent;
    color: inherit;
    font: inherit;
    padding: 8px 12px;
    border-radius: 999px;
    cursor: pointer;
  }
  .balpyo-viewer-bar button:hover { background: rgba(255, 255, 255, 0.12); }
  .balpyo-viewer-bar button:disabled { opacity: 0.35; cursor: default; background: transparent; }
  .balpyo-viewer-bar button[aria-pressed="true"] { background: rgba(255, 255, 255, 0.18); }
  .balpyo-viewer-counter {
    padding: 0 10px;
    color: #aaa;
    font-variant-numeric: tabular-nums;
  }

  html[data-balpyo-view="presenter"] .balpyo-viewer-notes {
    display: block;
    position: fixed;
    left: 50%;
    bottom: 64px;
    transform: translateX(-50%);
    z-index: 10;
    width: min(640px, calc(100vw - 48px));
    max-height: 40vh;
    overflow-y: auto;
    padding: 14px 18px;
    border-radius: 12px;
    background: rgba(20, 20, 22, 0.92);
    color: #e6e6e6;
    font: 15px/1.6 system-ui, sans-serif;
    white-space: pre-wrap;
  }
  /* hidden 속성이 이겨야 한다 — 위의 display: block이 UA 스타일을 덮으므로 명시한다. */
  html[data-balpyo-view="presenter"] .balpyo-viewer-notes[hidden] { display: none; }
  .balpyo-viewer-notes.balpyo-viewer-empty { color: #999; font-style: italic; }
`
}

// runtimeJS는 뷰어 런타임이다 — Render가 <script>로 인라인한다 (외부 참조 없음).
//
// 덱 데이터를 보간하지 않는 **고정 문자열**이다: 슬라이드·노트는 전부 DOM에서 읽으므로
// 저자 텍스트가 스크립트로 새는 경로가 없다. 클래스·속성 리터럴은 이 패키지의 계약 상수와
// 같아야 한다 — 어긋나면 runtime_test가 잡는다 (baseCSS가 .balpyo-slide를 리터럴로 쓰는
// 것과 같은 관계다).
//
// 조작: 컨트롤 바(이전/다음/카운터/노트 토글), 키보드(화살표·Space·PageUp/Down·Home/End,
// n=노트), URL 해시 딥링크(`#/<슬라이드 id>` 또는 `#/<1-based 번호>`, PRD §9.1).
const runtimeJS = `(() => {
  "use strict";
  const root = document.documentElement;
  const slides = Array.from(document.querySelectorAll(".balpyo-slide"));
  if (slides.length === 0) return;

  const button = (cls, label, text) => {
    const b = document.createElement("button");
    b.type = "button";
    b.className = cls;
    b.setAttribute("aria-label", label);
    b.textContent = text;
    return b;
  };
  const prev = button("balpyo-viewer-prev", "Previous slide", "‹");
  const next = button("balpyo-viewer-next", "Next slide", "›");
  const notesToggle = button("balpyo-viewer-notes-toggle", "Toggle speaker notes", "Notes");
  notesToggle.setAttribute("aria-pressed", "false");
  const counter = document.createElement("span");
  counter.className = "balpyo-viewer-counter";

  const bar = document.createElement("nav");
  bar.className = "balpyo-viewer-bar";
  bar.setAttribute("aria-label", "Presentation controls");
  bar.append(prev, counter, next, notesToggle);

  const notes = document.createElement("aside");
  notes.className = "balpyo-viewer-notes";
  notes.setAttribute("aria-label", "Speaker notes");
  notes.hidden = true;

  document.body.append(notes, bar);

  let current = -1;

  // 배율은 창에 꽉 차는 fit이다. 레이아웃 크기(offsetWidth)는 transform과 무관하게 논리
  // 스테이지이므로, 잰 값을 다시 스케일하는 되먹임이 없다.
  const fit = () => {
    const scale = Math.min(
      window.innerWidth / slides[0].offsetWidth,
      window.innerHeight / slides[0].offsetHeight);
    root.style.setProperty("--balpyo-view-scale", String(scale));
  };

  const show = (index, writeHash) => {
    const target = Math.min(Math.max(index, 0), slides.length - 1);
    if (target === current) return;
    if (current >= 0) slides[current].classList.remove("balpyo-viewer-active");
    current = target;
    const slide = slides[current];
    slide.classList.add("balpyo-viewer-active");
    counter.textContent = (current + 1) + " / " + slides.length;
    prev.disabled = current === 0;
    next.disabled = current === slides.length - 1;
    const note = slide.querySelector(".balpyo-notes");
    notes.textContent = note ? note.textContent : "No notes for this slide.";
    notes.classList.toggle("balpyo-viewer-empty", !note);
    if (writeHash) {
      // 히스토리를 쌓지 않는다 — 뒤로 가기가 슬라이드를 되감으면 발표 중에 페이지를
      // 떠나는 길이 없어진다. file://에서 막는 브라우저가 있어도 조용히 넘어간다.
      try { history.replaceState(null, "", "#/" + encodeURIComponent(slide.id)); } catch (e) {}
    }
  };

  const fromHash = () => {
    let raw = location.hash.replace(/^#\/?/, "");
    try { raw = decodeURIComponent(raw); } catch (e) {}
    if (raw === "") return 0;
    const byID = slides.findIndex((s) => s.id === raw);
    if (byID >= 0) return byID;
    const n = Number(raw);
    if (Number.isInteger(n) && n >= 1 && n <= slides.length) return n - 1;
    return 0;
  };

  prev.addEventListener("click", () => show(current - 1, true));
  next.addEventListener("click", () => show(current + 1, true));
  notesToggle.addEventListener("click", () => {
    notes.hidden = !notes.hidden;
    notesToggle.setAttribute("aria-pressed", String(!notes.hidden));
  });
  window.addEventListener("hashchange", () => show(fromHash(), false));
  window.addEventListener("resize", fit);
  window.addEventListener("keydown", (e) => {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    switch (e.key) {
    case "ArrowRight": case "ArrowDown": case "PageDown": case " ":
      e.preventDefault(); show(current + 1, true); break;
    case "ArrowLeft": case "ArrowUp": case "PageUp":
      e.preventDefault(); show(current - 1, true); break;
    case "Home": e.preventDefault(); show(0, true); break;
    case "End": e.preventDefault(); show(slides.length - 1, true); break;
    case "n": case "N": notesToggle.click(); break;
    }
  });

  root.setAttribute("data-balpyo-view", "presenter");
  fit();
  // 해시가 있으면 그 슬라이드로 열되 해시는 손대지 않는다 — 사용자가 준 딥링크 형태
  // (번호든 id든)를 열자마자 바꿔치지 않는다. 해시가 없으면 첫 슬라이드고 URL은 깨끗하다.
  show(fromHash(), false);
})();`
