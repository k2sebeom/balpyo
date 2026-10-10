package geom

import (
	"fmt"

	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// measurement는 브라우저가 한 번에 읽어 온 값이다 — 페이지 전체를 스크립트 한 번으로 읽는다.
//
// 요소마다 CDP 왕복을 하지 않는 이유는 속도만이 아니다. 왕복 사이에 페이지가 바뀌면(폰트 지연
// 적용, 리사이즈) 요소들의 좌표가 **서로 다른 레이아웃**에서 나온 값이 되고, 그 불일치는
// 산출물에서 "요소 하나만 어긋났다"로 나타난다. 한 번의 평가는 한 레이아웃을 본다.
//
// 필드는 스크립트가 만드는 JSON 모양이다 (measureScript) — 이름이 어긋나면 값이 조용히
// 0으로 들어오므로 태그를 손으로 맞춘다.
type measurement struct {
	// DPR은 window.devicePixelRatio다. 1이 아니면 논리 px과 실제 픽셀이 갈린다.
	DPR float64 `json:"dpr"`
	// Slides는 문서 순서의 슬라이드다.
	Slides []slideMeasurement `json:"slides"`
}

// slideMeasurement는 슬라이드 하나의 측정값이다.
type slideMeasurement struct {
	// Index는 `data-slide-index` 값이다 — 덱의 슬라이드 순서와 대조한다.
	Index int `json:"index"`
	// W/H는 브라우저가 잰 슬라이드 크기다 (CSS px). 논리 스테이지와 같아야 한다.
	W float64 `json:"w"`
	H float64 `json:"h"`
	// Content는 스테이지의 **패딩 안쪽** 사각형이다 (슬라이드 상대) — 콘텐츠 박스.
	//
	// 상수로 계산하지 않고 재는 이유: 여백은 테마가 레이아웃마다 다시 정한다 (minimal의
	// `image-full.css`는 `padding: 0`이다 — 전면 배치). 상수로 판정하면 그 레이아웃에서
	// 전면 그림마다 저자가 고칠 수 없는 경고가 나오고, 고칠 수 없는 경고는 나머지 판정
	// 전부를 함께 잃게 만든다.
	Content Rect `json:"content"`
	// Boxes는 이 슬라이드 안 요소의 측정값이다 (문서 순서, 슬라이드 상대 좌표).
	Boxes []boxMeasurement `json:"boxes"`
	// Chrome은 이 슬라이드의 덱 크롬이다 (문서 순서, 슬라이드 상대 좌표).
	//
	// Boxes와 따로 읽는 이유는 크롬이 덱 트리에 없다는 것이다 (Geometry.Chrome의 주석):
	// 같은 목록에 담으면 Refs ↔ Boxes 자리 맞대기가 크롬 수만큼 어긋난다.
	Chrome []chromeMeasurement `json:"chrome"`
}

// chromeMeasurement는 크롬 하나의 측정값이다 — kind와 rect.
type chromeMeasurement struct {
	Kind string `json:"kind"`
	Rect Rect   `json:"rect"`
}

// boxMeasurement는 요소 하나의 측정값이다.
type boxMeasurement struct {
	ID    string `json:"id"`
	El    string `json:"el"`
	Rect  Rect   `json:"rect"`
	Lines []Rect `json:"lines"`
}

// measureScript는 페이지에서 지오메트리를 읽는 스크립트다.
//
// # 좌표를 슬라이드에서 빼는 이유
//
// 요소 좌표를 문서 좌표로 넘기면 덱 컨테이너의 여백(24px)·슬라이드 간격·스크롤량이 값에 섞여
// 들어간다. 슬라이드 자신의 rect를 빼면 그 항들이 모두 상쇄되어, 두 번째 슬라이드의 첫 요소가
// 첫 번째 슬라이드의 첫 요소와 **같은 좌표**로 나온다 — 그것이 슬라이드 상대 좌표의 정의이자
// 검증 방법이다 (measure_browser_test.go). 스크린샷이 스크롤량만큼 어긋나는 것과 같은
// 종류의 실패를 좌표 쪽에서 미리 막는다.
//
// # 줄 박스를 **식별자를 갖지 않는 자손까지** 읽는 이유
//
// Range.getClientRects()는 선택 영역이 실제로 차지한 줄마다 사각형 하나를 준다 — 브라우저가
// 어디서 줄을 나눴는지 알 수 있는 유일한 값이다. 요소 **전체**를 선택하면 자기 식별자로 따로
// 측정되는 자손의 줄까지 들어와 같은 줄이 두 요소에 실리므로, 순회는 `data-balpyo-id`를 가진
// 자손에서 멈춘다. 그 밖의 자손 — 인라인 런(`<strong>`·`<em>`·`<a>`·`<span>`)과 코드
// 하이라이팅의 합성 런 — 은 **이 요소가 직접 그린 글자**다: 측정 대상은 블록
// 종류뿐이고(render/web의 측정 대상 계약) 인라인은 박스가 아니라 런이므로, 그 글자를 빼고
// 재면 잰 줄이 그림보다 좁아진다. 한 줄이 통째로 런인 경우(주석 한 줄짜리 코드) 그 줄은
// 아예 사라져 "잰 줄이 없는 자리에 그림이 있다"가 된다.
//
// 그래서 **같은 줄에 앉은 조각은 합친다**: 줄 수는 그림의 성질이고 런은 서식의 성질이라,
// `배포 <strong>자동화</strong>`가 두 줄로 세어지면 줄바꿈 판정의 좌변이 틀린다. 세로로 겹친
// 조각을 이어 붙이는 것으로 판정하는 이유는 줄 안에서 글자 크기가 다를 수 있다는 것이다
// (`<code>`가 낀 줄) — 같은 y를 요구하면 그 줄이 쪼개진다.
//
// 폭·높이가 0인 사각형은 버린다 — 태그 사이 줄바꿈에서 생긴 접힌 공백 노드가 빈 사각형을
// 만들고, 그것이 줄 수에 섞이면 줄바꿈 판정의 좌변이 틀린다.
//
// # 회전을 끄고 재는 이유
//
// 측정 직전에 **측정 뷰**를 켜고(web.MeasureView) 끝나면 되돌린다. `transform: rotate()`가
// 걸린 도형의 `getBoundingClientRect`는 회전된 외곽 bbox를 주는데(45도에서 1.41배), OOXML
// `xfrm@rot`은 회전 **전** `ext`를 중심 회전한 것이다 — 그대로 재면 좌표 대조는 통과하면서
// (측정값과 산출물이 같으므로) PPTX의 도형만 커진다. 뷰 전환이 속성 하나이므로 켜고
// 끄는 일이 이 스크립트 안에서 끝나고, 되돌리기를 `finally`에 둔 이유는 예외가 나도 페이지가
// 측정 상태로 남지 않아야 하기 때문이다 (그 상태로 찍은 스크린샷은 회전이 사라진 그림이다).
// 켜고 끄는 것과 읽는 것이 **한 번의 평가** 안에 있으므로 그 사이에 다른 레이아웃이 끼지 않는다.
const measureScript = `(() => {
  // 측정 뷰를 켠다 — 회전 선언이 무효화된 배치에서 rect를 읽는다 (위 주석).
  %s;
  // 진행 중인 전환을 끝낸다 — 뷰어 런타임은 열리자마자 발표자 뷰에서 등장(enter)을 시작하고, 뷰를 바꿔도
  // 이미 시작된 전환은 남을 수 있다 (전환은 캐스케이드에서 !important보다 위다). 등장의 시작 상태
  // (떠오르기의 24px 등)를 재면 빌드마다 좌표가 달라진다. 취소하면 끝 상태 — 배치 그대로 — 가 남는다.
  document.getAnimations().forEach((a) => a.cancel());
  try {
    const lineRects = (el) => {
      const range = document.createRange();
      const rects = [];
      // 순회는 식별자를 가진 자손에서 멈춘다 — 그 자손은 자기 박스로 따로 측정된다 (위 주석).
      const walk = (node) => {
        for (const child of node.childNodes) {
          if (child.nodeType === Node.TEXT_NODE) {
            range.selectNodeContents(child);
            for (const r of range.getClientRects()) {
              if (r.width > 0 && r.height > 0) rects.push(r);
            }
            continue;
          }
          if (child.nodeType !== Node.ELEMENT_NODE) continue;
          if (child.hasAttribute(%q)) continue;
          walk(child);
        }
      };
      walk(el);
      // 같은 줄의 조각을 합친다 — 줄 수는 그림의 성질이므로 인라인 런이 줄을 늘리면 안 된다.
      // 판정은 세로 겹침이 작은 쪽 높이의 절반을 넘는가다: 줄 안의 작은 글자(code 조각)는
      // 큰 줄 상자 안에 들어앉아 거의 전부 겹치고, 이웃한 두 줄은 겹쳐도 그만큼은 아니다.
      rects.sort((a, b) => (a.top - b.top) || (a.left - b.left));
      const lines = [];
      for (const r of rects) {
        const last = lines[lines.length - 1];
        const overlap = last ? Math.min(last.bottom, r.bottom) - Math.max(last.top, r.top) : 0;
        if (last && overlap > Math.min(last.bottom - last.top, r.bottom - r.top) / 2) {
          last.top = Math.min(last.top, r.top);
          last.bottom = Math.max(last.bottom, r.bottom);
          last.left = Math.min(last.left, r.left);
          last.right = Math.max(last.right, r.right);
          continue;
        }
        lines.push({top: r.top, bottom: r.bottom, left: r.left, right: r.right});
      }
      return lines.map((l) => ({x: l.left, y: l.top, width: l.right - l.left, height: l.bottom - l.top}));
    };
    // 콘텐츠 박스는 스테이지의 **패딩 안쪽**이다 — 여백을 레이아웃마다 테마가 정하므로
    // (slideMeasurement.Content의 주석) 상수가 아니라 계산된 스타일에서 읽는다. 테두리를
    // 함께 빼는 이유는 그것도 내용이 놓일 수 없는 띠라는 것이다. 스테이지가 없으면(있어야
    // 하지만) 0을 주고 판정 쪽이 기본 여백으로 물러선다.
    const contentRect = (stage, sr) => {
      if (!stage) return {x: 0, y: 0, w: 0, h: 0};
      const r = stage.getBoundingClientRect();
      const cs = getComputedStyle(stage);
      const n = (v) => parseFloat(v) || 0;
      const l = n(cs.borderLeftWidth) + n(cs.paddingLeft);
      const t = n(cs.borderTopWidth) + n(cs.paddingTop);
      const rt = n(cs.borderRightWidth) + n(cs.paddingRight);
      const b = n(cs.borderBottomWidth) + n(cs.paddingBottom);
      return {x: r.x - sr.x + l, y: r.y - sr.y + t, w: r.width - l - rt, h: r.height - t - b};
    };
    const slides = Array.from(document.querySelectorAll(%q));
    return {
      dpr: window.devicePixelRatio,
      slides: slides.map((slide) => {
        const sr = slide.getBoundingClientRect();
        const rel = (r) => ({x: r.x - sr.x, y: r.y - sr.y, w: r.width, h: r.height});
        const raw = slide.getAttribute(%q);
        const index = raw === null ? -1 : Number(raw);
        return {
          index: Number.isInteger(index) ? index : -1,
          w: sr.width,
          h: sr.height,
          content: contentRect(slide.querySelector(%q), sr),
          boxes: Array.from(slide.querySelectorAll(%q)).map((el) => ({
            id: el.getAttribute(%q) || "",
            el: el.getAttribute(%q) || "",
            rect: rel(el.getBoundingClientRect()),
            lines: lineRects(el).map(rel),
          })),
          // 크롬은 따로 집는다 — 덱 트리에 없으므로 요소 목록에 섞이면 안 된다
          // (slideMeasurement.Chrome의 주석). 줄 박스는 읽지 않는다: 크롬은 한 줄이고
          // (CSS의 nowrap), 글줄 넘침 판정의 대상도 아니다.
          chrome: Array.from(slide.querySelectorAll(%q)).map((el) => ({
            kind: el.getAttribute(%q) || "",
            rect: rel(el.getBoundingClientRect()),
          })),
        };
      }),
    };
  } finally {
    // 문서 뷰로 되돌린다 — 페이지가 측정 상태로 남으면 그 뒤의 스크린샷이 산출물이 아닌
    // 측정 배치를 찍는다 (같은 세션이 재고 찍는다 — internal/testkit).
    %s;
  }
})()`

// measureExpr는 식별자 계약과 뷰 전환 표현식을 스크립트에 채운다.
//
// 선택자·속성 이름·뷰 전환을 여기서 리터럴로 쓰지 않는 이유: 계약의 양쪽이 같은 문자열을
// 써야 한다 (render/web이 소유). 이름을 바꿨을 때 한쪽만 고치면 "요소를 못 찾았다"가 아니라
// "좌표가 하나도 없다"로 나타난다.
func measureExpr() string {
	return fmt.Sprintf(measureScript,
		web.MeasureViewExpr(), web.IDAttr,
		web.SlideSelector(), web.SlideIndexAttr, web.StageSelector(),
		web.ElementSelector(), web.IDAttr, web.ElAttr,
		web.ChromeSelector(), web.ChromeKindAttr,
		web.DocumentViewExpr())
}

// Measure는 렌더된 HTML을 헤드리스 브라우저에 올려 요소별 좌표를 읽는다.
//
// 브라우저는 이 호출 안에서 띄우고 닫는다 — 한 번의 빌드는 한 번만 잰다. 여러 번 재는 쪽
// (dev 서버)은 Session을 직접 만들어 Session.Measure를 쓴다.
//
// htmlPath는 웹 렌더러가 낸 index.html이어야 한다. 사용자가 받는 것과 다른 바이트를 재면
// 좌표는 정확한데 산출물만 어긋나는 상태가 된다.
func Measure(deck *soml.Deck, htmlPath string) (*Geometry, error) {
	session, err := Launch("")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	return session.Measure(deck, htmlPath)
}

// Measure는 열려 있는 세션으로 지오메트리를 읽는다.
func (s *Session) Measure(deck *soml.Deck, htmlPath string) (*Geometry, error) {
	if deck == nil {
		return nil, fmt.Errorf("deck is nil")
	}
	if err := s.Open(htmlPath); err != nil {
		return nil, err
	}
	var m measurement
	if err := s.Eval(measureExpr(), &m); err != nil {
		return nil, fmt.Errorf("cannot read the geometry of %s: %w", htmlPath, err)
	}
	g, err := newGeometry(deck, m)
	if err != nil {
		// 어긋남 진단은 어느 파일을 잰 것인지까지 말해야 쓸모가 있다 — 빌드 로그에서 이
		// 에러만 보고 원인을 찾게 된다.
		return nil, fmt.Errorf("%s: %w", htmlPath, err)
	}
	// 글꼴은 좌표를 읽은 **같은 문서**에서 묻는다 — 그 좌표를 만든 글꼴 해석이 이 문서의 것이다.
	missing, err := s.MissingFonts(ThemeFontProbes(deck.Theme))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", htmlPath, err)
	}
	g.MissingFonts = missing
	return g, nil
}

// ThemeFontProbes는 테마의 글꼴 토큰에서 물어볼 글꼴을 뽑는다 (토큰 순서, latin → ea).
//
// 토큰을 전부 묻는다 — 덱이 그 역할을 지금 쓰는지는 보지 않는다. 테마의 글꼴은 PPTX 테마
// 파트에 통째로 실려 저자가 PowerPoint에서 고쳐 쓰는 순간에도 쓰이고, 덱이 쓰는지를 따지면
// 코드 블록 하나를 더하는 편집이 처음 보는 경고를 낸다. 경고는 테마의 사실이다.
func ThemeFontProbes(theme soml.Theme) []FontProbe {
	var probes []FontProbe
	for _, token := range theme.Fonts {
		if token.Latin != "" {
			probes = append(probes, FontProbe{Family: token.Latin, Sample: LatinSample})
		}
		if token.EastAsian != "" {
			probes = append(probes, FontProbe{Family: token.EastAsian, Sample: EastAsianSample})
		}
	}
	return probes
}
