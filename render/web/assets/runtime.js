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

  // 진행 막대 — 지금 장이 덱의 어디인지를 화면 맨 위의 얇은 선으로 보여 준다 (발표자 뷰에서만 보인다).
  const progress = document.createElement("div");
  progress.className = "balpyo-viewer-progress";
  progress.setAttribute("aria-hidden", "true");

  document.body.append(notes, bar, progress);

  let current = -1;

  // 배율은 창에 꽉 차는 fit이다. 레이아웃 크기(offsetWidth)는 transform과 무관하게 논리
  // 스테이지이므로, 잰 값을 다시 스케일하는 되먹임이 없다.
  const fit = () => {
    const scale = Math.min(
      window.innerWidth / slides[0].offsetWidth,
      window.innerHeight / slides[0].offsetHeight);
    root.style.setProperty("--balpyo-view-scale", String(scale));
  };

  // ── 순차 등장 ──
  //
  // 지금 스텝은 런타임의 것이다: 마크업은 항목마다 번호를 나르고, CSS는
  // "번호를 가진 항목"까지만 겨눌 수 있어(수를 비교할 수 없다) 번호 ≤ 지금 스텝의 판정은
  // 여기서 한다. 스텝은 슬라이드를 옮길 때마다 새로 정해지므로 상태는 수 하나면 된다.
  const REVEAL_ATTR = "data-balpyo-reveal";
  const REVEALED_CLASS = "balpyo-viewer-revealed";
  let step = 0;

  const revealItems = (slide) => Array.from(slide.querySelectorAll("[" + REVEAL_ATTR + "]"));
  const stepOf = (el) => Number(el.getAttribute(REVEAL_ATTR)) || 0;
  const lastStep = (slide) => revealItems(slide).reduce((max, el) => Math.max(max, stepOf(el)), 0);

  // 표식은 지금 슬라이드에만 다시 칠한다 — 떠난 슬라이드의 표식은 그대로 남지만 CSS가
  // 활성 슬라이드 안에서만 그것을 보므로(viewer.css) 겹쳐 둔 다른 장의 항목이 새지 않는다.
  // 컨트롤 바의 활성 상태도 여기서 정한다: 스텝이 남아 있으면 마지막 장에서도 다음이 살아
  // 있어야 하고, 스텝을 먹은 첫 장에서는 이전이 되감을 자리가 된다.
  const applyReveal = () => {
    const slide = slides[current];
    revealItems(slide).forEach((el) => el.classList.toggle(REVEALED_CLASS, stepOf(el) <= step));
    prev.disabled = current === 0 && step === 0;
    next.disabled = current === slides.length - 1 && step === lastStep(slide);
  };

  // ── 등장 ──
  //
  // 지연은 마크업이 요소마다 나르고(data-balpyo-enter-delay) 여기서 변수(--balpyo-enter-delay)로 싣는다 — 한 번이면
  // 된다. 전환에 거는 것은 발표자 뷰의 CSS다: transition-delay를 인라인으로 걸면 길이가 0인 다른 뷰에서도
  // "지연 뒤에 바뀌는" 전환이 성립해(길이 + 지연 > 0), 문서 뷰 · 인쇄로 넘어간 순간 요소가 지연만큼 비어 있다.
  // 슬라이드에
  // 들어갈 때 표식을 떼었다가 다음 프레임에 붙이면 요소들이 시작 상태에서 끝 상태로 엇갈려 움직인다. 뒤로
  // 들어온 장(이미 말한 장)은 움직이지 않고 끝 상태로 온다.
  const ENTERED_CLASS = "balpyo-viewer-entered";
  const INSTANT_CLASS = "balpyo-viewer-instant";
  document.querySelectorAll("[data-balpyo-enter]").forEach((el) => {
    el.style.setProperty("--balpyo-enter-delay", (Number(el.getAttribute("data-balpyo-enter-delay")) || 0) + "ms");
  });
  const enter = (slide, instant) => {
    slide.classList.remove(ENTERED_CLASS);
    slide.classList.toggle(INSTANT_CLASS, instant);
    if (instant) { slide.classList.add(ENTERED_CLASS); return; }
    void slide.offsetWidth;
    requestAnimationFrame(() => requestAnimationFrame(() => slide.classList.add(ENTERED_CLASS)));
  };

  // ── 슬라이드 전환 ──
  //
  // 전환은 다음 · 이전으로 넘길 때만 있다 (딥링크 · 오버뷰 · Home/End는 곧장 간다). 들어오는 장의 전환을
  // 쓰고, 이전으로 가면 방향만 뒤집는다. 나가는 장은 전환 동안 보이게 남는다 — fade는 두 장이 엇갈려
  // 겹치고(크로스페이드), push는 두 장이 함께 밀린다. 표식은 전환 길이가 지나면 걷는다.
  const MOVE_CLASSES = ["balpyo-viewer-moving", "balpyo-viewer-leaving",
    "balpyo-viewer-from-fade", "balpyo-viewer-from-left", "balpyo-viewer-from-right"];
  const reducedMotion = () => window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const transition = (from, to, dir) => {
    const kind = to.getAttribute("data-balpyo-transition");
    if (!kind || !from || root.getAttribute("data-balpyo-view") !== "presenter" || reducedMotion()) return;
    const ms = Number(to.getAttribute("data-balpyo-transition-duration")) || 0;
    const start = kind === "push" ? (dir > 0 ? "balpyo-viewer-from-right" : "balpyo-viewer-from-left") : "balpyo-viewer-from-fade";
    const end = kind === "push" ? (dir > 0 ? "balpyo-viewer-from-left" : "balpyo-viewer-from-right") : "balpyo-viewer-from-fade";
    to.classList.add(start);
    from.classList.add("balpyo-viewer-leaving");
    void to.offsetWidth;
    [from, to].forEach((s) => { s.classList.add("balpyo-viewer-moving"); s.style.transitionDuration = ms + "ms"; });
    requestAnimationFrame(() => { to.classList.remove(start); from.classList.add(end); });
    setTimeout(() => [from, to].forEach((s) => {
      s.classList.remove(...MOVE_CLASSES);
      s.style.transitionDuration = "";
    }), ms + 50);
  };

  const show = (index, writeHash, revealed, dir) => {
    const target = Math.min(Math.max(index, 0), slides.length - 1);
    if (target === current) return;
    const from = current >= 0 ? slides[current] : null;
    if (from) from.classList.remove("balpyo-viewer-active");
    current = target;
    const slide = slides[current];
    slide.classList.add("balpyo-viewer-active");
    if (dir) transition(from, slide, dir);
    enter(slide, Boolean(revealed));
    counter.textContent = (current + 1) + " / " + slides.length;
    progress.style.width = ((current + 1) / slides.length * 100) + "%";
    // 들어가는 장의 스텝은 0이다 — 딥링크·오버뷰 타일·Home/End로 **곧장** 들어간 장은 아직
    // 말하지 않은 장이다 (뒤로 들어온 장만 전개된 상태로 온다).
    step = revealed ? lastStep(slide) : 0;
    applyReveal();
    const note = slide.querySelector(".balpyo-notes");
    notes.textContent = note ? note.textContent : "No notes for this slide.";
    notes.classList.toggle("balpyo-viewer-empty", !note);
    if (writeHash) {
      // 히스토리를 쌓지 않는다 — 뒤로 가기가 슬라이드를 되감으면 발표 중에 페이지를
      // 떠나는 길이 없어진다. file://에서 막는 브라우저가 있어도 조용히 넘어간다.
      try { history.replaceState(null, "", "#/" + encodeURIComponent(slide.id)); } catch (e) {}
    }
  };

  // 다음은 스텝을 **먼저** 먹고 그 다음에 슬라이드를 넘긴다. 뒤로는 스텝을 하나 되감고,
  // 스텝이 없으면 앞 슬라이드로 가되 그 장은 **전부 전개된** 상태로 들어간다:
  // 이미 말한 장을 다시 한 단계씩 누르게 만들지 않는다.
  const forward = () => {
    if (step < lastStep(slides[current])) { step++; applyReveal(); return; }
    show(current + 1, true, false, 1);
  };
  const backward = () => {
    if (step > 0) { step--; applyReveal(); return; }
    show(current - 1, true, true, -1);
  };

  // ── 오버뷰 ──
  //
  // 오버뷰는 발표자 뷰 안의 상태가 아니라 **같은 속성의 다른 값**이다. 드나드는
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

  // 버튼과 키는 **같은 동작**이다 (스텝을 먼저 먹는다) — 둘이 갈라지면 마우스로 넘긴
  // 사람만 순차 등장을 건너뛴다.
  prev.addEventListener("click", () => backward());
  next.addEventListener("click", () => forward());
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
      e.preventDefault(); forward(); break;
    case "ArrowLeft": case "ArrowUp": case "PageUp":
      e.preventDefault(); backward(); break;
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
