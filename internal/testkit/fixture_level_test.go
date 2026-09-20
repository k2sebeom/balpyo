package testkit

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 픽스처 수준 검증. PPTX 구조의 단위 단정은 render/pptx의 테스트가 이미 갖고 있다 —
// 여기서는 실제 픽스처의 산출물 전체를 놓고, 골든에 들어간 파트 목록·좌표·토큰 보존이
// 기대와 같은지, 그리고 두 렌더 엔진이 같은 자리에서 줄을 나누는지를 본다 (PRD §13-1).
const (
	// textFixtureName은 Iteration 1의 <text> 전용 픽스처다.
	textFixtureName = "01_text"
	// textSubsetFixtureName은 Iteration 2의 HTML 텍스트 서브셋 픽스처다 (블록·리스트·
	// 인라인 서식·pre — 항목 02/08이 활성화했다).
	textSubsetFixtureName = "02_text_subset"
	// layoutFixtureName은 Iteration 3의 레이아웃 태그 픽스처다 (row/col/grid/spacer +
	// 자식 배치 속성 + `style` 탈출구 — 항목 03/05가 활성화했다). 도형은 여전히 텍스트
	// 박스뿐이라 아래 검증들(줄 수 대조·잉크 띠의 단색 배경 가정·schemeClr)이 그대로
	// 성립한다 — 이 픽스처가 새로 시험하는 것은 **한 줄에 여러 박스가 앉는 배치**다.
	layoutFixtureName = "03_layout"
	// shapeFixtureName은 Iteration 4의 도형 픽스처다 (프리셋 12종 + 외관 + 도형 안 텍스트
	// — 항목 04/06이 활성화했다). 여기서 처음으로 **면이 그려지므로** 위 셋에서 성립했던
	// 두 전제가 깨진다: 슬라이드 파트에 리터럴 `srgbClr`가 0건이라는 것(리터럴 색이 어휘에
	// 들어왔다 — fixtureLiteralColors), 그리고 "가장 흔한 색이 배경"이라는 잉크 띠의
	// 전제(칠한 도형이 큰 면을 덮는다 — inkBandSkipSlides).
	shapeFixtureName = "04_shape"
	// imageFixtureName은 Iteration 5의 그림 픽스처다 (`fit` 2종 + 외관 + 포맷·파트 재사용 +
	// variant — 항목 05/05가 활성화했다). 여기서 처음으로 **골든에 바이너리가 들어간다**:
	// 웹 산출 에셋(`assets/*`)과 PPTX 미디어 파트(`ppt/media/*`)가 같은 그림을 두 번 담는다
	// (두 산출물이 각자 온전해야 하므로 정상이다 — 에셋 파일이 작아야 하는 이유가 여기다).
	// 도형 픽스처가 깨뜨린 두 전제(리터럴 색·잉크 띠의 단색 배경)에서 이 픽스처는 앞쪽만
	// 되돌린다: 그림의 테두리도 토큰 색이라 슬라이드 파트에 `srgbClr`가 0건이지만
	// (fixtureLiteralColors에 행이 없다), 큰 색면은 도형보다 오히려 넓다 — inkBandSkipSlides가
	// 그 사실을 슬라이드마다 적는다.
	imageFixtureName = "05_image"
	// stackFixtureName은 Iteration 6의 겹침 픽스처다 (`<stack>` + 자식 배치 `at`·`x`·`y` +
	// `<slide layout="canvas">` — 항목 06이 활성화했다). 이 픽스처가 새로 시험하는 것은
	// **겹친 것이 두 타깃에서 같은 순서로 겹치는가**다: 문서 순서가 곧 z-순서라는 계약을
	// (PRD §13-3) spTree 순서 전수(TestStackFixtureZOrderIsDocumentOrder)와 겹친 영역 크롭
	// 비교(TestStackFixtureOverlapRegions)가 판정한다.
	//
	// 도형·그림 픽스처가 깬 두 전제(리터럴 색·잉크 띠의 단색 배경)에서 이 픽스처는 앞쪽을
	// 되돌린다: 모든 색이 토큰이라 슬라이드 파트에 리터럴 `srgbClr`가 0건이다
	// (fixtureLiteralColors에 행이 없다). 잉크 띠는 04·05와 같은 이유로 깨지지만 — 겹침
	// 슬라이드에는 **띠의 주인이 없다** (겹친 자리에 여러 도형이 포개진다) — 이번에는 겹치지
	// 않은 텍스트 슬라이드(010 content-explicit)가 하나 있어 그 슬라이드는 판정에 남긴다
	// (inkBandSkipSlides는 001~009만 적는다).
	//
	// **캔버스 슬라이드(008·009)가 파트를 늘리지 않았다는 것**이 goldenPartLists로 확인된다
	// (항목 02·05의 판단): `layout` 속성은 렌더러에 닿을 뿐 파트 구조를 바꾸지 않는다.
	stackFixtureName = "06_stack"
)

// fixtureNames는 픽스처 수준 검증(파트 목록·색 계약·좌표 대조·도형 수 등식·잉크 띠)이 도는
// 픽스처 전부다 — 새 픽스처가 생기면 여기 더해 같은 계약을 받게 한다.
//
// 이 표와 아래 검사들의 이름에 "text"가 없는 이유(Iteration 4 항목 06의 판단): 검사가 재는
// 것은 텍스트가 아니라 **픽스처 하나의 산출물 전체**다. 이름이 `textFixtureNames`로 남아
// 있으면 도형 픽스처를 넣는 것이 표의 뜻을 어기는 일처럼 보이고, 그러면 다음 사람이 같은
// 계약을 받을 픽스처를 두 번째 표에 적는다.
var fixtureNames = []string{textFixtureName, textSubsetFixtureName, layoutFixtureName, shapeFixtureName,
	imageFixtureName, stackFixtureName}

// fixtureByName은 픽스처 러너와 같은 수집 경로로 픽스처를 집는다 — 경로를 직접 조립하면
// 러너가 못 보는 픽스처를 여기만 보는 상태가 생길 수 있다.
func fixtureByName(t *testing.T, name string) Fixture {
	t.Helper()
	for _, fixture := range Fixtures(t) {
		if fixture.Name == name {
			return fixture
		}
	}
	t.Fatalf("픽스처 %s가 수집되지 않았다", name)
	return Fixture{}
}

// textFixture는 01_text다 (presenter_test 등 Iteration 1 전용 검증이 쓴다).
func textFixture(t *testing.T) Fixture {
	t.Helper()
	return fixtureByName(t, textFixtureName)
}

// goldenPartLists는 픽스처별 골든 파트 목록의 전수 기대값이다. 파트별 XML은 골든 비교가
// 보지만, "무엇이 있어야 하고 무엇이 없어야 하는가"는 목록이 말한다 — 특히 노트 파트는
// 노트가 있는 슬라이드에만 있어야 하고(01_text는 1·3, 02_text_subset은 5),
// diagnostics.txt는 경고가 있을 때만 생긴다: 01_text는 경고 없이 빌드되어야 하고,
// 02_text_subset은 모르는 variant 슬라이드가 경고를 내므로 **있어야** 한다 — 그 골든이
// 경고 경로(문구·위치)의 유일한 회귀 테스트다 (fixtures README).
var goldenPartLists = map[string][]string{
	textFixtureName: {
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		"pptx/ppt/notesSlides/_rels/notesSlide1.xml.rels",
		"pptx/ppt/notesSlides/_rels/notesSlide3.xml.rels",
		"pptx/ppt/notesSlides/notesSlide1.xml",
		"pptx/ppt/notesSlides/notesSlide3.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/theme/theme1.xml",
	},
	textSubsetFixtureName: {
		"diagnostics.txt",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		"pptx/ppt/notesSlides/_rels/notesSlide5.xml.rels",
		"pptx/ppt/notesSlides/notesSlide5.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		// 007은 목록 컨테이너의 외관(P0.3)을 시험한다 — 파트는 슬라이드 하나만 늘린다.
		"pptx/ppt/slides/_rels/slide7.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/slides/slide7.xml",
		"pptx/ppt/theme/theme1.xml",
	},
	layoutFixtureName: {
		// 007이 `style`에 외관을 써서 경고 2건을 내므로 diagnostics.txt가 **있어야** 한다 —
		// 그 골든이 경고 경로(문구·위치·순서)의 유일한 회귀 테스트다.
		"diagnostics.txt",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		// 노트는 001에만 있다 — 레이아웃 컨테이너가 노트 파트를 만들지 않는다는 사실도
		// 이 목록이 지킨다.
		"pptx/ppt/notesSlides/_rels/notesSlide1.xml.rels",
		"pptx/ppt/notesSlides/notesSlide1.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		"pptx/ppt/slides/_rels/slide7.xml.rels",
		// 008(슬라이드 justify)이 파트를 늘리지 않았다 — 세로 분배는 스테이지의 인라인
		// 선언이고 잰 좌표만 움직인다 (ADR 0044 D17).
		"pptx/ppt/slides/_rels/slide8.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/slides/slide7.xml",
		"pptx/ppt/slides/slide8.xml",
		"pptx/ppt/theme/theme1.xml",
	},
	shapeFixtureName: {
		// 008이 모르는 variant 둘(도형 + 텍스트)에 경고를 내므로 diagnostics.txt가
		// **있어야** 한다 — 도형 variant 경로의 유일한 회귀 테스트다 (문구는 텍스트
		// 경고와 바이트가 같고 나열된 목록만 다르다 — core.warnUnknownVariant).
		"diagnostics.txt",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		// 노트는 006(카드 그리드)에만 있다 — 도형이 노트 파트를 만들지 않는다는 사실도
		// 이 목록이 지킨다.
		"pptx/ppt/notesSlides/_rels/notesSlide6.xml.rels",
		"pptx/ppt/notesSlides/notesSlide6.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		"pptx/ppt/slides/_rels/slide7.xml.rels",
		"pptx/ppt/slides/_rels/slide8.xml.rels",
		// 009·010은 프리셋 12종의 **도형 안 텍스트**다 (텍스트 영역 — ADR 0040): 슬라이드
		// 파트 둘을 더하고 다른 파트는 늘리지 않는다.
		"pptx/ppt/slides/_rels/slide9.xml.rels",
		"pptx/ppt/slides/_rels/slide10.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/slides/slide7.xml",
		"pptx/ppt/slides/slide8.xml",
		"pptx/ppt/slides/slide9.xml",
		"pptx/ppt/slides/slide10.xml",
		"pptx/ppt/theme/theme1.xml",
	},
	imageFixtureName: {
		// **바이너리 골든이 처음 들어온 자리다.** 웹 산출 에셋 4장과 PPTX 미디어 파트 4장이
		// 같은 그림을 두 번 담는다 — 두 산출물이 각자 온전해야 하기 때문이고, 목록이 그
		// 중복을 눈에 보이게 갖고 있는 것이 정상이다 (골든이 커지는 대가는 에셋을 작게
		// 만들어 치른다: 네 장 합쳐 6.3KB).
		//
		// **파트 목록 전수 검사가 XML만 가정하지 않는지도 이 픽스처가 처음 시험한다** —
		// 목록은 골든 디렉토리를 순회해 만들므로 확장자를 보지 않는다 (AssertGoldenTree도
		// 바이너리를 바이트로 비교한다).
		"assets/dot.png",
		"assets/photo.jpg",
		"assets/tall.png",
		"assets/wide.png",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		// 미디어 파트는 **에셋 표 순서**로 번호가 붙고 확장자는 포맷에서 온다
		// (image1=wide.png, image2=tall.png, image3=dot.png, image4=photo.jpg —
		// `.jpg`가 하나뿐이라 Content_Types의 Default도 포맷당 하나다). 덱이 `wide.png`를
		// 다섯 번 쓰는데 파트가 하나인 것이 중복 제거 계약의 실물이다.
		"pptx/ppt/media/image1.png",
		"pptx/ppt/media/image2.png",
		"pptx/ppt/media/image3.png",
		"pptx/ppt/media/image4.jpg",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		// 노트는 004(그림+텍스트)에만 있다 — 그 슬라이드가 노트·하이퍼링크·미디어를 한
		// rels에 함께 실은 유일한 자리이고(rId2 노트 · rId3 그림 · rId4 링크), 관계 번호가
		// 어긋날 수 있는 곳이 실물에서 여기뿐이다.
		"pptx/ppt/notesSlides/_rels/notesSlide4.xml.rels",
		"pptx/ppt/notesSlides/notesSlide4.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		"pptx/ppt/slides/_rels/slide7.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/slides/slide7.xml",
		"pptx/ppt/theme/theme1.xml",
		// diagnostics.txt는 **없다** — 그림 픽스처는 경고 없이 빌드되어야 한다 (모르는
		// variant를 쓰는 슬라이드가 없다: 007은 테마에 있는 `framed`를 쓴다).
	},
	stackFixtureName: {
		// 그림은 두 슬라이드(002 배지 오버레이·006 카드 그리드)가 같은 `dot.png`를 쓰지만
		// 웹 에셋은 한 장, PPTX 미디어 파트도 하나다 (중복 제거 — image1.png). 바이너리
		// 골든이 이 픽스처에서도 한 쌍뿐인 이유가 그것이다.
		"assets/dot.png",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/media/image1.png",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		// **노트 파트가 없다** — 겹침 픽스처의 어느 슬라이드도 `<note>`를 갖지 않는다.
		// notesMaster는 노트가 있을 때만 생기므로(01~05가 노트 있는 슬라이드에서 그것을
		// 보였다) 노트 없는 덱은 notesMasters/notesSlides가 통째로 빠진다.
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		"pptx/ppt/slides/_rels/slide7.xml.rels",
		"pptx/ppt/slides/_rels/slide8.xml.rels",
		"pptx/ppt/slides/_rels/slide9.xml.rels",
		"pptx/ppt/slides/_rels/slide10.xml.rels",
		"pptx/ppt/slides/_rels/slide11.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/slides/slide7.xml",
		"pptx/ppt/slides/slide8.xml",
		"pptx/ppt/slides/slide9.xml",
		"pptx/ppt/slides/slide10.xml",
		"pptx/ppt/slides/slide11.xml",
		"pptx/ppt/theme/theme1.xml",
		// **캔버스 슬라이드(008·009·011)가 파트를 늘리지 않았다** — layout 속성은 슬라이드 파트
		// 하나에 담기고 새 마스터·레이아웃을 만들지 않는다 (slideLayout1 하나뿐이다).
		// diagnostics.txt도 **없다** — 겹침 픽스처는 경고 없이 빌드되어야 한다.
	},
}

// 골든에 들어간 파트 목록을 픽스처별로 통째로 단정한다.
func TestFixtureGoldenPartList(t *testing.T) {
	for _, name := range fixtureNames {
		t.Run(name, func(t *testing.T) {
			want := goldenPartLists[name]
			var got []string
			goldenDir := fixtureByName(t, name).GoldenDir()
			err := filepath.Walk(goldenDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				rel, err := filepath.Rel(goldenDir, path)
				if err != nil {
					return err
				}
				got = append(got, filepath.ToSlash(rel))
				return nil
			})
			if err != nil {
				t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
			}

			wantSet := map[string]bool{}
			for _, name := range want {
				wantSet[name] = true
			}
			for _, name := range got {
				if !wantSet[name] {
					t.Errorf("기대하지 않은 골든 파트: %s", name)
				}
				delete(wantSet, name)
			}
			for name := range wantSet {
				t.Errorf("골든에 파트가 없다: %s", name)
			}
		})
	}
}

// slideCounts는 픽스처별 슬라이드 수다 (골든 슬라이드 파트 순회용).
var slideCounts = map[string]int{
	textFixtureName:       3,
	textSubsetFixtureName: 6,
	layoutFixtureName:     8,
	shapeFixtureName:      8,
	imageFixtureName:      7,
	stackFixtureName:      11,
}

// fixtureLiteralColors는 슬라이드 파트에 **리터럴 색으로 나와야 하는** 값의 전수 목록이다
// (`<픽스처>/<슬라이드 id>` → `srgbClr@val` 값, 중복 없이 정렬). 표에 없는 슬라이드는
// 리터럴이 0건이어야 한다.
//
// 왜 표가 필요해졌는가: Iteration 3까지 이 계약은 "슬라이드 파트에 srgbClr 0건"이었고,
// 리터럴 색이 어휘에 없던 그 세 iteration에서는 그 문장이 **"토큰이 눕지 않는다"와 같은
// 문장**이었다. Iteration 4가 `fill="#0F65F9"`를 열면서 두 문장이 갈라진다: 리터럴이
// `srgbClr`로 나오는 것은 **정상**이고(render/pptx의 단위 테스트가 양방향으로 단정한다),
// 토큰이 `srgbClr`로 나오는 것은 **제품 계약 위반**이다 (PRD §9.3 ② — 그 도형은
// PowerPoint에서 테마 색 편집을 따라가지 못한다).
//
// 슬라이드를 예외로 빼는 길(inkBandSkipSlides의 선례)을 버린 이유: 그러면 리터럴이 있는
// 슬라이드에서는 계약이 아예 없어지고, 같은 슬라이드의 **토큰 색**(003의 `color="bg"`)이
// 눕는 것을 아무것도 보지 않는다. 값을 표에 적으면 예상 밖의 리터럴은 "표에 없는 값"으로,
// 토큰의 눕음은 "표에 없는 값이 토큰의 RGB와 같음"으로 함께 걸린다 — 계약이 오히려 강해진다.
var fixtureLiteralColors = map[string][]string{
	// 003_literal_colors: `fill="#0F65F9"`(도형 채움)과 `border="2 solid #6B7280"`(테두리).
	// 0F65F9는 accent1과 **같은 값**이라 눈으로는 구별되지 않는다 — 그것이 이 슬라이드가
	// 있는 이유다: 같은 색이 어떻게 선언되었는지에 따라 산출물이 갈라진다.
	shapeFixtureName + "/shape-literal-colors": {"0F65F9", "6B7280"},
}

// 슬라이드 파트의 색은 **토큰으로 선언한 것은 schemeClr 참조로, 리터럴로 선언한 것만
// srgbClr로** 나와야 한다. 토큰이 어딘가에서 RGB로 눕으면 그 도형은 테마 색 편집을 따라가지
// 못한다 (제품 핵심 계약, PRD §9.3 ②) — 자동 검사로 그것을 보는 자리는 여기뿐이다
// (나머지 절반, 즉 PowerPoint에서 실제로 따라오는지는 사람이 리뷰에서 본다).
func TestFixtureGoldenSlideColorsKeepTokenReferences(t *testing.T) {
	for _, name := range fixtureNames {
		t.Run(name, func(t *testing.T) {
			goldenDir := fixtureByName(t, name).GoldenDir()
			for i := 1; i <= slideCounts[name]; i++ {
				slide := fmt.Sprintf("slide%d.xml", i)
				data, err := os.ReadFile(filepath.Join(goldenDir, "pptx", "ppt", "slides", slide))
				if err != nil {
					t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
				}
				xml := string(data)
				if !strings.Contains(xml, "<a:schemeClr val=") {
					t.Errorf("%s에 schemeClr 참조가 없다", slide)
				}
				// 슬라이드 id로 표를 찾는다 — 파트 번호가 아니라 id여야 슬라이드를 하나
				// 끼워 넣었을 때 표가 엉뚱한 슬라이드를 가리키지 않는다.
				id := goldenSlideID(t, xml)
				want := fixtureLiteralColors[name+"/"+id]
				if got := literalColors(xml); !equalStrings(got, want) {
					t.Errorf("%s(%s)의 리터럴 색 = %v, want %v — 표에 없는 srgbClr는 토큰이 RGB로 눕은 것이다",
						slide, id, got, want)
				}
			}
		})
	}
}

// literalColorPattern은 슬라이드 파트의 리터럴 색 선언이다.
var literalColorPattern = regexp.MustCompile(`<a:srgbClr val="([0-9A-Fa-f]{6})"`)

// literalColors는 파트에 있는 리터럴 색 값을 중복 없이 정렬해 돌려준다.
func literalColors(xml string) []string {
	seen := map[string]bool{}
	var values []string
	for _, m := range literalColorPattern.FindAllStringSubmatch(xml, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		values = append(values, m[1])
	}
	sort.Strings(values)
	return values
}

// goldenSlideIDPattern은 조립기가 슬라이드 id를 남기는 자리다 (`<p:cSld name="…">`).
var goldenSlideIDPattern = regexp.MustCompile(`<p:cSld name="([^"]*)"`)

// goldenSlideID는 골든 파트에서 슬라이드 id를 읽는다 — 표의 열쇠를 파트 번호가 아니라
// 저자가 붙인 id로 두기 위해서다 (슬라이드를 끼워 넣어도 표가 따라 움직인다).
func goldenSlideID(t *testing.T, xml string) string {
	t.Helper()
	m := goldenSlideIDPattern.FindStringSubmatch(xml)
	if m == nil {
		t.Fatal("골든 슬라이드 파트에 cSld@name이 없다 — 표의 열쇠를 만들 수 없다")
	}
	return m[1]
}

// equalStrings는 두 목록이 같은지다 (빈 목록과 nil을 같게 본다 — 표에 없는 슬라이드의
// 기대값이 nil이다).
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 슬라이드 파트의 xfrm 좌표는 지오메트리 패스가 잰 값의 EMU 환산과 정확히 같아야 한다.
// 골든 XML만 보면 숫자가 그럴듯한지 알 수 없다 — 같은 빌드의 Result.Geometry와 맞대어
// "이 좌표가 그 측정에서 나왔다"를 못 박는다.
func TestFixtureCoordinatesMatchGeometry(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다 — 없으면 건너뛴다(로컬).
	for _, name := range fixtureNames {
		t.Run(name, func(t *testing.T) {
			fixture := fixtureByName(t, name)
			result := fixture.Build(t)
			if result.Geometry == nil {
				t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
			}

			data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
			if err != nil {
				t.Fatal(err)
			}
			parts := PPTXParts(t, data)
			roles := shapeRoles(t, result.Deck)
			if len(roles) != len(result.Geometry.Boxes) {
				t.Fatalf("요소 %d개, 측정 박스 %d개", len(roles), len(result.Geometry.Boxes))
			}
			for i, box := range result.Geometry.Boxes {
				role := roles[i]
				if role.rect == nil {
					// 측정되지만 자기 도형을 갖지 않는 요소다: `<li>`는 컨테이너의 도형
					// 하나가 좌표를 소유하고 (pptx.writeListShape), **도형 안 텍스트**는
					// 도형의 txBody 문단이 되며 (pptx.shapeBodyXML), 레이아웃 컨테이너와
					// `<spacer>`는 도형이 아예 없다 (pptx.writeShapeNode) — 도형 수와의
					// 관계는 아래 TestFixtureShapeCountExcludesNonShapeElements가 본다.
					continue
				}
				part := fmt.Sprintf("ppt/slides/slide%d.xml", box.Slide+1)
				xml := string(parts[part])
				if xml == "" {
					t.Fatalf("파트 %s가 없다", part)
				}
				// 잰 rect → 도형 rect 변환은 종류가 정한다: 텍스트 박스는 정렬 방향의 안전
				// 여유를 더하고(PRD §9.3, pptx.TextBoxRect) 도형은 잰 값 그대로다 — 저자가
				// 정한 크기를 넓히면 그림이 커진다 (pptx.writeShapeSp). 같은 함수로 기대값을
				// 만들어 조립기가 그 값을 썼는지 본다.
				rect := role.rect(box.Rect)
				x, y := pptx.Offset(rect)
				cx, cy := pptx.Extent(rect)
				xfrm := fmt.Sprintf("<a:off x=\"%d\" y=\"%d\"/>\n            <a:ext cx=\"%d\" cy=\"%d\"/>", x, y, cx, cy)
				if !strings.Contains(xml, xfrm) {
					t.Errorf("%s: %s의 xfrm이 측정값 %s(→도형 rect)의 EMU와 다르다 (want %q)", part, box.ID, box.Rect, xfrm)
				}
			}
		})
	}
}

// 도형 수 = 측정 박스 수 − (레이아웃 컨테이너 + `<spacer>` + `<li>` + 도형 안 텍스트).
//
// 이 관계를 골든이 아니라 계약으로 두는 이유: "레이아웃 컨테이너는 도형이 되지 않는다"가
// 골든 XML의 모양으로만 남으면, 컨테이너가 도형을 내기 시작한 변경은 골든 diff에서 "도형이
// 하나 늘었다"로만 보이고 그것이 계약 위반인지 새 픽스처 슬라이드인지 사람이 판단해야 한다.
// 측정 목록(geom.Refs — 컨테이너도 잰다)과 실제 도형 수를 맞대면 그 판단이 필요 없다.
//
// 항이 넷인 이유는 도형이 들어왔기 때문이다 (Iteration 4): `<shape>`는 **재는 것과 도형이
// 되는 것이 처음 일치하는 종류**이지만 그 자식 텍스트는 재고도 도형이 되지 않는다 (부모
// 도형의 txBody 문단이 된다 — pptx.shapeBodyXML). 그래서 종류 판정이 태그 이름만으로는
// 부족하고 **부모가 도형인가**를 알아야 한다 — 판정을 shapeRoles의 순회 하나가 소유한다
// (좌표 대조와 같은 목록이어야 두 검사가 같은 계약을 본다).
//
// **"도형"의 뜻이 Iteration 5에서 넓어졌다**: `<image>`도 재고 도형이 되지만 그 요소는
// `p:sp`가 아니라 `p:pic`이다 (pptx.writePicShape). 등식의 항이 아니라 항의 뜻이 넓어진 것이라
// 세는 쪽(countedShapeTags)이 두 태그를 함께 센다 — `p:sp`만 세면 그림이 든 슬라이드에서
// 등식이 "도형이 하나 모자라다"로 깨지고, 그것은 계약 위반이 아니라 세는 코드의 낡음이다.
func TestFixtureShapeCountExcludesNonShapeElements(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다.
	for _, name := range fixtureNames {
		t.Run(name, func(t *testing.T) {
			fixture := fixtureByName(t, name)
			result := fixture.Build(t)
			if result.Geometry == nil {
				t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
			}
			data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
			if err != nil {
				t.Fatal(err)
			}
			parts := PPTXParts(t, data)

			roles := shapeRoles(t, result.Deck)
			if len(roles) != len(result.Geometry.Boxes) {
				t.Fatalf("요소 %d개, 측정 박스 %d개", len(roles), len(result.Geometry.Boxes))
			}
			want := 0
			excluded := map[string]int{}
			for _, role := range roles {
				if role.rect != nil {
					want++
					continue
				}
				excluded[role.excluded]++
			}
			got := 0
			for i := range result.Deck.Slides {
				got += countShapes(string(parts[fmt.Sprintf("ppt/slides/slide%d.xml", i+1)]))
			}
			// 등식을 로그에 그대로 적는다 — 항 이름이 순회의 분류에서 오므로 항이 늘면
			// 로그도 함께 자란다 (손으로 적은 항 목록은 갱신을 잊는다).
			terms := make([]string, 0, len(shapeExcludedKinds))
			for _, kind := range shapeExcludedKinds {
				terms = append(terms, fmt.Sprintf("− %s %d", kind, excluded[kind]))
			}
			t.Logf("측정 박스 %d개 %s = 도형 %d개",
				len(roles), strings.Join(terms, " "), got)
			if got != want {
				t.Errorf("도형 %d개, want %d개 (측정 박스 %d %s)",
					got, want, len(roles), strings.Join(terms, " "))
			}
		})
	}
}

// shapeFixtureSlideShapes는 04_shape의 슬라이드별 도형(`<p:sp>`) 수다 — 슬라이드 id → 개수.
//
// 위의 등식(도형 수 = 측정 박스 − 제외 항)은 덱 **전체**의 합만 보므로 한 슬라이드에서 도형이
// 하나 늘고 다른 슬라이드에서 하나 줄면 통과한다. 이 표는 그 상쇄를 막고, 무엇보다 **도형 안
// 텍스트가 자기 도형이 되지 않는다**를 슬라이드마다 수로 못 박는다: 001은 도형 12개와 그
// 안의 이름표 12개인데 도형은 13개(제목 + 12)여야 한다 — 흡수가 깨지면 25개가 된다.
//
// 브라우저 없이 도는 검사다 (골든 파트만 읽는다).
var shapeFixtureSlideShapes = map[string]int{
	// 제목 텍스트 박스 1 + 프리셋 12종 + 도형 밖 이름표 12개 = 25. 이름표를 도형 밖에
	// 둔 이유는 픽스처 주석에 있다 (프리셋의 텍스트 사각형은 외곽선보다 작다).
	"shape-types": 25,
	// 제목 1 + 도형 3 (`<row>`는 도형이 아니다). 자식 h3·p 5개는 흡수된다.
	"shape-token-colors": 4,
	// 제목 1 + 도형 2.
	"shape-literal-colors": 3,
	// 제목 1 + 도형 7 (radius 3 + shadow·opacity·rotate·flip 4). 회전 도형은 자식이
	// 없으므로 빈 txBody를 가진 도형 하나다.
	"shape-appearance": 8,
	// 제목 1 + 도형 5. 마지막 도형의 자식 둘(h3+p)이 문단 둘로 흡수된다.
	"shape-text-anchor": 6,
	// 제목 1 + (헤더+본문) × 4 + span 배너 1 = 10. `<col>`·`<grid>`는 도형이 아니다.
	"shape-cards": 10,
	// 제목 1 + 도형 4.
	"shape-variant": 5,
	// 제목 1 + 도형 1 + 슬라이드 직속 `<text>` 1 = 3 (경고는 도형 수를 바꾸지 않는다).
	"shape-variant-unknown": 3,
}

// 도형 픽스처의 슬라이드마다 도형 수가 표와 같은가 — 도형 안 텍스트가 흡수된다는 계약을
// 슬라이드 단위로 본다.
func TestShapeFixtureSlideShapeCounts(t *testing.T) {
	goldenDir := fixtureByName(t, shapeFixtureName).GoldenDir()
	seen := map[string]bool{}
	for i := 1; i <= slideCounts[shapeFixtureName]; i++ {
		slide := fmt.Sprintf("slide%d.xml", i)
		data, err := os.ReadFile(filepath.Join(goldenDir, "pptx", "ppt", "slides", slide))
		if err != nil {
			t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
		}
		xml := string(data)
		id := goldenSlideID(t, xml)
		want, ok := shapeFixtureSlideShapes[id]
		if !ok {
			t.Errorf("%s(%s)가 표에 없다 — 슬라이드가 늘면 기대 도형 수를 함께 적는다", slide, id)
			continue
		}
		seen[id] = true
		if got := countShapes(xml); got != want {
			t.Errorf("%s(%s): 도형 %d개, want %d개", slide, id, got, want)
		}
	}
	for id := range shapeFixtureSlideShapes {
		if !seen[id] {
			t.Errorf("표의 슬라이드 %s가 골든에 없다", id)
		}
	}
}

// rotatedXfrmPattern은 회전이 실린 `xfrm`과 그 크기다 (조립기가 내는 순서 그대로).
var rotatedXfrmPattern = regexp.MustCompile(
	`<a:xfrm rot="(\d+)">\s*<a:off x="\d+" y="\d+"/>\s*<a:ext cx="(\d+)" cy="(\d+)"/>`)

// 회전 도형의 `ext`가 **회전 전** 크기인가.
//
// 좌표 대조(TestFixtureCoordinatesMatchGeometry)로는 이것을 볼 수 없다: 그 검사는 산출물이
// **측정값과 같은지**만 보므로, 측정 패스가 회전된 외접 사각형을 재기 시작하면 두 값이 함께
// 커지며 통과한다 (`getBoundingClientRect`는 기본적으로 그 bbox를 돌려주고, 그래서 측정 뷰가
// 변형을 끈다 — web.MeasureView). 그러면 PowerPoint에서 도형의 조절점이 실제 도형보다 크고
// 회전 각도가 그 위에 또 얹혀 그림이 웹과 갈라진다.
//
// 그래서 저자가 **선언한 크기**를 기대값의 출처로 둔다 (골든 XML만 읽는다 — 브라우저 없이 돈다).
func TestShapeFixtureRotatedExtentIsPreRotation(t *testing.T) {
	// 004_appearance의 `<shape type="rect" rotate="45" w="120" h="120"/>` 하나뿐이다.
	const (
		declaredPx = 120.0
		degrees    = 45.0
	)
	goldenDir := fixtureByName(t, shapeFixtureName).GoldenDir()
	var found [][]string
	for i := 1; i <= slideCounts[shapeFixtureName]; i++ {
		data, err := os.ReadFile(filepath.Join(goldenDir, "pptx", "ppt", "slides", fmt.Sprintf("slide%d.xml", i)))
		if err != nil {
			t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
		}
		found = append(found, rotatedXfrmPattern.FindAllStringSubmatch(string(data), -1)...)
	}
	if len(found) != 1 {
		t.Fatalf("회전이 실린 도형이 %d개다, want 1개 — 픽스처가 바뀌면 이 기대값을 함께 고친다", len(found))
	}
	m := found[0]
	wantRot := fmt.Sprint(pptx.Rotation(degrees))
	if m[1] != wantRot {
		t.Errorf("rot=%s, want %s", m[1], wantRot)
	}
	want := fmt.Sprint(pptx.EMU(declaredPx))
	// 회전된 bbox는 변의 √2배다 — 그 값이 나오면 측정이 변형을 끄지 않은 것이다.
	bbox := fmt.Sprint(pptx.EMU(declaredPx * math.Sqrt2))
	for i, got := range []string{m[2], m[3]} {
		axis := [...]string{"cx", "cy"}[i]
		if got == want {
			continue
		}
		if got == bbox {
			t.Errorf("%s=%s가 회전된 bbox(%.1fpx)다 — 측정 패스가 변형을 끄지 않았다",
				axis, got, declaredPx*math.Sqrt2)
			continue
		}
		t.Errorf("%s=%s, want %s (선언 %gpx의 EMU)", axis, got, want, declaredPx)
	}
	t.Logf("회전 도형: rot=%s, ext=%s×%s EMU (선언 %gpx, 회전된 bbox는 %s였을 것)",
		m[1], m[2], m[3], declaredPx, bbox)
}

// shapeCropMargin은 도형별 픽셀 비교에서 잰 박스 밖으로 넓히는 여유다 (px).
//
// 도형의 잉크는 잰 rect 밖으로 나간다: 테두리는 경로 중앙에 그려져 굵기의 절반이 번지고
// (양 타깃 모두 — render/web/shape.go), `callout`의 꼬리는 프리셋 정의대로 박스 아래로
// 12.5%(h=110에서 13.75px) 나간다. 16px은 그 둘을 덮으면서 격자 간격(`gap="md"` = 24px)보다
// 작아 **이웃 도형이 크롭에 들어오지 않는다** — 들어오면 오차의 주인이 흐려진다.
const shapeCropMargin = 16.0

// shapePresetDiffMaxRatio는 도형 하나(크롭 영역) 안의 픽셀 오차 허용 비율이다.
//
// 슬라이드 전체 비율(fixtureDiffOptions)과 갈라 두는 이유: 12종이 한 슬라이드에 있으면
// 프리셋 하나의 기하가 어긋나도 슬라이드 비율에서는 12분의 1로 희석된다 — 그리고 이
// iteration에서 갈라지면 그것은 잔차가 아니라 **모양이 다른 것**이다 (면은 글리프와 달리
// 경계 픽셀 몇 개가 아니라 영역으로 다르다).
//
// 10%는 측정값에 여유를 더한 값이다 (Iteration 4 항목 06의 측정 — 12종 전수의 표):
// 크롭 안에서 글자는 소수 픽셀이고 대부분은 채움·테두리이므로 기하가 같으면 오차는 테두리
// 안티앨리어싱과 이름표 글리프에서만 나온다. 측정 최대는 `parallelogram` 7.95%이고
// 최소는 `ellipse` 2.90%인데, 이 순서가 오차의 정체를 말한다 — **긴 사선 테두리**를 가진
// 프리셋(parallelogram·cloud 6.04%·arrow-right 5.74%)이 크고 매끄러운 곡선(ellipse)이
// 작다. 기하가 어긋나면 오차는 이 폭이 아니라 어긋난 **면적**으로 나타나므로(수십 %)
// 이 임계값은 기하 갈라짐을 여전히 잡는다.
const shapePresetDiffMaxRatio = 0.10

// 프리셋 12종의 기하가 웹과 LibreOffice에서 **같은 모양**인가 — 도형별로 잰다.
//
// 이 iteration의 가장 큰 판정이다. 웹은 프리셋을 정사각형에서 평가한 정규 공간에 두고 박스에
// 늘리고(render/web.shapeGeometries), PPTX는 잰 박스로 `avLst` 조정값을 계산해 그 늘림을
// 맞춘다(pptx.AdjustValue) — 두 유도가 어긋나면 같은 프리셋이 두 타깃에서 다른 비율로
// 그려진다. 슬라이드 전체 비율로는 어느 프리셋이 어긋났는지 알 수 없으므로, 격자에 12개가
// 있다는 구조를 이용해 도형마다 영역을 잘라 수치를 남긴다.
func TestShapeFixturePresetGeometryPerShape(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, shapeFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	const slideID = "shape-types"
	index := -1
	for i, slide := range result.Deck.Slides {
		if slide.ID == slideID {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("슬라이드 %s가 없다", slideID)
	}

	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(index))
	loImg := shots[index]

	// 프리셋 이름은 덱 모델에서 문서 순서로 모은다 — 측정 박스도 같은 순서이므로 자리로
	// 짝지을 수 있다 (골든 XML의 `prstGeom`을 읽으면 조립기의 매핑을 조립기로 검증하게 된다).
	presets := shapePresetsInOrder(result.Deck.Slides[index])
	var boxes []geom.Box
	for _, box := range result.Geometry.Boxes {
		if box.Slide == index && box.El == soml.ShapeElement {
			boxes = append(boxes, box)
		}
	}
	if len(boxes) != len(presets) {
		t.Fatalf("측정 도형 %d개, 덱의 도형 %d개", len(boxes), len(presets))
	}
	if len(presets) != len(soml.ShapeTypes()) {
		t.Fatalf("슬라이드에 도형이 %d개다 — `type` 12종 전수여야 한다", len(presets))
	}

	worst, worstName := 0.0, ""
	for i, box := range boxes {
		crop := geom.Rect{
			X: box.Rect.X - shapeCropMargin,
			Y: box.Rect.Y - shapeCropMargin,
			W: box.Rect.W + 2*shapeCropMargin,
			H: box.Rect.H + 2*shapeCropMargin,
		}
		cmp, err := Diff(cropImage(webImg, crop), cropImage(loImg, crop), DefaultDiffOptions())
		if err != nil {
			t.Fatalf("%s: %v", presets[i], err)
		}
		t.Logf("%-14s %s: %s", presets[i], box.Rect, cmp)
		if cmp.Ratio > worst {
			worst, worstName = cmp.Ratio, presets[i]
		}
		if cmp.Ratio > shapePresetDiffMaxRatio {
			t.Errorf("%s: 도형 안 픽셀 오차 %.2f%%가 허용 %.2f%%를 넘는다 — 프리셋 기하가 갈라졌다면 "+
				"고칠 곳은 오차가 아니라 유도표(render/web.shapeGeometries)와 조정값(pptx.AdjustValue)이다",
				presets[i], cmp.Ratio*100, shapePresetDiffMaxRatio*100)
		}
	}
	t.Logf("가장 큰 도형 오차: %s %.2f%% (허용 %.2f%%)", worstName, worst*100, shapePresetDiffMaxRatio*100)
}

// shapePresetsInOrder는 슬라이드의 도형 `type`을 문서 순서로 모은다.
func shapePresetsInOrder(slide *soml.Slide) []string {
	var presets []string
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if style := elem.ShapeStyle(); style != nil {
			presets = append(presets, style.Type)
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, node := range slide.Content {
		walk(node)
	}
	return presets
}

// cropImage는 이미지에서 사각형 영역을 새 이미지로 잘라 온다 (이미지 경계로 자른다).
//
// 두 렌더의 같은 영역을 잘라야 하므로 좌표는 반올림 없이 바닥·천장으로 맞춘다 — 한쪽만
// 반올림하면 크롭이 1px 어긋나 그 차이가 전부 오차로 계산된다.
func cropImage(img image.Image, r geom.Rect) image.Image {
	b := img.Bounds()
	x0, y0 := max(int(math.Floor(r.X)), 0), max(int(math.Floor(r.Y)), 0)
	x1, y1 := min(int(math.Ceil(r.Right())), b.Dx()), min(int(math.Ceil(r.Bottom())), b.Dy())
	out := image.NewRGBA(image.Rect(0, 0, max(x1-x0, 0), max(y1-y0, 0)))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			out.SetRGBA(x-x0, y-y0, rgb8(img.At(b.Min.X+x, b.Min.Y+y)))
		}
	}
	return out
}

// inShapeBandInset은 도형 안 줄 세기에서 잰 rect를 안으로 줄이는 폭이다 (px).
//
// 테두리를 영역에서 빼기 위한 것이다: 테두리는 경로 중앙에 그려져 잰 rect의 변에 걸치므로
// (최대 굵기 3px), 영역에 들어오면 위아래 변이 각각 하나의 띠가 되어 줄 수에 더해진다.
// 4px은 가장 굵은 테두리(3px)에 안티앨리어싱 1px을 더한 값이다.
const inShapeBandInset = 4.0

// inShapeBandRegion은 도형 안 줄 세기를 판정할 영역이다 — 잰 rect가 아니라 **프리셋의 텍스트
// 영역**(ADR 0040)을 테두리·모서리만큼 더 줄인 사각형이고, 그 영역이 성립하지 않으면 false다.
//
// 텍스트 영역에서 재는 이유는 InkBands의 전제다: "영역에서 가장 흔한 색이 배경"인데, 채움이
// 잰 rect를 덮지 않는 프리셋(diamond·star·cloud…)에서는 배경 추정이 도형 밖 흰색을 골라
// **도형 자체가 하나의 거대한 띠**가 된다. 프리셋의 텍스트 영역은 실루엣 안이라(그래서 글이
// 거기 흐른다) 그 영역에서는 채움이 다시 최빈색이다 — 전에는 이 검사를 rect·rounded-rect로
// 닫아 두었는데, 영역을 옮기면 12종 전수로 열린다.
//
// 성립하지 않는 자리는 chevron 하나다: 그 프리셋의 rect는 폭 0으로 붕괴해 우리가 **박스
// 전체**를 쓰므로(PPTX가 되돌린다 — ADR 0041) 영역 안에 도형 밖 흰색이 크게 남는다. 판정을
// 이름이 아니라 `ShapeTextCancel`로 하는 이유는 목록이 두 벌이 되지 않게 하는 것이다.
func inShapeBandRegion(entry inShapeText) (geom.Rect, bool) {
	if cancel, ok := web.ShapeTextCancel(entry.preset); ok && cancel.HasPercent() {
		return geom.Rect{}, false
	}
	// 위아래는 **모서리 반경만큼 더** 줄인다: 둥근 모서리에서는 잰 rect의 위·아래 몇 줄에
	// 도형 밖 흰색이 남아, 그 줄이 배경(채움)과 달라 하나의 띠로 세어진다 (측정: radius=32
	// 도형이 글줄 하나인데 띠 셋으로 읽혔다 — 위 모서리·글자·아래 모서리). 반경만큼
	// 내려가면 그 행부터는 폭 전체가 채움이다 — 텍스트 영역의 px 인셋(0.29·radius)은 원호에
	// **닿는** 사각형이라 그만큼으로는 모자란다.
	insetY := inShapeBandInset
	if entry.radius+2 > insetY {
		insetY = entry.radius + 2
	}
	text := presetTextRect(entry)
	x0 := math.Max(text.X, entry.rect.X+inShapeBandInset)
	y0 := math.Max(text.Y, entry.rect.Y+insetY)
	x1 := math.Min(text.Right(), entry.rect.Right()-inShapeBandInset)
	y1 := math.Min(text.Bottom(), entry.rect.Bottom()-insetY)
	if x1-x0 <= 0 || y1-y0 <= 0 {
		return geom.Rect{}, false
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, true
}

// inShapeLineDivergences는 도형 안 문단의 줄 수가 **실제로 갈라지는** 자리다
// (`<픽스처>/<요소 id>` → 브라우저 줄 수 / LibreOffice 줄 수).
//
// 갈라짐을 표로 **단정**하는 이유(값을 완화하지 않는 이유): 이것은 오차가 아니라 두 엔진의
// **줄바꿈 규칙 차이**다 — Chromium은 한글을 음절 사이에서 끊고(`데이터와 모델의 편향/을`),
// LibreOffice는 낱말 경계(공백)에서만 끊는다(`데이터와 모델의/편향을 측정한다`). 그래서 같은
// 폭에서 LibreOffice의 첫 줄이 더 짧아지고, 밀린 글자가 마지막 줄을 하나 더 만든다.
// 픽셀 비율로 덮으면 이 사실이 기록에서 사라지므로, 갈라진 자리와 수를 여기 적어 **바뀌면
// 테스트가 깨지게** 둔다 (고쳐도 깨진다 — 그때 표를 지우는 것이 고쳤다는 증거다).
//
// 왜 도형에서 처음 드러났는가: 텍스트 박스는 잰 글자 폭보다 4% 넓게 내보내 흡수하지만
// (pptx.TextBoxRect) **도형은 넓힐 수 없다** — 박스가 저자가 정한 크기다. 다만 원인은 여유가
// 아니라 규칙이므로 여유가 8.0%인 자리에서도 갈라진다 (아래 s1.1.1) — 넓히는 것으로는 닫히지
// 않는 문제이고, 후보 수단은 웹에 `word-break: keep-all`을 걸어 **규칙 자체를 맞추는 것**이다
// (모든 덱의 타이포그래피가 움직이므로 이 항목의 범위 밖이다 — NOTES 04/06의 열린 질문).
var inShapeLineDivergences = map[string][2]int{
	// 002_token_colors의 accent1 카드 본문 `글자색도 토큰(bg)이다.` — 여유 8.0%.
	// LibreOffice가 `(bg)` 뒤에서 끊어 `이다.`가 셋째 줄로 내려간다.
	shapeFixtureName + "/s1.1.1": {2, 3},
	// 006_cards의 첫 카드 본문 `데이터와 모델의 편향을 측정한다.` — 여유 2.4%.
	// LibreOffice는 `데이터와 모델의`까지만 첫 줄에 두고(낱말 경계) 마침표가 셋째 줄이 된다.
	shapeFixtureName + "/s6.1.0.1": {2, 3},
}

// 도형 안 문단이 두 엔진에서 **같은 줄 수**로 접히는가.
//
// 슬라이드 단위 잉크 띠 판정(TestFixtureLineBreaksMatchLibreOffice)이 도형 픽스처에서
// 성립하지 않으므로(칠한 도형이 큰 면을 덮어 띠가 글줄이 아니라 도형이 된다 —
// inkBandSkipSlides) 줄 수 계약을 **도형 안쪽**으로 옮긴다: 도형의 잰 rect 안에서는 가장 흔한
// 색이 채움이고 잉크는 글자다.
//
// 이 검사가 이 iteration에 필요한 이유(항목 05의 5번): 텍스트 박스는 정렬 방향으로 4% 넓혀
// LibreOffice의 스크립트 간격을 흡수하지만(pptx.TextBoxRect) **도형은 넓힐 수 없다** — 박스가
// 저자가 정한 크기이므로 넓히면 그림이 커진다. 그래서 도형 안 문단에는 여유를 만드는 장치가
// 없고, 여유가 0에 가까운 문단은 LibreOffice에서 줄이 하나 늘어난다. 늘어난 줄은 도형 밖으로
// 흐르므로 픽셀 비율로는 잔차와 구별되지 않는다 — 그것이 이 검사가 따로 있는 이유다.
func TestShapeFixtureInShapeTextLineCounts(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, shapeFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

	webShots := make([]image.Image, len(result.Deck.Slides))
	for i := range result.Deck.Slides {
		webShots[i] = browser.ScreenshotElement(t, web.SlideIndexSelector(i))
	}

	entries := inShapeTexts(t, result.Deck, result.Geometry)
	if len(entries) == 0 {
		t.Fatal("도형 안 텍스트가 하나도 없다 — 픽스처가 이 계약을 시험하지 않는다")
	}
	worst := 1.0
	seen := map[string]bool{}
	for _, entry := range entries {
		inner, ok := inShapeBandRegion(entry)
		if !ok {
			t.Logf("%s %s: 줄 세기를 건너뛴다 — 프리셋의 텍스트 영역이 실루엣 안이 아니다 (inShapeBandRegion)",
				entry.id, entry.preset)
			continue
		}
		webBands := InkBands(webShots[entry.slide], inner)
		loBands := InkBands(shots[entry.slide], inner)
		if entry.slack < worst {
			worst = entry.slack
		}
		t.Logf("%s %s %s: 웹 잰 줄 %d / 웹 잉크 띠 %d / LibreOffice 잉크 띠 %d, 여유 %.1f%%",
			entry.id, entry.preset, entry.rect, entry.lines, len(webBands), len(loBands), entry.slack*100)
		if len(webBands) != entry.lines {
			// 웹 그림과 잰 줄이 어긋나면 측정 방법이 틀린 것이다 (LibreOffice 판정보다 먼저
			// 본다 — bandLineOwners와 같은 태도).
			t.Errorf("%s: 웹 잉크 띠 %d개가 잰 줄 %d개와 다르다 — 도형 안 줄 세기 방법이 틀렸다\n  띠 %v",
				entry.id, len(webBands), entry.lines, webBands)
			continue
		}
		if want, known := inShapeLineDivergences[shapeFixtureName+"/"+entry.id]; known {
			seen[entry.id] = true
			if len(webBands) != want[0] || len(loBands) != want[1] {
				t.Errorf("%s: 알려진 갈라짐이 %d/%d줄인데 %d/%d줄이다 (브라우저/LibreOffice) — "+
					"고쳤으면 inShapeLineDivergences에서 지우고, 나빠졌으면 원인을 찾는다\n  웹 %v\n  LibreOffice %v",
					entry.id, want[0], want[1], len(webBands), len(loBands), webBands, loBands)
			}
			continue
		}
		if len(loBands) != len(webBands) {
			t.Errorf("%s: 도형 안 줄 수가 갈라진다 — 브라우저 %d줄, LibreOffice %d줄 (여유 %.1f%%). "+
				"도형은 넓힐 수 없으므로(pptx.writeShapeSp) 오차를 키울 자리가 아니다\n  웹 %v\n  LibreOffice %v",
				entry.id, len(webBands), len(loBands), entry.slack*100, webBands, loBands)
		}
	}
	t.Logf("도형 안 문단의 최소 여유 비율 = %.1f%% — 이 값이 0에 가까울수록 LibreOffice에서 줄이 늘 위험이 크다", worst*100)
	// 표에 있는데 픽스처에서 사라진 자리는 지운다 — 남아 있으면 갈라짐이 고쳐졌는지
	// 없어졌는지 구별되지 않는다.
	for key := range inShapeLineDivergences {
		if id := strings.TrimPrefix(key, shapeFixtureName+"/"); !seen[id] {
			t.Errorf("inShapeLineDivergences의 %s가 픽스처에 없다 — 표에서 지운다", key)
		}
	}
}

// inShapeTextRectSlides는 프리셋별 텍스트 영역을 시험하는 슬라이드다 (id → 가운데 정렬인가).
//
// 두 장인 이유는 판정이 둘이기 때문이다: 가운데 정렬은 영역의 **중심**을, 왼쪽·위 정렬은
// **왼쪽·위 변**을 드러낸다 (대칭 인셋 프리셋은 가운데 정렬로는 어긋남이 상쇄된다 — 픽스처
// 009·010의 주석).
var inShapeTextRectSlides = map[string]bool{
	"shape-text-rect":        true,
	"shape-text-rect-corner": false,
}

// inShapeTextRectSlack은 도형 안 글자 자리를 두 타깃에서 견줄 때의 허용 오차다 (px).
//
// 두 엔진의 글리프 래스터라이즈가 다르므로 잉크의 바깥 경계는 완전히 같을 수 없다 (좌우
// 사이드 베어링, 첫 기준선의 반올림 — 실측 4px). 그 상수는 `rect` 행이 대조군으로
// **빼 주므로**(inShapeTextRectControl) 여기 남는 것은 프리셋마다 다른 몫뿐이고, 그래서 이
// 값이 표에서 가장 작은 인셋(rounded-rect의 `0.29289 × radius` = 반경 12에서 3.5px)보다
// 작을 수 있다 — 어긋남의 정체를 가리지 않는다는 뜻이다.
const inShapeTextRectSlack = 2.0

// inShapeTextRectControl은 대조군 프리셋이다 — 텍스트 영역이 박스 전체인 `rect`다.
//
// 왜 필요한가: 두 엔진의 첫 줄 기준선이 같은 자리가 아니어서 같은 텍스트 영역에서도 잉크의
// 위 경계가 상수만큼 (실측 4px) 어긋난다. 그 상수를 허용 오차에 흡수시키면 오차가 표의 가장
// 작은 인셋만큼 커져 시험이 무뎌지므로, **영역이 박스와 같은 프리셋**에서 그 상수를 재고
// 나머지 프리셋의 어긋남에서 뺀다 — 남는 것이 순수하게 텍스트 영역의 몫이다.
const inShapeTextRectControl = "rect"

// inShapeTextRectDelta는 도형 하나에서 잰 두 타깃의 어긋남이다 — 대조군을 빼려면 한 슬라이드의
// 잰 값이 다 모인 뒤에 판정해야 하므로 잰 것과 판정하는 것이 두 단계로 나뉜다.
type inShapeTextRectDelta struct {
	entry inShapeText
	// dx·dy는 웹 잉크 − LibreOffice 잉크다 (what이 무엇을 견줬는지 말한다: 중심 또는 왼쪽 위).
	dx, dy float64
	what   string
	// text는 표가 뜻하는 텍스트 영역, loInk는 LibreOffice가 실제로 글자를 놓은 자리다 —
	// 진단 로그와 "영역 안에 있는가" 단정이 쓴다.
	text  geom.Rect
	loInk geom.Rect
}

// 프리셋의 텍스트 영역이 두 타깃에서 **같은 자리**인가 (ADR 0040의 회귀 테스트).
//
// 표(render/web.ShapeTextRect)가 맞는지를 우리 코드 두 벌을 견주는 것으로는 알 수 없다 —
// 수치의 출처가 PowerPoint·LibreOffice가 들고 있는 프리셋 정의이고, PPTX 쪽은 그 값을 **쓰지
// 않는다**(ADR 0040: `bodyPr` 인셋은 저자의 `pad`다). 그래서 보장을 픽셀로 옮긴다: 같은
// SOML에서 웹 잉크와 LibreOffice 잉크가 같은 자리에 오면 웹이 읽은 표가 프리셋과 같다는
// 뜻이다. 표가 틀리면 이 검사가 깨지고, 표를 두 벌로 공유하는 것보다 강하다.
//
// 잉크를 **박스 전체**에서 재는 이유: 고치기 전의 웹 글자는 텍스트 영역 밖(박스 모서리)에
// 있으므로 영역으로 크롭하면 증거가 잘려 나간다. 박스 안에서 글자만 잉크가 되는 것은 픽스처의
// `fill="surface"` 덕이다 (009의 주석 — 잉크 임계값보다 칠과 배경의 차가 작다).
func TestShapeFixtureInShapeTextRect(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, shapeFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	centered := map[int]bool{}
	for i, slide := range result.Deck.Slides {
		if mode, ok := inShapeTextRectSlides[slide.ID]; ok {
			centered[i] = mode
		}
	}
	if len(centered) != len(inShapeTextRectSlides) {
		t.Fatalf("텍스트 영역 슬라이드를 %d장만 찾았다 — %v", len(centered), inShapeTextRectSlides)
	}

	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	webShots := map[int]image.Image{}
	for index := range centered {
		webShots[index] = browser.ScreenshotElement(t, web.SlideIndexSelector(index))
	}

	seen := map[string]int{}
	var measured []inShapeTextRectDelta
	control := map[int]inShapeTextRectDelta{}
	for _, entry := range inShapeTexts(t, result.Deck, result.Geometry) {
		mode, ok := centered[entry.slide]
		if !ok {
			continue
		}
		seen[entry.preset]++
		text := presetTextRect(entry)
		webInk, webOK := inkBounds(webShots[entry.slide], entry.rect)
		loInk, loOK := inkBounds(shots[entry.slide], entry.rect)
		if !webOK || !loOK {
			t.Errorf("%s %s: 박스 안에 잉크가 없다 (웹 %t, LibreOffice %t) — 글자가 박스를 "+
				"벗어났거나 칠이 잉크로 읽혔다", entry.id, entry.preset, webOK, loOK)
			continue
		}
		delta := inShapeTextRectDelta{entry: entry, text: text, loInk: loInk, what: "왼쪽 위"}
		if mode {
			delta.dx = (webInk.X + webInk.W/2) - (loInk.X + loInk.W/2)
			delta.dy = (webInk.Y + webInk.H/2) - (loInk.Y + loInk.H/2)
			delta.what = "중심"
		} else {
			delta.dx, delta.dy = webInk.X-loInk.X, webInk.Y-loInk.Y
		}
		measured = append(measured, delta)
		if entry.preset == inShapeTextRectControl {
			control[entry.slide] = delta
		}
	}
	if len(control) != len(centered) {
		t.Fatalf("대조군 %q를 %d장에서만 찾았다, want %d장 — 두 엔진의 기준선 차를 뺄 수 없다",
			inShapeTextRectControl, len(control), len(centered))
	}
	for _, d := range measured {
		base := control[d.entry.slide]
		dx, dy := d.dx-base.dx, d.dy-base.dy
		// 표가 뜻하는 인셋과 LibreOffice가 실제로 준 인셋을 나란히 남긴다 — 깨졌을 때
		// 고칠 곳이 웹인지 표인지 이 줄이 가른다.
		t.Logf("%-14s %s: 잉크 %s Δ = (%+5.1f, %+5.1f)px (대조군 보정 전 %+.1f, %+.1f) | "+
			"표 l %.1f t %.1f r %.1f b %.1f | LibreOffice 잉크 − 박스 l %+.1f t %+.1f",
			d.entry.preset, d.entry.rect, d.what, dx, dy, d.dx, d.dy,
			d.text.X-d.entry.rect.X, d.text.Y-d.entry.rect.Y,
			d.entry.rect.Right()-d.text.Right(), d.entry.rect.Bottom()-d.text.Bottom(),
			d.loInk.X-d.entry.rect.X, d.loInk.Y-d.entry.rect.Y)
		if math.Abs(dx) > inShapeTextRectSlack || math.Abs(dy) > inShapeTextRectSlack {
			t.Errorf("%s %s: 도형 안 글자의 %s가 두 타깃에서 (%+.1f, %+.1f)px 어긋난다 (허용 %.1fpx) — "+
				"웹이 프리셋의 텍스트 영역을 읽지 않으면 이 값은 표의 인셋과 같다 (ADR 0040)",
				d.entry.id, d.entry.preset, d.what, dx, dy, inShapeTextRectSlack)
		}
		// LibreOffice의 글자는 프리셋의 텍스트 영역 안에 있어야 한다 — 표가 프리셋과 같다는
		// 직접 단정이다 (`pad`는 영역 안쪽에 더 들어가므로 한 방향으로만 본다).
		if d.loInk.X < d.text.X-inShapeTextRectSlack || d.loInk.Y < d.text.Y-inShapeTextRectSlack ||
			d.loInk.Right() > d.text.Right()+inShapeTextRectSlack || d.loInk.Bottom() > d.text.Bottom()+inShapeTextRectSlack {
			t.Errorf("%s %s: LibreOffice 잉크 %s가 표의 텍스트 영역 %s을 벗어난다 — 표의 행이 "+
				"프리셋 정의와 다르다 (render/web.ShapeTextRect)", d.entry.id, d.entry.preset, d.loInk, d.text)
		}
	}
	for _, preset := range soml.ShapeTypes() {
		if seen[preset] != len(inShapeTextRectSlides) {
			t.Errorf("%s: 텍스트 영역 슬라이드에서 %d번 잡혔다, want %d번 — 프리셋 전수여야 한다",
				preset, seen[preset], len(inShapeTextRectSlides))
		}
	}
}

// presetTextRect는 프리셋의 텍스트 영역을 잰 박스 위에 얹는다 (ADR 0040의 표).
func presetTextRect(entry inShapeText) geom.Rect {
	insets, ok := web.ShapeTextRect(entry.preset)
	if !ok {
		return entry.rect
	}
	l, t, r, b := insets.Px(entry.rect.W, entry.rect.H, entry.radius)
	return geom.Rect{X: entry.rect.X + l, Y: entry.rect.Y + t, W: entry.rect.W - l - r, H: entry.rect.H - t - b}
}

// inkBounds는 영역 안 잉크 전체를 감싸는 사각형이다 (띠의 합집합).
func inkBounds(img image.Image, r geom.Rect) (geom.Rect, bool) {
	bands := InkBands(img, r)
	if len(bands) == 0 {
		return geom.Rect{}, false
	}
	out := bands[0]
	for _, band := range bands[1:] {
		if band.X < out.X {
			out.W += out.X - band.X
			out.X = band.X
		}
		if band.Right() > out.Right() {
			out.W = band.Right() - out.X
		}
		out.H = band.Bottom() - out.Y
	}
	return out, true
}

// inShapeText는 텍스트를 담은 도형 하나다 (도형 안 줄 수 대조의 단위).
type inShapeText struct {
	id     string
	preset string
	slide  int
	rect   geom.Rect
	// radius는 모서리 반경 px다 (rounded-rect가 아니면 0) — 줄 세기 영역이 둥근 모서리를
	// 피해야 한다.
	radius float64
	// lines는 브라우저가 이 도형 안에서 나눈 줄 수다 (자식 문단 전부의 합).
	lines int
	// slack은 자식 문단 중 **가장 좁은** 여유 비율이다 (Slack/MaxLineWidth).
	slack float64
}

// inShapeTexts는 텍스트를 담은 도형을 지오메트리와 같은 순서로 훑어 모은다.
//
// 순회 규칙은 geom.Refs와 같아야 한다 (인라인은 자손까지 건너뛴다) — 어긋나면 자리가 밀려
// 엉뚱한 요소의 rect를 도형의 rect로 쓴다. 그 계약을 마지막에 개수로 확인한다.
func inShapeTexts(t *testing.T, deck *soml.Deck, g *geom.Geometry) []inShapeText {
	t.Helper()
	var out []inShapeText
	index := 0
	var walk func(node soml.Node, shape *inShapeText)
	walk = func(node soml.Node, shape *inShapeText) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		kind := soml.KindOf(elem.Name)
		if kind == soml.KindInline {
			return
		}
		if index >= len(g.Boxes) {
			t.Fatalf("순회가 측정 박스(%d개)보다 길다", len(g.Boxes))
		}
		box := g.Boxes[index]
		index++
		if kind == soml.KindShape {
			style := elem.ShapeStyle()
			if style == nil {
				t.Fatalf("<shape>의 외관이 해석되지 않았다")
			}
			entry := inShapeText{id: box.ID, preset: style.Type, slide: box.Slide, rect: box.Rect, slack: 1.0}
			if style.Radius != nil {
				entry.radius = style.Radius.Px
			}
			for _, child := range elem.Children {
				walk(child, &entry)
			}
			if entry.lines > 0 {
				out = append(out, entry)
			}
			return
		}
		if shape != nil {
			shape.lines += box.LineCount()
			if width := box.MaxLineWidth(); width > 0 {
				if ratio := box.Slack() / width; ratio < shape.slack {
					shape.slack = ratio
				}
			}
		}
		for _, child := range elem.Children {
			walk(child, shape)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node, nil)
		}
	}
	if index != len(g.Boxes) {
		t.Fatalf("순회 %d개, 측정 박스 %d개 — 순회 규칙이 geom.Refs와 어긋났다", index, len(g.Boxes))
	}
	return out
}

// 한 줄에 **여러 박스가 같은 y**에 앉는 배치에서 좌표 대조가 뜻을 유지하는가.
//
// 지금까지 모든 픽스처는 세로 스택이라 박스의 y가 단조 증가했고, "그리드가 그리드다"는 사실은
// 어느 검사도 보지 않았다 — 골든 xfrm은 숫자가 그럴듯한지 말해 주지 않고, 좌표 대조
// (TestFixtureCoordinatesMatchGeometry)는 측정값과 산출물이 같은지만 본다. 측정 자체가
// 어긋나면(트랙 계산이 한 칸씩 밀리면) 두 검사가 함께 통과한다.
//
// 그래서 배치의 뜻을 직접 단정한다: 같은 줄의 카드는 **같은 y**에 앉고(정확히 같은 EMU가
// 산출물에 있다), 4열 그리드의 x는 **등간격**이다. 이것이 TASKS의 리뷰 포인트("좌표 계산
// 0회로 그리드 배치가 PPTX에 정확히 이식되는지")의 자동화 가능한 절반이다 — 나머지 절반
// (편집 가능한 개별 도형인가)은 사람이 PowerPoint에서 본다.
func TestLayoutFixtureSameRowBoxesShareY(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다.
	fixture := fixtureByName(t, layoutFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := PPTXParts(t, data)

	// 슬라이드마다 "첫 줄에 나란히 앉아야 하는 요소"를 적는다. 001은 <row> 안의 두 <col>이고
	// 002는 <grid cols="4">의 카드 4개다 (다섯 번째 카드는 span="2"라 둘째 줄로 내려간다 —
	// 그 사실도 y 그룹이 갈리는 것으로 함께 확인된다).
	for _, tc := range []struct {
		slideID string
		el      string
		want    int
		pitch   bool // 등간격 x까지 보는가 (균등 트랙인 grid만)
	}{
		{slideID: "row-col", el: "h3", want: 2},
		{slideID: "grid-cards", el: "h3", want: 4, pitch: true},
	} {
		t.Run(tc.slideID, func(t *testing.T) {
			index := -1
			for i, slide := range result.Deck.Slides {
				if slide.ID == tc.slideID {
					index = i
				}
			}
			if index < 0 {
				t.Fatalf("슬라이드 %s가 없다", tc.slideID)
			}

			// 첫 줄 = 같은 요소 이름의 박스 중 y가 가장 작은 것들.
			var top []geom.Box
			for _, box := range result.Geometry.Boxes {
				if box.Slide != index || box.El != tc.el {
					continue
				}
				switch {
				case len(top) == 0 || box.Rect.Y < top[0].Rect.Y:
					top = []geom.Box{box}
				case box.Rect.Y == top[0].Rect.Y:
					top = append(top, box)
				}
			}
			if len(top) != tc.want {
				t.Fatalf("<%s>가 같은 y에 %d개, want %d개", tc.el, len(top), tc.want)
			}

			// PPTX의 y는 같은 EMU여야 한다. TextBoxRect의 안전 여유는 가로로만 붙으므로
			// (정렬이 정하는 방향 — pptx.TextBoxRect) y는 잰 값의 환산 그대로다.
			_, y := pptx.Offset(top[0].Rect)
			part := fmt.Sprintf("ppt/slides/slide%d.xml", index+1)
			xml := string(parts[part])
			if xml == "" {
				t.Fatalf("파트 %s가 없다", part)
			}
			if got := strings.Count(xml, fmt.Sprintf("y=\"%d\"", y)); got < tc.want {
				t.Errorf("%s: y=%d인 도형이 %d개, want %d개 이상 — 같은 줄의 박스가 같은 y로 굽히지 않았다",
					part, y, got, tc.want)
			}
			t.Logf("%s: <%s> %d개가 y=%.1fpx(%d EMU)를 공유한다", tc.slideID, tc.el, len(top), top[0].Rect.Y, y)

			if !tc.pitch {
				return
			}
			sort.Slice(top, func(i, j int) bool { return top[i].Rect.X < top[j].Rect.X })
			var pitches []float64
			for i := 1; i < len(top); i++ {
				pitches = append(pitches, top[i].Rect.X-top[i-1].Rect.X)
			}
			t.Logf("%s: 트랙 pitch %v (gap lg 36px 포함)", tc.slideID, pitches)
			for i, pitch := range pitches {
				// 트랙 폭은 (컨테이너 폭 − 3×gap)/4라 소수점이 남는다 — 브라우저가 트랙마다
				// 다른 방향으로 반올림하므로 1px까지는 등간격으로 본다. 한 칸이 밀리면
				// (트랙 계산이 틀리면) 차이는 트랙 폭 단위로 벌어진다.
				if math.Abs(pitch-pitches[0]) > 1.0 {
					t.Errorf("%s: %d번째 pitch %.1fpx가 첫 pitch %.1fpx와 다르다 — 트랙이 등간격이 아니다",
						tc.slideID, i+1, pitch, pitches[0])
				}
			}
		})
	}
}

// shapeRoles의 자리가 geom.Refs와 정확히 맞는가 — 좌표 대조와 도형 수 등식이 둘 다 이
// 자리 맞대기 위에 서 있다 (어긋나면 좌표가 엉뚱한 요소의 변환과 만나고, 그 실패는 산출물을
// 열어 보기 전까지 드러나지 않는다).
//
// 브라우저 없이 도는 검사다: geom.Refs는 덱 모델만 보고 만들어지므로(측정은 그 목록의 자리를
// 채우는 일이다) 순회 두 벌의 대조는 파싱·해석만으로 성립한다. 도형이 든 덱을 여기서
// 직접 만드는 이유는 도형 픽스처가 아직 비활성이기 때문이다 (항목 06이 켠다).
func TestShapeRolesLineUpWithGeometryRefs(t *testing.T) {
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	src := `<slide>
  <ul>
    <li>첫째 <strong>강조</strong></li>
    <li>둘째</li>
  </ul>
  <col gap="md">
    <text>컨테이너 안 텍스트</text>
    <spacer/>
    <shape type="rect" fill="surface" w="240" h="120" pad="sm">
      <h3>도형 안 제목</h3>
      <p anchor="center">도형 안 본문</p>
    </shape>
  </col>
  <shape type="ellipse" fill="accent1" w="120" h="120"/>
  <stack>
    <text at="center">겹침 배지</text>
    <shape type="rect" fill="surface" w="120" h="80" x="-16" y="-8"/>
  </stack>
</slide>`
	doc, diags := soml.Parse("slides/001.slide", src)
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	deck := &soml.Deck{Title: "shape roles", Aspect: soml.Aspect16x9, Stage: stage}
	deck.Slides = append(deck.Slides, &soml.Slide{
		Index: 0, ID: "s1", Source: "slides/001.slide",
		Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens,
	})
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}

	refs := geom.Refs(deck)
	roles := shapeRoles(t, deck)
	if len(roles) != len(refs) {
		t.Fatalf("역할 %d개, 측정 대상 %d개", len(roles), len(refs))
	}
	// 요소 이름 → 기대 분류. 도형 안 텍스트(h3·p)와 슬라이드 직속 텍스트(text)가 **같은
	// 종류인데 다른 분류**인 것이 이 순회의 존재 이유다.
	want := []struct {
		el       string
		excluded string
	}{
		{"ul", ""},
		{"li", "li"},
		{"li", "li"},
		{"col", "컨테이너"},
		{"text", ""},
		{"spacer", "spacer"},
		{"shape", ""},
		{"h3", "도형 안 텍스트"},
		{"p", "도형 안 텍스트"},
		{"shape", ""},
		// `<stack>`은 레이아웃 컨테이너다 — `<col>`과 같은 "컨테이너" 분류로 도형이 되지
		// 않고 자식만 도형이 된다 (Iteration 6). group(Iteration 9)이 열리면 grpSp로
		// 남아 이 분류에서 갈라지지만, stack은 배치 후 소멸하므로 여기 남아야 한다. 이
		// 목록을 좌표 대조와 도형 수 등식이 함께 쓰므로, 이 세 줄이 stack에서 두 계약을
		// 동시에 지킨다 (자리가 한 칸 밀리면 좌표가 엉뚱한 요소와 만난다).
		{"stack", "컨테이너"},
		{"text", ""},
		{"shape", ""},
	}
	if len(want) != len(refs) {
		t.Fatalf("기대표 %d행, 측정 대상 %d개 (%v)", len(want), len(refs), refs)
	}
	for i, ref := range refs {
		if ref.El != want[i].el {
			t.Fatalf("%d번째 측정 대상이 <%s>다 (want <%s>) — 기대표가 순회와 어긋났다",
				i, ref.El, want[i].el)
		}
		role := roles[i]
		if want[i].excluded == "" {
			if role.rect == nil {
				t.Errorf("%d번째 <%s>가 도형이 되지 않는다고 표시되었다 (%s)", i, ref.El, role.excluded)
			}
			continue
		}
		if role.rect != nil {
			t.Errorf("%d번째 <%s>가 도형으로 표시되었다 — want %s", i, ref.El, want[i].excluded)
		}
		if role.excluded != want[i].excluded {
			t.Errorf("%d번째 <%s>의 분류 = %q, want %q", i, ref.El, role.excluded, want[i].excluded)
		}
	}
	// 분류 이름은 등식이 쓰는 어휘여야 한다 (로그의 항 이름이 순회에서 온다).
	for _, role := range roles {
		if role.excluded == "" {
			continue
		}
		found := false
		for _, kind := range shapeExcludedKinds {
			if kind == role.excluded {
				found = true
			}
		}
		if !found {
			t.Errorf("분류 %q가 shapeExcludedKinds에 없다 — 등식의 항이 빠진다", role.excluded)
		}
	}
	// 도형의 rect 변환은 잰 값 그대로이고, 텍스트 박스는 안전 여유를 더한다.
	measured := geom.Rect{X: 64, Y: 64, W: 240, H: 120}
	if got := roles[6].rect(measured); got != measured {
		t.Errorf("도형의 rect가 %s로 변했다 — 저자가 정한 크기를 넓혔다", got)
	}
	if got := roles[4].rect(measured); got == measured {
		t.Error("텍스트 박스에 안전 여유가 붙지 않았다")
	}
}

// countedShapeTags는 **재고 도형이 되는** 요소의 spTree 태그다.
//
// 이름을 손으로 비교하지 않고 목록으로 두는 이유: 종류가 늘 때(Iteration 5의 `p:pic`,
// Iteration 9의 `p:grpSp`) 세는 자리가 하나여야 도형 수 등식과 슬라이드별 표가 같은 것을
// 센다. 두 곳에서 세면 한쪽만 갱신되고, 그 어긋남은 "등식은 통과하는데 표가 깨진다"로 나타나
// 원인이 계약인지 세는 코드인지 알 수 없다.
var countedShapeTags = []string{"<p:sp>", "<p:pic>"}

// countShapes는 슬라이드 파트의 도형 수다 (`p:sp` + `p:pic`).
func countShapes(xml string) int {
	n := 0
	for _, tag := range countedShapeTags {
		n += strings.Count(xml, tag)
	}
	return n
}

// shapeExcludedKinds는 **측정되지만 도형이 되지 않는** 요소의 분류다 (등식의 항 이름,
// 순서 = 등식에 적는 순서). shapeRoles가 붙이고 도형 수 등식이 로그에 그대로 쓴다.
var shapeExcludedKinds = []string{"컨테이너", "spacer", "li", "도형 안 텍스트"}

// shapeRole은 측정된 요소 하나가 PPTX에서 무엇이 되는지다.
type shapeRole struct {
	// rect는 잰 rect → 도형 rect 변환이다. nil이면 이 요소는 도형이 되지 않는다.
	rect func(geom.Rect) geom.Rect
	// excluded는 도형이 되지 않는 요소의 분류다 (shapeExcludedKinds의 값) — 도형이 되면
	// 빈 문자열이다.
	excluded string
}

// shapeRoles는 지오메트리와 같은 문서 순서(geom.Refs — 인라인은 자손까지 건너뛴다)로,
// 측정된 요소마다 PPTX 조립기가 그 도형의 좌표를 만드는 방법을 모은다.
//
// 자기 도형을 갖지 않는 요소는 rect가 nil이다 — `<li>`(컨테이너가 도형 하나다), **도형 안
// 텍스트**(부모 도형의 txBody 문단이다), 레이아웃 컨테이너와 `<spacer>`(도형이 아예 없다).
// 이 목록 하나를 좌표 대조와 도형 수 등식이 함께 쓰는 이유: 두 검사가 다른 판정을 들면
// 한쪽만 계약을 보게 되고, 어긋난 자리에서 좌표가 엉뚱한 요소와 만난다.
func shapeRoles(t *testing.T, deck *soml.Deck) []shapeRole {
	t.Helper()
	var roles []shapeRole
	textBox := func(align soml.TextAlign) func(geom.Rect) geom.Rect {
		return func(r geom.Rect) geom.Rect { return pptx.TextBoxRect(r, align) }
	}
	// 도형은 잰 rect **그대로**다 — 안전 여유는 글자를 잰 박스의 장치이고 도형의 박스는
	// 저자가 정한 크기다 (pptx.writeShapeSp).
	asMeasured := func(r geom.Rect) geom.Rect { return r }

	// inShape는 지금 도형 안을 걷고 있는지다 — 도형 안 텍스트는 태그 이름만으로는 슬라이드
	// 직속 텍스트와 구별되지 않는다 (같은 `<p>`다).
	var walk func(slide *soml.Slide, node soml.Node, inShape bool)
	walk = func(slide *soml.Slide, node soml.Node, inShape bool) {
		elem, ok := node.(*soml.Element)
		if !ok {
			// 문자 데이터는 자기 박스를 갖지 않는다 (geom.collect와 같은 규칙).
			return
		}
		kind := soml.KindOf(elem.Name)
		switch {
		case kind == soml.KindInline:
			// 인라인은 측정 대상이 아니다 — 자손까지 건너뛴다 (geom.collect의 규칙 그대로.
			// 여기가 갈라지면 목록과 Boxes의 자리가 어긋나 좌표가 엉뚱한 요소와 만난다).
			return
		case inShape:
			// 도형 안 텍스트는 **재지만 도형이 되지 않는다** (pptx.shapeBodyXML) — `<li>`와
			// 같은 처지다. 자리를 비워 두어야 뒤따르는 형제의 좌표가 제 변환과 만난다.
			roles = append(roles, shapeRole{excluded: "도형 안 텍스트"})
		case kind == soml.KindShape:
			roles = append(roles, shapeRole{rect: asMeasured})
		case kind == soml.KindImage:
			// 그림도 **재고 도형이 된다** (요소는 `p:pic`이다 — pptx.writePicShape). 좌표는
			// 도형과 같이 잰 rect 그대로다: 잰 rect가 이미 그려진 그림이므로 여백·안전 여유를
			// 더할 것이 없다 (render/web/image.go의 fit 결정).
			roles = append(roles, shapeRole{rect: asMeasured})
		case kind == soml.KindListContainer:
			// 리스트 도형의 안전 여유는 언제나 start 정렬로 적용된다 — li는 align 속성을
			// 받지 않아 확정 정렬이 항상 start다 (pptx.writeListShape의 결정).
			roles = append(roles, shapeRole{rect: textBox(soml.AlignStart)})
		case kind == soml.KindListItem:
			roles = append(roles, shapeRole{excluded: "li"})
		case kind == soml.KindLayoutContainer:
			// 레이아웃 컨테이너와 `<spacer>`는 **측정되지만 도형이 되지 않는다**
			// (pptx.writeShapeNode — Iteration 3의 핵심 계약). 컨테이너는 자식으로
			// 내려가고(아래 순회), spacer는 void라 자식이 없다.
			roles = append(roles, shapeRole{excluded: "컨테이너"})
		case kind == soml.KindSpacer:
			roles = append(roles, shapeRole{excluded: "spacer"})
		default:
			style := elem.TextStyle()
			if style == nil {
				t.Fatalf("%s: <%s>의 외관이 해석되지 않았다", slide.Source, elem.Name)
			}
			roles = append(roles, shapeRole{rect: textBox(style.Align)})
		}
		for _, child := range elem.Children {
			// 도형의 자식은 도형 안이다 — 도형이 자식을 흡수하는 것과 컨테이너가 사라지는
			// 것이 여기서 정반대로 갈린다 (pptx.writeShapeNode의 네 분기).
			walk(slide, child, inShape || kind == soml.KindShape)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(slide, node, false)
		}
	}
	return roles
}

// imageRegionDiffMaxRatio는 **그림 하나**(잰 rect)의 픽셀 오차 허용 비율이다.
//
// 슬라이드 전체 비율(fixtureDiffOptions)과 갈라 두는 이유가 이 iteration의 핵심 계약이다:
// 슬라이드는 대부분 배경이므로 크롭 축이 뒤바뀐 그림 하나는 전체 비율에서 희석된다 (006의
// 전면 그림처럼 큰 그림 하나가 슬라이드의 45%인 슬라이드도 있지만, 001·005는 그림 하나가
// 3~5%다 — 축이 뒤바뀌어도 픽스처 임계값 안에 든다). 그림만 잘라 보면 `cover`가 **다른 부분**을
// 보여주는 것이 곧바로 수치가 된다: 잘린 쪽이 바뀌면 표식(좌상단 흰 사각형)이 크롭 안에서
// 사라지거나 나타나므로 오차가 그림 면적의 수십 %가 된다.
//
// 여유를 넣지 않고 **잰 rect 그대로** 자르는 이유: 그림자·회전 외접 사각형·경계 중앙에
// 그려지는 `a:ln`은 rect 밖으로 24px까지 퍼지는데, 픽스처의 가장 좁은 칸 간격이 `gap="md"`
// = 24px이라 여유를 그만큼 주면 **이웃 그림이 크롭에 들어온다** (도형은 여유 16px < 간격
// 24px으로 그 문제를 피했다 — shapeCropMargin). 밖으로 나간 잉크의 판정은 슬라이드 전체
// 비교가 갖고, 이 검사는 "그림의 어느 부분이 어디에 그려졌는가"만 본다.
//
// 6%는 측정값에 여유를 더한 값이다 (항목 05/05의 측정 — 아래 로그의 표): 22장 중 최대는
// 004_image_text의 `cover` 그림 4.05%이고 그 잔차의 정체는 **리샘플링 커널**이다. 두 렌더러가
// 같은 배율로 줄이지 않으므로(브라우저는 부드럽게, LibreOffice는 거칠게) 표식 경계의 한 줄이
// 서로 다른 픽셀로 떨어진다 — 그림 안이 단색인 2색 에셋을 고른 이유가 이것이고(잔차가 경계에만
// 남는다), 그래서 오차는 그림 크기가 아니라 **경계의 길이**에 비례한다.
const imageRegionDiffMaxRatio = 0.06

// imageRegionDiffMinPixels는 그림 오차의 **절대 하한 허용값**이다 (다른 픽셀 수).
//
// 잔차는 표식 경계에 남으므로 픽셀 수가 그림 **크기와 무의존**한데(리샘플링 커널의 차이는
// 경계 한 줄에서 난다) 비율은 면적으로 나눈 값이다 — 아주 작은 그림에서는 그 한 줄이 큰
// 비율로 계산되어 **방법 문제로 죽는다**: 001_fit의 `dot.png`(16×8 = 128픽셀)에서 경계
// 20픽셀이 13.9%다. 32는 그 측정값에 여유를 더한 값이고, 게이트를 약화하지 않는다 — 크롭
// 축이 갈라지는 `cover` 그림은 전부 17,000픽셀 이상이라 6%가 1,000픽셀을 넘고, 그 그림들에서
// 이 하한은 판정에 닿지 않는다. loLineWidthMinPx가 짧은 잉크 띠에 둔 하한과 같은 종류의
// 장치다 (그쪽은 AA 잡음 1px, 이쪽은 경계 한 줄).
const imageRegionDiffMinPixels = 32

// 그림이 두 타깃에서 **같은 영역을 같은 자리에** 보여주는가 — 그림마다 잰다.
//
// 이 iteration의 가장 큰 판정이다 (TASKS Iteration 5의 리뷰 포인트 "cover 크롭이 양 타깃에서
// 같은 영역을 보여주는지"의 자동 절반). 웹은 `object-fit: cover`가 중앙을 보여주고 PPTX는
// `srcRect`로 양쪽에서 같은 몫을 자르는데(pptx.coverCrop), 두 계산이 어긋나면 **같은 그림의
// 다른 부분**이 같은 자리에 그려진다 — 좌표 대조(xfrm == 측정 EMU)는 그것을 볼 수 없다.
func TestImageFixturePictureRegions(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, imageFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

	worst, worstAt := 0.0, ""
	pictures := 0
	for i, slide := range result.Deck.Slides {
		var boxes []geom.Box
		for _, box := range result.Geometry.Boxes {
			if box.Slide == i && box.El == soml.ImageElement {
				boxes = append(boxes, box)
			}
		}
		if len(boxes) == 0 {
			continue
		}
		webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(i))
		loImg := shots[i]
		for _, box := range boxes {
			cmp, err := Diff(cropImage(webImg, box.Rect), cropImage(loImg, box.Rect), DefaultDiffOptions())
			if err != nil {
				t.Fatalf("%s/%s: %v", slide.ID, box.ID, err)
			}
			pictures++
			t.Logf("%-16s %-10s %s: %s", slide.ID, box.ID, box.Rect, cmp)
			if cmp.Ratio > worst {
				worst, worstAt = cmp.Ratio, slide.ID+"/"+box.ID
			}
			if cmp.Ratio > imageRegionDiffMaxRatio && cmp.Changed > imageRegionDiffMinPixels {
				t.Errorf("%s/%s: 그림 안 픽셀 오차 %.2f%%가 허용 %.2f%%를 넘는다 — 크롭이 갈라졌다면 "+
					"고칠 곳은 오차가 아니라 웹의 `fit` CSS(render/web.imageFitCSS)와 크롭 계산(pptx.coverCrop)이다",
					slide.ID, box.ID, cmp.Ratio*100, imageRegionDiffMaxRatio*100)
			}
		}
	}
	if pictures != imageFixturePictures {
		t.Errorf("그림 %d개를 쟀다, want %d개 — 픽스처가 바뀌면 이 수를 함께 고친다", pictures, imageFixturePictures)
	}
	t.Logf("가장 큰 그림 오차: %s %.2f%% (허용 %.2f%%, 그림 %d개)",
		worstAt, worst*100, imageRegionDiffMaxRatio*100, pictures)
}

// imageFixturePictures는 그림 픽스처의 `<image>` 수다 (문서 전체).
//
// 수를 못 박는 이유: 위 검사는 측정 목록을 걷다가 그림을 찾으므로, 순회가 그림을 놓치기
// 시작하면 **아무것도 비교하지 않고 통과한다** (측정 목록에서 `<image>`가 빠지는 실패가
// 정확히 그 모양이다). 슬라이드별 내역은 001 5 + 002 6 + 003 3 + 004 1 + 005 5 + 006 1 +
// 007 3 = 24다.
const imageFixturePictures = 24

// 산출 에셋과 미디어 파트가 **입력 파일과 바이트가 같은가** — 엔진은 그림을 고치지 않는다.
//
// 골든이 이미 두 자리의 바이트를 갖고 있지만(웹 `assets/*`·PPTX `ppt/media/*`) 골든은
// "지난번과 같은가"만 말한다: 엔진이 리사이즈·재압축을 시작하면 골든을 한 번 갱신한 뒤로는
// 아무도 그것을 보지 않는다. 이 검사는 기준을 **커밋된 입력 파일**에 두므로 그 경로가 없다.
//
// 미디어 파트 이름의 기대값을 에셋 표에서 만드는 것도 계약이다: 파트 번호는 표의 자리이고
// 확장자는 **포맷**에서 나온다 (NOTES 05/04 — `.jpg`·`.jpeg`가 한 Content_Types Default로
// 모이게 하는 결정). 이름을 손으로 적으면 그 두 규칙이 테스트에서 사라진다.
func TestImageFixtureAssetsAreByteIdentical(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다.
	fixture := fixtureByName(t, imageFixtureName)
	result := fixture.Build(t)

	if len(result.Deck.Assets) == 0 {
		t.Fatal("그림 픽스처의 에셋 표가 비었다")
	}
	data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := PPTXParts(t, data)
	media := map[string][]byte{}
	for name, part := range parts {
		if strings.HasPrefix(name, "ppt/media/") {
			media[name] = part
		}
	}

	for i, asset := range result.Deck.Assets {
		want, err := os.ReadFile(filepath.Join(fixture.Dir, filepath.FromSlash(asset.Key)))
		if err != nil {
			t.Fatalf("픽스처 에셋을 읽을 수 없다: %v", err)
		}
		// 웹: 산출 디렉토리에 저자가 쓴 경로 그대로 나간다 (render/web.assetFiles).
		got, err := os.ReadFile(filepath.Join(result.OutDir, filepath.FromSlash(asset.Key)))
		if err != nil {
			t.Errorf("웹 산출에 에셋이 없다: %v", err)
		} else if !bytes.Equal(got, want) {
			t.Errorf("웹 산출 %s의 바이트가 입력과 다르다 (%dB vs %dB) — 엔진이 그림을 고쳤다",
				asset.Key, len(got), len(want))
		}
		// PPTX: 파트 이름은 표의 자리 + 포맷의 확장자다.
		part := fmt.Sprintf("ppt/media/image%d%s", i+1, asset.Format.Ext())
		if got, ok := media[part]; !ok {
			t.Errorf("미디어 파트 %s가 없다 (있는 것: %v)", part, mediaNames(media))
		} else if !bytes.Equal(got, want) {
			t.Errorf("%s의 바이트가 %s와 다르다 (%dB vs %dB) — 엔진이 그림을 고쳤다",
				part, asset.Key, len(got), len(want))
		}
		delete(media, part)
		t.Logf("%-16s → %s (%dB, %dx%d)", asset.Key, part, len(want), asset.Width, asset.Height)
	}
	// 표에 없는 파트가 남으면 쓰이지 않는 그림이 패키지에 실린 것이다 (파일이 부푼다).
	for name := range media {
		t.Errorf("에셋 표에 없는 미디어 파트가 패키지에 있다: %s", name)
	}
}

// mediaNames는 진단용 파트 이름 목록이다 (정렬순 — 맵 순회 순서가 메시지를 흔들지 않게).
func mediaNames(media map[string][]byte) []string {
	names := make([]string, 0, len(media))
	for name := range media {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LibreOffice 렌더가 견뎌야 하는 글줄 오차의 임계값 (PRD §13-1의 수치 — 이 테스트가 잰
// 값에 여유를 더해 못 박은 회귀 감시선이다. 측정값은 NOTES.md 01/07의 표에 있다).
const (
	// loLineOffsetMaxPx는 글줄 잉크 세로 중심의 허용 오차다. spcPts(절대 줄간격)로 줄 전진
	// 폭은 맞췄지만 첫 기준선 배치는 엔진 소관이라 0이 되지 않는다 — LibreOffice가 일관되게
	// 아래로 +2~5px(측정 최대 +5.0px, 캡션 20px)이고, 6px은 거기에 픽셀 반올림 여유를 더한
	// 값이다. 이 선을 넘으면 줄간격 매핑이 다시 갈라진 것이다 (spcPct 시절에는 +8~13px였다).
	loLineOffsetMaxPx = 6.0
	// loLineWidthMaxRatio는 글줄 잉크 폭의 허용 오차 비율이다. LibreOffice는 한글↔라틴
	// 경계마다 스크립트 간격을 넣어 혼합 문자 줄을 넓게 그린다 — 01_text의 긴 본문에서 +3.3%
	// (측정값). 4%는 텍스트 박스의 안전 여유(pptx.TextBoxRect)와 같은 값이다.
	//
	// **비율은 이 간격의 자연스러운 단위가 아니다**: 간격은 줄 길이가 아니라 **경계 수**에
	// 비례하므로(경계당 +0.245 em — script_spacing_test.go가 잰다) 같은 문장을 좁은 트랙에
	// 넣으면 비율만 커진다. 그것이 아래 loLineWidthMaxRatios가 픽스처마다 조금씩 커진 이유이고,
	// 이 상수를 그만큼 키울 수 없는 이유는 여유를 키우면 반대 방향으로 줄이 줄어들기
	// 때문이다 (docs/guides/known-differences.md).
	loLineWidthMaxRatio = 0.04
	// loLineWidthMinPx는 폭 오차의 절대 하한 허용값이다. 안티앨리어싱은 줄 길이와 무관하게
	// 글리프 경계에서 ±1px의 잉크를 만든다 — 비율 임계값은 긴 줄에서 측정한 값이라, 아주
	// 짧은 띠(pre의 `}` 한 글자, 잉크 폭 6px)에서는 그 1px이 16.7%로 계산되어 방법 문제로
	// 죽는다. 2px은 AA 잡음의 상한이고, 긴 줄에서는 비율 허용값이 항상 이보다 크므로
	// 게이트를 약화하지 않는다 (Iteration 2 항목 08의 측정).
	loLineWidthMinPx = 2.0
)

// loLineWidthMaxRatios는 픽스처별 글줄 잉크 폭 비율의 조정이다 — 없는 픽스처는
// loLineWidthMaxRatio(4%)를 쓴다. 조정은 측정 수치와 근거가 있을 때만 더한다.
//
// 02_text_subset: 측정 최대 +5.0% (003_inline의 code+링크 문단 — Iteration 2 항목 08).
// LibreOffice의 스크립트 간격은 한글↔라틴 **경계 수**에 비례하는데 비율 임계값은 줄
// 길이로 나눈 값이라, 짧은 줄에 경계가 몰리면(545px 줄에 mono 런 2회 전환 + 라틴
// 문장부호 경계 — 웹은 넣지 않는 간격 5곳) 01_text의 긴 본문(1125px, +3.3%)보다 비율이
// 커진다. 구조 갈라짐(줄 수·재접힘)은 0 오차의 줄 수 검사가 따로 잡고, 이 픽스처의 최소
// 여유 비율은 40.5%라 5%대 폭 오차로 줄이 늘 위험은 없다. 5.5%는 측정값에 01_text와
// 같은 폭의 여유를 더한 값이다 (3.3% 측정 → 4% 허용과 같은 태도).
var loLineWidthMaxRatios = map[string]float64{
	textSubsetFixtureName: 0.055,
	// 03_layout: 측정 최대 +5.8% (002_grid_cards의 두 칸 카드 문단 — Iteration 3 항목 05).
	// 02_text_subset과 같은 원인(스크립트 간격은 한글↔라틴 **경계 수**에 비례하고 비율은 줄
	// 길이로 나눈 값이다)이 배치에서 더 세게 나타난다: 트랙이 좁아 줄이 짧아지는데
	// (381px — 01_text 본문의 3분의 1) 그 짧은 줄에 `grid-column: span 2`의 라틴 런과
	// 인용 부호가 몰려 경계가 6곳이다. 이 줄의 박스는 두 칸 트랙(558px)이라 여유 비율이
	// 46%이므로 5.8%의 폭 오차로 줄이 접힐 위험은 없다 (구조 갈라짐은 0 오차의 줄 수 검사가
	// 따로 잡는다). 6.5%는 측정값에 앞 픽스처들과 같은 폭의 여유를 더한 값이다.
	layoutFixtureName: 0.065,
	// 06_stack: 측정 최대 +5.5% (010 content-explicit의 첫 문단 — 항목 06). 겹침이 아니라
	// **겹치지 않은 텍스트 슬라이드**의 잔차다 (이 픽스처에서 잉크 띠 판정에 남는 유일한
	// 슬라이드다 — inkBandSkipSlides가 001~009만 뺀다). 원인은 02·03과 같다: LibreOffice의
	// 스크립트 간격은 한글↔라틴 **경계 수**에 비례하는데 비율은 줄 길이로 나눈 값이라, 짧은
	// 문단(`layout="content"는 기본 스테이지의 이름이다.` — 라틴 런 `layout="content"` +
	// 인용부호가 한글 문장에 낀다)에서 비율이 커진다. 줄 수 갈라짐은 0 오차의 줄 수 검사가
	// 따로 잡고(이 슬라이드는 줄 수가 같다), 6.0%는 측정값에 앞 픽스처들과 같은 폭의 여유를
	// 더한 값이다 (02 5.0%→5.5%, 03 5.8%→6.5%와 같은 태도).
	stackFixtureName: 0.06,
}

// inkBandSkipSlides는 잉크 띠 판정을 **적용할 수 없는** 슬라이드다 (`<픽스처>/<슬라이드 id>`).
//
// InkBands는 "영역에서 가장 흔한 색이 배경이고 잉크는 글자다"에 기대므로, 저자가 배경을 칠한
// 슬라이드에서는 그 사각형이 하나의 거대한 띠가 되어 글줄 판정 자체가 성립하지 않는다 —
// 값을 완화하는 문제가 아니라 **측정 방법의 전제가 깨지는** 문제다 (웹 띠 1152×40px = 칠한
// 띠, LibreOffice 띠 737×22px = 글자. 폭 차 −36%).
//
// 판정 범위를 박스별로 좁혀도 마찬가지다: 칠한 것은 부모 `<row>`이고 그 사각형이 자식 박스를
// 덮는다. 이 슬라이드의 배치는 좌표로 0 오차 대조된다
// (TestFixtureCoordinatesMatchGeometry — xfrm == 측정 EMU).
var inkBandSkipSlides = map[string]bool{
	// 007_style_appearance: `style="background: #0F65F9"`가 웹에만 적용된다 (§7.9의 계약).
	"03_layout/style-appearance-warning": true,
	// 04_shape 전 슬라이드: 도형이 들어오면 이 방법의 전제가 슬라이드 단위로 깨진다.
	// 칠한 도형은 03_layout/007(저자가 `style`로 칠한 띠)이 이미 깨뜨린 전제이고, 도형
	// 픽스처는 **대부분의 슬라이드가 그렇다** — 잰 줄 하나마다 띠가 하나 있어야 하는 전제
	// 대신, 도형 하나가 채움으로 하나의 큰 띠가 되고 그 안의 글줄은 띠 안에 묻힌다
	// (측정: 001의 첫 줄 도형 넉 장이 1103×112px 띠 **하나**가 된다 — 글줄 넷이 아니다).
	// 값 완화가 아니라 **방법의 전제가 깨진 것**이므로 슬라이드 단위로 건너뛴다.
	//
	// 대신 줄 수 계약을 잃지 않는다: 도형 **안쪽**에서는 가장 흔한 색이 채움이므로 같은
	// 방법이 성립한다 — TestShapeFixtureInShapeTextLineCounts가 도형마다 그것을 본다
	// (그 검사가 이 픽스처에서 줄바꿈 갈라짐의 판정자다). 슬라이드 직속 텍스트(제목)의
	// 줄바꿈은 01~03 픽스처가 같은 계약으로 이미 지킨다.
	"04_shape/shape-types":           true,
	"04_shape/shape-token-colors":    true,
	"04_shape/shape-literal-colors":  true,
	"04_shape/shape-appearance":      true,
	"04_shape/shape-text-anchor":     true,
	"04_shape/shape-cards":           true,
	"04_shape/shape-variant":         true,
	"04_shape/shape-variant-unknown": true,
	// 05_image 전 슬라이드: 그림이 들어오면 전제가 **도형보다 더 크게** 깨진다. 도형의 채움은
	// 잰 rect 하나를 덮지만 그림은 그 자체가 큰 색면이고, 무엇보다 **잰 줄이 없는 자리에
	// 잉크가 있다** — 그래서 판정이 "허용 오차를 넘었다"가 아니라 `띠에 주인이 없다`로
	// 죽는다 (측정: 001의 첫 줄 그림 네 장이 1068×202px 띠 **하나**가 되고, 006의 전면 그림은
	// 1152×360px 띠 하나다. 004처럼 오른쪽에 글이 있는 슬라이드에서는 그림 띠가 그 글줄과
	// 겹쳐 **주인을 찾아 통과하는데**, 그때 재는 것은 글자가 아니라 그림이다 — 값 완화보다
	// 나쁜, 뜻이 뒤바뀐 통과다).
	//
	// 이 픽스처는 일곱 슬라이드 전부에 그림이 있어 "그림이 없는 슬라이드 한정"으로 좁힐
	// 자리가 없다. 잃는 것과 남는 것: 슬라이드 직속 텍스트(제목·캡션)의 줄바꿈 계약은
	// 01~03 픽스처가 같은 검사로 이미 지키고, 그림의 자리·크기는 좌표 대조가 0 오차로
	// (TestFixtureCoordinatesMatchGeometry), 그림의 **내용**은 그림별 크롭 비교가
	// (TestImageFixturePictureRegions) 판정한다.
	"05_image/image-fit":        true,
	"05_image/image-appearance": true,
	"05_image/image-formats":    true,
	"05_image/image-text":       true,
	"05_image/image-grid":       true,
	"05_image/image-full":       true,
	"05_image/image-variant":    true,
	// 06_stack의 겹침 슬라이드 001~009: 겹침은 04·05가 깬 전제(칠한 도형·그림이 단색 배경
	// 가정을 깬다)를 **한 단계 더** 깬다 — 겹친 자리에는 도형이 여러 층 포개져 **띠에 주인이
	// 하나가 아니다**. 잰 줄 하나 ↔ 잉크 띠 하나의 대응이 성립하지 않으므로(포갠 도형들이
	// 하나의 큰 띠로 뭉치거나, 위 도형이 아래 도형의 글줄을 덮는다) 값 완화가 아니라 방법의
	// 전제가 깨진 것이다. 슬라이드 단위로 건너뛴다 (04가 8장, 05가 7장을 같은 이유로 뺐다).
	//
	// 대신 잃는 것과 남는 것: 겹침 순서는 spTree 순서 전수와 겹친 영역 크롭 비교가
	// (TestStackFixtureZOrderIsDocumentOrder·TestStackFixtureOverlapRegions), 좌표는 0 오차
	// 대조가(TestFixtureCoordinatesMatchGeometry) 판정한다. 무엇보다 **010
	// content-explicit는 겹침이 없는 순수 텍스트 슬라이드**라 단색 배경 가정이 성립하므로
	// 잉크 띠 판정에 **남긴다** (아래 표에 없다) — 픽스처 전체를 통째로 빼지 않는다는 것이
	// 이 슬라이드가 픽스처에 있는 이유다 (제목·본문·리스트의 줄바꿈 계약을 이 픽스처에서도
	// 지킨다).
	"06_stack/anchors":       true, // surface 배경 + accent1 배지 9장이 면을 덮는다
	"06_stack/badge-overlay": true, // 그림 위 배지 — 그림이 큰 색면이다
	"06_stack/watermark":     true, // 반투명 워터마크 도형이 본문 아래에 깔린다
	"06_stack/z-order":       true, // 세 도형이 계단식으로 포개진다 (띠에 주인이 여럿)
	"06_stack/absolute":      true, // surface 배경 + x/y 도형들
	"06_stack/in-container":  true, // 그림 카드 3장 + 가로 열 안 도형
	"06_stack/nested":        true, // 중첩 stack — 도형 안 도형이 면을 덮는다
	"06_stack/canvas-free":   true, // 캔버스 자유 배치 도형들
	"06_stack/canvas-stack":  true, // 캔버스 위 stack + 도형
}

// 두 렌더 엔진이 같은 자리에서 줄을 나누는가 — Iteration 1의 리뷰 포인트를 수치로 판정한다
// (PRD §13-1). 지오메트리 패스의 줄 박스(geom.Box.Lines)를 슬라이드 단위로 합쳐 기대 띠를
// 만들고, 웹 스크린샷과 LibreOffice 렌더 각각의 잉크 띠와 맞댄다. Iteration 2부터는 인라인
// 서식이 낀 줄(굵은 런은 글자 폭이 다르다)과 리스트·pre까지 같은 계약을 받는다.
//
// 요소 박스 안에서 재지 않고 슬라이드 전체에서 재는 이유: 박스는 잰 줄 수만큼의 높이로
// 고정되므로, 다른 엔진이 줄을 하나 더 만들면 그 줄은 **박스 밖**에 그려진다 — 박스 안만
// 보면 늘어난 줄이 측정에서 빠져 갈라짐이 통과한다 (01_text의 본문이 처음에 정확히
// 그렇게 조용히 통과했다).
func TestFixtureLineBreaksMatchLibreOffice(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	for _, name := range fixtureNames {
		t.Run(name, func(t *testing.T) {
			widthMaxRatio := loLineWidthMaxRatio
			if override, ok := loLineWidthMaxRatios[name]; ok {
				widthMaxRatio = override
			}
			fixture := fixtureByName(t, name)
			result := fixture.Build(t)
			if result.Geometry == nil {
				t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
			}
			stage := result.Deck.Stage
			stageRect := geom.Rect{W: float64(stage.Width), H: float64(stage.Height)}
			shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))

			browser := NewBrowser(t)
			browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

			minRatio := 1.0
			for _, box := range result.Geometry.Boxes {
				if box.LineCount() == 0 {
					continue
				}
				if ratio := box.Slack() / box.MaxLineWidth(); ratio < minRatio {
					minRatio = ratio
				}
			}
			t.Logf("최소 여유 비율 Slack/MaxLineWidth = %.2f%% — 이 덱이 견디는 글자 폭 오차", minRatio*100)

			for i, slide := range result.Deck.Slides {
				if inkBandSkipSlides[name+"/"+slide.ID] {
					t.Logf("%s: 잉크 띠 판정을 건너뛴다 — 저자가 칠한 배경이 단색 배경 가정을 깬다 (inkBandSkipSlides)", slide.Source)
					continue
				}
				lines := measuredLines(result.Geometry, i)
				if len(lines) == 0 {
					continue
				}

				// 웹 스크린샷은 잰 줄 박스와 같은 엔진의 그림이다 — 여기가 어긋나면 잉크 띠 측정
				// 자체가 틀린 것이므로, LibreOffice 판정보다 먼저 방법을 검증한다: 잰 줄마다
				// 웹 잉크가 있어야 하고(줄 → 띠), 잉크 띠마다 주인이 있어야 한다(띠 → 줄).
				webBands := InkBands(browser.ScreenshotElement(t, web.SlideIndexSelector(i)), stageRect)
				owners, ok := bandLineOwners(t, slide.Source, webBands, lines)
				if !ok {
					continue
				}

				loBands := InkBands(shots[i], stageRect)
				if len(loBands) != len(webBands) {
					t.Errorf("%s: 줄 수가 갈라진다 — 브라우저 %d줄, LibreOffice %d줄 (%s)",
						slide.Source, len(webBands), len(loBands), strings.Join(flatOwners(owners), ", "))
					continue
				}
				// 띠끼리의 비교는 웹 잉크 vs LibreOffice 잉크다 — 줄 박스(glyph box)는 잉크보다
				// 크므로(어센더·디센더의 빈 행) 줄 박스와 잉크의 중심을 맞대면 그 차이가 엔진
				// 오차처럼 보인다. 잰 줄은 띠의 주인 이름에만 쓴다.
				for j, wb := range webBands {
					lo := loBands[j]
					who := strings.Join(owners[j], "+")
					offset := (lo.Y + lo.H/2) - (wb.Y + wb.H/2)
					widthDelta := lo.W/wb.W - 1
					t.Logf("%s %s: 세로 중심 %+.1fpx, 잉크 폭 %+.1f%% (웹 %s, LibreOffice %s)",
						slide.Source, who, offset, widthDelta*100, wb, lo)
					if math.Abs(offset) > loLineOffsetMaxPx {
						t.Errorf("%s %s: 글줄이 %+.1fpx 밀렸다 (허용 ±%.0fpx)",
							slide.Source, who, offset, loLineOffsetMaxPx)
					}
					// 폭 판정은 비율과 절대 하한 중 큰 쪽이다 — 비율은 스크립트 간격(줄
					// 길이에 비례하는 오차)을, 하한은 AA 잡음(줄 길이와 무관한 ±1px)을 잰다.
					if allowed := math.Max(widthMaxRatio, loLineWidthMinPx/wb.W); math.Abs(widthDelta) > allowed {
						t.Errorf("%s %s: 글줄 잉크 폭이 %+.1f%% 다르다 (허용 ±%.1f%%)",
							slide.Source, who, widthDelta*100, allowed*100)
					}
				}
			}
		})
	}
}

// measuredLine은 지오메트리가 잰 글줄 하나와 그 주인 요소다.
type measuredLine struct {
	rect geom.Rect
	id   string
}

// measuredLines는 슬라이드의 모든 줄 박스를 위에서 아래 순서로 모은다.
func measuredLines(g *geom.Geometry, slide int) []measuredLine {
	var lines []measuredLine
	for _, box := range g.Boxes {
		if box.Slide != slide {
			continue
		}
		for _, line := range box.Lines {
			lines = append(lines, measuredLine{rect: line, id: box.ID})
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].rect.Y < lines[j].rect.Y })
	return lines
}

// bandLineOwners는 잰 줄을 **웹 잉크 띠**에 배정하고, 띠마다 주인 요소 이름을 돌려준다.
// 배정이 성립하지 않으면(줄 자리에 잉크가 없거나, 주인 없는 잉크가 있으면) 측정 방법이 틀린
// 것이므로 실패로 보고하고 false를 돌려준다.
//
// 왜 줄 박스끼리 미리 합치지 않는가 (Iteration 3 항목 05에서 고친 것): 겹치는 줄 박스를 합치는
// 규칙은 세로 스택에서만 맞는다. 배치가 열리면 한 줄에 여러 박스가 앉고 그 박스들의 y가 조금씩
// 다른데(`<col gap="xs">` 안의 두 문단은 6px씩 엇갈린다), 줄 박스는 잉크보다 커서
// (어센더·디센더의 빈 행) 이웃한 두 줄의 박스가 서로 닿는다 — 합치기가 **전이적으로** 번져
// 004_child_attrs에서 세 줄이 한 띠가 되었고, 실제 그림은 세 띠였다 (방법 문제로 죽었다).
//
// 그래서 합치는 기준을 잰 rect가 아니라 **그림**에 둔다: 잉크 띠는 그 엔진이 실제로 그린
// 줄이고, 줄 박스는 세로로 가장 많이 겹치는 띠의 주인이 된다. 슬라이드 전체에서 재는 것은
// 그대로다 — LibreOffice가 줄을 하나 더 만들면 그 줄은 박스 밖에 그려지므로, 박스 안만 보면
// 갈라짐이 조용히 통과한다 (01_text의 본문이 처음에 정확히 그렇게 통과했다).
func bandLineOwners(t *testing.T, source string, bands []geom.Rect, lines []measuredLine) ([][]string, bool) {
	t.Helper()
	owners := make([][]string, len(bands))
	ok := true
	for _, line := range lines {
		best, overlap := -1, 0.0
		for i, band := range bands {
			top := math.Max(line.rect.Y, band.Y)
			bottom := math.Min(line.rect.Bottom(), band.Bottom())
			if bottom-top > overlap {
				best, overlap = i, bottom-top
			}
		}
		if best < 0 {
			t.Errorf("%s %s: 잰 줄 %s 자리에 웹 잉크가 없다 — 측정 방법이 틀렸다",
				source, line.id, line.rect)
			ok = false
			continue
		}
		owners[best] = appendOwner(owners[best], line.id)
	}
	for i, band := range bands {
		if len(owners[i]) == 0 {
			t.Errorf("%s: 웹 잉크 띠 %s에 주인이 없다 — 잰 줄이 없는 자리에 그림이 있다 (측정 방법이 틀렸다)",
				source, band)
			ok = false
		}
	}
	return owners, ok
}

// flatOwners는 진단 메시지용 요소 이름 목록이다 (띠 경계를 지운 평면 목록).
func flatOwners(owners [][]string) []string {
	var ids []string
	for _, band := range owners {
		ids = appendOwner(ids, band...)
	}
	return ids
}

// appendOwner는 중복 없이 요소 이름을 더한다 — 한 요소의 문자 노드 여러 개가 같은 줄에
// 있을 때(인라인이 쪼갠 경우) 진단 메시지에 같은 이름이 반복되지 않게 한다.
func appendOwner(ids []string, more ...string) []string {
	for _, id := range more {
		seen := false
		for _, have := range ids {
			if have == id {
				seen = true
				break
			}
		}
		if !seen {
			ids = append(ids, id)
		}
	}
	return ids
}

// cNvPrPattern은 spTree의 도형 이름표다 (id + name). id=1은 spTree 그룹(nvGrpSpPr)이라 뺀다.
var cNvPrPattern = regexp.MustCompile(`<p:cNvPr id="(\d+)" name="([^"]*)"`)

// boxIDPath는 도형 이름표의 **문서 위치 경로**를 정수 튜플로 판다 ("shape s3.1.2" → [3 1 2]).
//
// 박스 id는 web.ElementID가 문서 순서(전위 순회)로 매기므로(`s<슬라이드>.<자식>.<자식>…`),
// 이 튜플의 사전식 비교(짧은 접두가 앞)가 곧 문서 순서다 — 즉 겹침 z-순서다.
func boxIDPath(t *testing.T, name string) []int {
	t.Helper()
	fields := strings.Fields(name)
	if len(fields) == 0 {
		t.Fatalf("도형 이름 %q가 비었다", name)
	}
	id := strings.TrimPrefix(fields[len(fields)-1], "s")
	var path []int
	for _, part := range strings.Split(id, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			t.Fatalf("도형 이름 %q에서 문서 위치 경로를 읽을 수 없다: %v", name, err)
		}
		path = append(path, n)
	}
	return path
}

// lessDocPath는 전위 순회(문서) 순서 비교다 — 사전식이되 짧은 접두가 앞이다 (부모가 자식보다
// 앞). 두 경로가 같으면 false다 (같은 요소는 두 번 나오지 않아야 한다).
func lessDocPath(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

// spTreeShapeNames는 골든 슬라이드 파트의 도형(p:sp·p:pic) 이름표를 **등장 순서**로 돌려준다
// (spTree 그룹 id=1 제외).
func spTreeShapeNames(xml string) []string {
	var names []string
	for _, m := range cNvPrPattern.FindAllStringSubmatch(xml, -1) {
		if m[1] == "1" {
			continue // spTree 그룹(nvGrpSpPr) — 도형이 아니다
		}
		names = append(names, m[2])
	}
	return names
}

// 겹침 슬라이드에서 spTree의 도형 등장 순서 = 문서 순서 = z-순서인가 (PRD §13-3).
//
// 항목 05의 단위 테스트(render/pptx.TestRenderStackZOrderIsDocumentOrder)가 합성 덱으로 본
// 것을 **실제 픽스처 전수**로 한 번 더 본다 — TestLayoutFixtureSameRowBoxesShareY가 레이아웃
// 픽스처에 한 것과 같은 자리·같은 성격의 검사다. OOXML은 나중 도형을 위에 그리므로 저자가 쓴
// 문서 순서가 곧 겹침 순서가 되어야 하고, 특히 항목 04가 손댄 배합(z-order 슬라이드 — `x`/`y`
// 좌표 자식과 `at`·정적 자식을 한 stack에 섞은 자리, 웹의 칠 순서와 갈라지는 유일한 슬라이드)
// 에서 그 순서가 유지되는지 본다. 웹의 칠 순서는 픽셀 비교(TestStackFixtureOverlapRegions)가
// 보고, 여기서는 산출 XML의 순서를 본다. 브라우저 없이 골든만 읽는다.
func TestStackFixtureZOrderIsDocumentOrder(t *testing.T) {
	goldenDir := fixtureByName(t, stackFixtureName).GoldenDir()
	for i := 1; i <= slideCounts[stackFixtureName]; i++ {
		slide := fmt.Sprintf("slide%d.xml", i)
		data, err := os.ReadFile(filepath.Join(goldenDir, "pptx", "ppt", "slides", slide))
		if err != nil {
			t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
		}
		xml := string(data)
		id := goldenSlideID(t, xml)
		names := spTreeShapeNames(xml)
		if len(names) == 0 {
			t.Errorf("%s(%s): spTree에 도형이 없다", slide, id)
			continue
		}
		var prev []int
		for _, name := range names {
			path := boxIDPath(t, name)
			if prev != nil && !lessDocPath(prev, path) {
				t.Errorf("%s(%s): 도형 %q가 앞 도형보다 문서상 뒤가 아니다 — spTree 순서 ≠ 문서 순서(z-순서)\n  순서: %v",
					slide, id, name, names)
			}
			prev = path
		}
		t.Logf("%s(%s): 도형 %d개가 문서 순서로 spTree에 나온다 (%s)",
			slide, id, len(names), strings.Join(names, ", "))
	}
}

// stackOverlapRegionDiffMaxRatio는 겹친 영역 크롭의 픽셀 오차 허용 비율이다.
//
// 슬라이드 전체 비율(fixtureDiffOptions·slideDiffOptions)과 갈라 두는 것이 이 항목의 핵심
// 계약이다: 슬라이드는 대부분 배경이라 위아래가 뒤바뀐 겹침 하나는 전체 비율에서 희석된다
// (겹친 면적만큼만 튀므로 z-order 슬라이드의 전체 오차는 0.54%다). 겹친 사각형만 잘라 보면
// 위 도형이 아래 도형으로 바뀐 것이 곧바로 수치가 된다 — 겹친 영역 안쪽은 위 도형의 단색
// 채움이라 두 타깃이 같으면 오차가 잔차(AA·리샘플링)뿐이고, 순서가 뒤집히면 **영역 전체**가
// 다른 색이 되어 수십 %로 튄다.
//
// 값은 측정값에 여유를 더한 것이다 (항목 06의 측정). z-order 슬라이드의 겹친 쌍 셋 중
// **핵심은 muted∩accent1** — 위치 지정(x/y) 자식 위에 정적(at=center) 자식이 문서상 뒤로 얹힌
// 자리다 (항목 04가 position:relative로 문서 순서를 되돌린 그 divergence). 그 영역(136×156px)의
// 측정 오차는 2.63%이고, accent1∩fg(112×102px)는 1.42%다 — 겹친 영역 안쪽은 위 도형의 단색
// 채움이라 잔차는 채움 위 AA와 그림 리샘플링뿐이다. 6%는 거기에 여유를 더한 값이고
// (imageRegionDiffMaxRatio와 같은 값·같은 성질의 잔차다), 순서가 뒤집히면 영역 전체가 다른
// 색이 되어 100%에 가깝게 튀므로(2.63% → ~100%) 이 임계값은 여전히 뒤집힘을 잡는다.
const stackOverlapRegionDiffMaxRatio = 0.06

// stackOverlapRegionMinPixels는 겹친 영역 오차의 **절대 하한 허용값**이다 (다른 픽셀 수) —
// 아주 작은 겹침에서 잔차 몇 픽셀이 큰 비율로 계산되어 방법 문제로 죽는 것을 막는다
// (imageRegionDiffMinPixels·loLineWidthMinPx와 같은 종류의 장치). 순서가 뒤집힌 겹침은
// 영역 전체가 바뀌므로 이 하한에 걸리지 않는다.
const stackOverlapRegionMinPixels = 32

const (
	// stackOverlapInset은 겹친 사각형을 안으로 물리는 폭이다 (px) — 위 도형의 테두리
	// (경로 중앙 1px, w=9525 EMU)와 AA가 크롭을 가로지르지 않게 안쪽 채움만 남긴다.
	stackOverlapInset = 4.0
	// stackOverlapMinSide는 크롭이 뜻을 갖는 최소 변 길이다 (px). 이보다 얇은 겹침은 두 도형이
	// 맞닿는 **색 경계**(예: muted|fg 세로 경계)가 영역을 세로로 가로지르고, 그 경계 한 줄을 두
	// 래스터라이저가 서로 다르게 리샘플링해(그림 리샘플링과 같은 잔차) 얇은 띠에서 큰 비율이 된다
	// — 측정: muted∩fg의 16×66px 슬리버가 그 경계 한 줄(≈129px)로 12%였다. 그 쌍의 z-순서
	// (fg가 위)는 accent1∩fg(112×102px)가 이미 같은 관계로 판정하므로 얇은 슬리버는 신호를 더하지
	// 않는다 — 채움 대비 경계 비중이 작은 굵은 겹침만 남긴다 (핵심 divergence인 muted∩accent1은
	// 136×156px라 남는다). imageRegion이 잰 rect 그대로 자르되 작은 그림을 픽셀 하한으로 거른
	// 것과 같은 자리의 장치다.
	stackOverlapMinSide = 24.0
)

// intersectRect는 두 사각형의 겹친 사각형이다 (겹치지 않으면 빈 사각형).
func intersectRect(a, b geom.Rect) geom.Rect {
	x0, y0 := math.Max(a.X, b.X), math.Max(a.Y, b.Y)
	x1, y1 := math.Min(a.Right(), b.Right()), math.Min(a.Bottom(), b.Bottom())
	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// 겹친 도형이 두 타깃에서 **같은 순서로** 겹치는가 — 겹친 사각형만 잘라 잰다.
//
// 이 iteration의 핵심 증거다 (TASKS의 리뷰 포인트 "겹침 순서가 웹과 같은지"의 자동 절반).
// z-order 슬라이드는 `at="center"`인 정적 도형과 `x`/`y`로 놓인 도형을 한 stack에 섞었다 —
// 웹은 positioned를 non-positioned 위로 올리고(칠 순서 규칙) PPTX는 문서 순서로 그리는데,
// 항목 04가 그 둘을 **같은 순서**(문서 순서)로 맞췄다. 두 유도가 갈라지면 겹친 자리의 색이
// 두 타깃에서 달라지고, 슬라이드 전체 비율로는 겹친 면적만큼만 튀어 희석되므로(0.54%) 겹친
// 사각형만 잘라 본다. 절대 하한 픽셀 수를 함께 둔다.
func TestStackFixtureOverlapRegions(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, stackFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	const slideID = "z-order"
	index := -1
	for i, slide := range result.Deck.Slides {
		if slide.ID == slideID {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("슬라이드 %s가 없다", slideID)
	}
	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(index))
	loImg := shots[index]

	// 겹침 도형을 문서 순서로 모은다 (z-order 슬라이드의 세 rect — 텍스트는 뺀다).
	var boxes []geom.Box
	for _, box := range result.Geometry.Boxes {
		if box.Slide == index && box.El == soml.ShapeElement {
			boxes = append(boxes, box)
		}
	}
	if len(boxes) < 2 {
		t.Fatalf("겹침 도형이 %d개다 — z-order 슬라이드는 겹친 도형이 여럿이어야 한다", len(boxes))
	}

	worst, worstAt, pairs := 0.0, "", 0
	for a := 0; a < len(boxes); a++ {
		for b := a + 1; b < len(boxes); b++ {
			region := intersectRect(boxes[a].Rect, boxes[b].Rect)
			if region.W == 0 {
				continue
			}
			region = geom.Rect{
				X: region.X + stackOverlapInset, Y: region.Y + stackOverlapInset,
				W: region.W - 2*stackOverlapInset, H: region.H - 2*stackOverlapInset,
			}
			if region.W < stackOverlapMinSide || region.H < stackOverlapMinSide {
				continue
			}
			pairs++
			cmp, err := Diff(cropImage(webImg, region), cropImage(loImg, region), DefaultDiffOptions())
			if err != nil {
				t.Fatalf("%s %s∩%s: %v", slideID, boxes[a].ID, boxes[b].ID, err)
			}
			t.Logf("%s %s∩%s %s: %s", slideID, boxes[a].ID, boxes[b].ID, region, cmp)
			if cmp.Ratio > worst {
				worst, worstAt = cmp.Ratio, fmt.Sprintf("%s∩%s", boxes[a].ID, boxes[b].ID)
			}
			if cmp.Ratio > stackOverlapRegionDiffMaxRatio && cmp.Changed > stackOverlapRegionMinPixels {
				t.Errorf("%s %s∩%s: 겹친 영역 픽셀 오차 %.2f%%가 허용 %.2f%%를 넘는다 — 위아래가 뒤바뀌었다면 "+
					"고칠 곳은 오차가 아니라 항목 04의 칠 순서 결정(render/web)과 문서 순서 조립(render/pptx)이다",
					slideID, boxes[a].ID, boxes[b].ID, cmp.Ratio*100, stackOverlapRegionDiffMaxRatio*100)
			}
		}
	}
	if pairs == 0 {
		t.Fatal("겹친 영역이 하나도 없다 — z-order 슬라이드가 겹치지 않는다 (픽스처가 바뀌었다)")
	}
	t.Logf("가장 큰 겹침 영역 오차: %s %.2f%% (허용 %.2f%%, 겹친 쌍 %d개)",
		worstAt, worst*100, stackOverlapRegionDiffMaxRatio*100, pairs)
}
