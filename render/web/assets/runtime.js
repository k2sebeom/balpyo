// 이 파일은 뷰어 런타임이다 (render/web.runtimeJS가 go:embed로 싣는다). index.html의 <body>
// 끝에 script 태그로 인라인되므로 여닫는 태그 문자열을 담을 수 없다 — 닫는 태그는 문서를 그
// 자리에서 끝내고, 여는 태그는 "런타임은 한 번만 실린다" 게이트를 속인다 (runtime_test가 막는다). 덱 데이터를 보간하지 않는 **고정 파일**이다: 슬라이드·노트는
// 전부 DOM에서 읽으므로 저자 텍스트가 스크립트로 새는 경로가 없다.
(() => {
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
  const overviewToggle = button("balpyo-viewer-overview", "Toggle slide overview", "Grid");
  overviewToggle.setAttribute("aria-pressed", "false");
  const counter = document.createElement("span");
  counter.className = "balpyo-viewer-counter";

  const bar = document.createElement("nav");
  bar.className = "balpyo-viewer-bar";
  bar.setAttribute("aria-label", "Presentation controls");
  bar.append(prev, counter, next, overviewToggle, notesToggle);

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

  // ── 오버뷰 ──
  //
  // 오버뷰는 발표자 뷰 안의 상태가 아니라 **같은 속성의 다른 값**이다 (ADR 0062). 드나드는
  // 것으로 현재 슬라이드가 바뀌지 않는다 — 오버뷰는 이동이 아니라 보기이고, 나오면 보고 있던
  // 자리로 돌아온다. 자리를 옮기는 것은 타일 클릭(과 오버뷰에서도 듣는 화살표)뿐이다.
  const overviewOn = () => root.getAttribute("data-balpyo-view") === "overview";

  // 격자는 **전부 한 화면에** 들어와야 한다: 열 수를 1..n으로 훑어 타일이 가장 커지는 값을
  // 고르고 그 폭에서 zoom을 얻는다. 간격·여백은 viewer.css와 같은 값이고, 아래쪽은 컨트롤
  // 바가 타일을 덮지 않도록 바 높이만큼 더 비운다.
  const OVERVIEW_GAP = 16, OVERVIEW_PAD = 16, OVERVIEW_BAR = 48;
  const overviewFit = () => {
    const stageWidth = slides[0].offsetWidth, stageHeight = slides[0].offsetHeight;
    const roomWidth = window.innerWidth - 2 * OVERVIEW_PAD;
    const roomHeight = window.innerHeight - 2 * OVERVIEW_PAD - OVERVIEW_BAR;
    let best = {columns: 1, width: 0};
    for (let columns = 1; columns <= slides.length; columns++) {
      const rows = Math.ceil(slides.length / columns);
      const width = Math.min(
        (roomWidth - (columns - 1) * OVERVIEW_GAP) / columns,
        ((roomHeight - (rows - 1) * OVERVIEW_GAP) / rows) * (stageWidth / stageHeight));
      if (width > best.width) best = {columns, width};
    }
    root.style.setProperty("--balpyo-overview-columns", String(best.columns));
    root.style.setProperty("--balpyo-overview-zoom", String(Math.max(best.width, 1) / stageWidth));
  };

  const setOverview = (on) => {
    if (on) overviewFit();
    root.setAttribute("data-balpyo-view", on ? "overview" : "presenter");
    overviewToggle.setAttribute("aria-pressed", String(on));
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
  overviewToggle.addEventListener("click", () => setOverview(!overviewOn()));
  // 타일 클릭은 오버뷰에서만 슬라이드를 고른다 — 발표 중의 클릭은 넘기기가 아니다
  // (링크·코드 선택을 삼키지 않는다).
  slides.forEach((slide, index) => {
    slide.addEventListener("click", () => {
      if (!overviewOn()) return;
      show(index, true);
      setOverview(false);
    });
  });
  window.addEventListener("hashchange", () => show(fromHash(), false));
  window.addEventListener("resize", () => {
    fit();
    if (overviewOn()) overviewFit();
  });
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
    case "o": case "O": e.preventDefault(); setOverview(!overviewOn()); break;
    case "Escape": if (overviewOn()) { e.preventDefault(); setOverview(false); } break;
    }
  });

  root.setAttribute("data-balpyo-view", "presenter");
  fit();
  // 해시가 있으면 그 슬라이드로 열되 해시는 손대지 않는다 — 사용자가 준 딥링크 형태
  // (번호든 id든)를 열자마자 바꿔치지 않는다. 해시가 없으면 첫 슬라이드고 URL은 깨끗하다.
  show(fromHash(), false);
})();
