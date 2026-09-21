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
})();
