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
	// Boxes는 이 슬라이드 안 요소의 측정값이다 (문서 순서, 슬라이드 상대 좌표).
	Boxes []boxMeasurement `json:"boxes"`
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
// 검증 방법이다 (measure_browser_test.go). Iteration 0에서 스크린샷이 스크롤량만큼 어긋난
// 것과 같은 종류의 실패를 좌표 쪽에서 미리 막는다.
//
// # 줄 박스를 요소의 **직계 문자 노드**에서만 읽는 이유
//
// Range.getClientRects()는 선택 영역이 실제로 차지한 줄마다 사각형 하나를 준다 — 브라우저가
// 어디서 줄을 나눴는지 알 수 있는 유일한 값이다. 요소 전체를 선택하면 자손 요소의 줄까지
// 함께 들어오는데, 자손은 자기 식별자로 따로 측정되므로 그러면 같은 줄이 두 요소에 실린다.
// 직계 문자 노드만 재면 각 요소의 Lines가 그 요소가 직접 그린 글자에 대응한다.
//
// 폭·높이가 0인 사각형은 버린다 — 태그 사이 줄바꿈에서 생긴 접힌 공백 노드가 빈 사각형을
// 만들고, 그것이 줄 수에 섞이면 줄바꿈 판정의 좌변이 틀린다.
//
// # 회전을 끄고 재는 이유
//
// 측정 직전에 **측정 뷰**를 켜고(web.MeasureView) 끝나면 되돌린다. `transform: rotate()`가
// 걸린 도형의 `getBoundingClientRect`는 회전된 외곽 bbox를 주는데(45도에서 1.41배), OOXML
// `xfrm@rot`은 회전 **전** `ext`를 중심 회전한 것이다 — 그대로 재면 좌표 대조는 통과하면서
// (측정값과 산출물이 같으므로) PPTX의 도형만 커진다. 뷰 전환이 속성 하나이므로(ADR 0006) 켜고
// 끄는 일이 이 스크립트 안에서 끝나고, 되돌리기를 `finally`에 둔 이유는 예외가 나도 페이지가
// 측정 상태로 남지 않아야 하기 때문이다 (그 상태로 찍은 스크린샷은 회전이 사라진 그림이다).
// 켜고 끄는 것과 읽는 것이 **한 번의 평가** 안에 있으므로 그 사이에 다른 레이아웃이 끼지 않는다.
const measureScript = `(() => {
  // 측정 뷰를 켠다 — 회전 선언이 무효화된 배치에서 rect를 읽는다 (위 주석).
  %s;
  try {
    const lineRects = (el) => {
      const range = document.createRange();
      const rects = [];
      for (const node of el.childNodes) {
        if (node.nodeType !== Node.TEXT_NODE) continue;
        range.selectNodeContents(node);
        for (const r of range.getClientRects()) {
          if (r.width > 0 && r.height > 0) rects.push(r);
        }
      }
      return rects;
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
          boxes: Array.from(slide.querySelectorAll(%q)).map((el) => ({
            id: el.getAttribute(%q) || "",
            el: el.getAttribute(%q) || "",
            rect: rel(el.getBoundingClientRect()),
            lines: lineRects(el).map(rel),
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
		web.MeasureViewExpr(),
		web.SlideSelector(), web.SlideIndexAttr, web.ElementSelector(), web.IDAttr, web.ElAttr,
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
		return nil, fmt.Errorf("덱이 nil이다")
	}
	if err := s.Open(htmlPath); err != nil {
		return nil, err
	}
	var m measurement
	if err := s.Eval(measureExpr(), &m); err != nil {
		return nil, fmt.Errorf("%s의 지오메트리를 읽을 수 없다: %w", htmlPath, err)
	}
	g, err := newGeometry(deck, m)
	if err != nil {
		// 어긋남 진단은 어느 파일을 잰 것인지까지 말해야 쓸모가 있다 — 빌드 로그에서 이
		// 에러만 보고 원인을 찾게 된다.
		return nil, fmt.Errorf("%s: %w", htmlPath, err)
	}
	return g, nil
}
