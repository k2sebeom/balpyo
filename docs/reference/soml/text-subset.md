# 텍스트 서브셋 — `h1`~`h3` · `p` · `ul`/`ol`/`li` · `blockquote` · `pre` · 인라인 태그

HTML의 익숙한 텍스트 태그를 그대로 쓰는 서브셋이다. 블록 태그는
`<text role="…">`의 **축약 표기**이고 (아래 등가 표), 인라인 태그는 블록 안 텍스트 런의
서식이 된다. 어휘는 여전히 닫혀 있다 — 여기 없는 HTML 태그(`h4`, `div`, `img` …)는
무시되지 않고 파스 에러다.

## 태그 표

| 태그 | 종류 | 허용 속성 | 배치 규칙 |
|---|---|---|---|
| `h1` `h2` `h3` `p` `blockquote` | 블록 | `variant` `size` `color` `align` `anchor` + [자식 배치 속성](layout.md) | 슬라이드 또는 레이아웃 컨테이너 직속. 내용은 문자 데이터 + 인라인 태그 |
| `pre` | 블록 | **`lang`** + `variant` `size` `color` `align` `anchor` + 자식 배치 속성 | 슬라이드 또는 레이아웃 컨테이너 직속. 내용은 **문자 데이터만** (공백 원문 보존) |
| `ul` `ol` | 리스트 컨테이너 | `size` `color` `align` `anchor` + 자식 배치 속성 (**`variant`는 없다**) | 슬라이드 또는 레이아웃 컨테이너 직속. 자식은 `<li>`만 |
| `li` | 리스트 항목 | **`reveal`** — 외관은 컨테이너에서 **상속한다** | `<ul>`/`<ol>` 안에서만. 내용은 문자 데이터 + 인라인 태그 |
| `strong` `em` `code` | 인라인 | 없음 | 블록·`<text>`·`<li>` 안. 인라인끼리 중첩 가능 |
| `a` | 인라인 | `href` (필수) | `<a>` 안에 `<a>` 불가 |
| `span` | 인라인 | `color` `weight` | — |
| `br` | void | 없음 | 자기 닫힘 `<br/>`만 — 내용을 가질 수 없다 |

배치 규칙에는 각각 이유가 있다:

- **`<li>`는 `<ul>`/`<ol>` 안에서만**: 마커와 들여쓰기는 리스트 컨테이너 단위의 표현이다 —
  컨테이너 밖의 항목은 마커를 그릴 기준이 없다.
- **중첩 리스트는 아직 불가**: PPTX의 들여쓰기 단계(`lvl`) 매핑 없이 열면 웹에서만
  동작한다 — 한 타깃만 그리는 어휘는 열지 않는다.
- **블록 안에 블록 불가**: v1의 블록은 문단이지 문서 섹션이 아니다. "문단 안의 문단"은
  두 타깃 어디에도 대응하는 구조가 없다.
- **`<a>` 안에 `<a>` 불가**: 겹친 링크는 "무엇을 클릭했는가"가 정의되지 않고, PPTX
  `hlinkClick`은 런 하나에 링크 하나다. 사이에 다른 인라인이 끼어도 잡힌다.
- **`<span>`에 `size` 불가** (`color`·`weight`만): 런 하나가 커지면 웹의 줄 박스 높이가
  변하는데 PPTX 줄간격은 문단 절대값(`spcPts`)이라 두 타깃의 글줄이 갈라진다. 같은
  이유로 `<code>`도 code 역할의 크기(20px)를 끌어오지 않고 부모 크기를 따른다.
- **인라인은 슬라이드 직속이 될 수 없다**: 문단 없는 런은 두 타깃 어디에도 대응하는
  구조가 없다.
- **`<li>`와 인라인 태그에는 자식 배치 속성이 없다**: `<li>`는 리스트 컨테이너의 자식이지
  flex 항목이 아니고(PPTX에서 `<ul>` 하나가 도형 하나다), 인라인은 박스가 아니라 런이다 —
  열어 두면 `<li grow="1">`이 웹에서만 뜻을 갖는다 ([레이아웃 태그](layout.md)).
- **목록의 외관은 컨테이너에 쓰고 항목이 상속한다**: `<ul size="20">`은 그 목록의 항목
  전부를 20px로 만든다. 항목마다 크기가 갈린 목록은 목록이 아니라 문단 여럿이므로 `<li>`에
  외관 속성을 열지 않았다 — 저자는 목록에 한 번만 쓰고, 한 항목을 빠뜨려 어긋난 목록이
  유효한 문서가 되는 일도 없다.
  상속 값은 항목 자신의 캐스케이드에서 **컴포넌트 기본값보다 위, 자기 인라인 속성보다
  아래**에 얹힌다 ([외관 속성](../appearance.md)) — 지금은 `<li>`에 인라인 속성이 없으므로
  컨테이너의 선언이 늘 이긴다.
- **`<li>`가 여는 속성은 `reveal` 하나다**: 외관이 컨테이너의 것이라는 규칙과 부딪히지 않는다 —
  순차 등장은 외관이 아니라 **순서**이고, 항목이 언제 나타나는지는 항목마다 갈리는 성질이라
  컨테이너로 올릴 수 없다 (아래 [순차 등장](#순차-등장-reveal)).
- **컨테이너는 `variant`를 받지 않는다**: variant는 값 하나가 아니라 외관 번들(크기 + 굵기
  + 색)이라, 상속시키면 "목록 글자를 조금 줄인다"가 아니라 "항목들의 역할을 바꾼다"가
  된다 — 그건 등가 표가 소유한 위계(`<li>` ≡ body)를 속성 하나로 뒤집는 일이다.
- **`<ul anchor>`는 지금 뜻을 갖지 않는다**: 세로 앵커가 보이는 자리는 [도형 안](shape.md)
  뿐이고 도형은 아직 리스트를 자식으로 받지 않는다. 그래도 받는 이유는 "텍스트 외관을 받는
  요소는 4종을 받는다"를 예외 없는 한 문장으로 두는 것이다 — 슬라이드 직속 블록의
  `anchor`도 같은 처지다 ([`<text>`](text.md)).

## 역할 등가 표

블록 태그는 텍스트 역할의 축약이다. 등가는 확정 외관의 깊은 비교 수준이다 — 렌더 결과가
갈라지면 등가가 깨진 것이다. 태그 → 역할 대응은 언어(soml)가, 태그별 기본값은 테마가
소유한다.

| 태그 | 등가 표기 | 태그 기본값 (`minimal` 테마) |
|---|---|---|
| `<h1>` | `<text role="title">` | — |
| `<h2>` | `<text role="heading">` | — |
| `<h3>` | `<text role="heading">` | 크기 28 |
| `<p>` | `<text role="body">` | — |
| `<blockquote>` | `<text role="body">` | 색 `muted` |
| `<pre>` | `<text role="code">` | — |
| `<li>` | `<text role="body">` | — (리스트 문단으로 렌더된다) + 컨테이너의 외관 상속 |

- **`h1` → `title`인 이유**: 슬라이드는 문서가 아니라 화면 하나다. 슬라이드에서 가장 큰
  표제는 곧 그 슬라이드의 제목이고, `h1`을 `heading`으로 두면 title 역할에 대응하는 블록
  표기가 없어져 서브셋만 쓰는 저자가 표지를 만들 수 없다.
- **태그 기본값은 캐스케이드의 컴포넌트 기본값 단**에 얹힌다 — 인라인 속성과 `variant`가
  덮어쓸 수 있다 ([외관 속성의 캐스케이드](../appearance.md)). `h3`의 28은 heading(36)과
  body(24) 사이의 반 단계로, heading의 줄높이 1.3을 그대로 받아 줄박스(≈36.4px)가 본문
  한 줄(36px)의 수직 리듬에 얹힌다. `blockquote`는 색만 muted다 — 왼쪽 괘선 같은 시각
  장식은 [도형](shape.md) 외관의 몫이다.
- **비대칭 — `caption`에는 대응 블록 태그가 없다**: `figcaption`은 어휘 밖이고
  `<text role="caption">`이 그 자리를 맡는다.
- 블록은 `role` 속성을 받지 않는다 — `<p role="title">`은 두 표기의 등가를 깬다. 역할은
  등가 표에서만 온다.

## 공백 규칙

- **요소 양 끝의 공백은 잘리고, 형제 사이의 공백은 내용이다.** `<strong>배포</strong>
  자동화`의 태그 사이 한 칸은 산출물에 남는다. 반대로 `<strong>배포 </strong>자동화`처럼
  태그 **안쪽** 끝에 쓴 공백은 잘린다 — 공백은 태그 밖에 쓴다.
- 원문 개행·들여쓰기는 파서가 접는다 (ASCII 공백 런 → 한 칸). 웹과 PPTX가 **같은
  문자열**을 받으므로 줄바꿈이 원문 서식에 의존하지 않는다.
- **`<pre>` 안은 원문 그대로다** — 개행·연속 공백·탭이 전부 내용이다. 여는 태그 직후와
  닫는 태그 직전의 개행 **하나씩만** 벗긴다 (HTML 관례 — 저자가 `<pre>` 다음 줄부터 쓰게
  하기 위한 것). 그래서 빈 줄로 시작하는 코드도 표현할 수 있다.
- **탭만은 예외다** — 값 층이 칼럼을 세어 스페이스로 편다 (아래 코드 블록 절).

## 코드 블록 — `lang`과 탭

`<pre>`는 서브셋에서 `lang`을 받는 유일한 태그다.

```html
<pre lang="go">// 배포 때 캐시를 한 번에 채운다 — TTL이 전부 같은 시각이 된다.
func warm(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := cache.Set(ctx, key(id), load(id), 30*time.Minute); err != nil {
			return err
		}
	}
	return nil
}</pre>
```

들여쓰기가 **탭 그대로**인 것이 이 예제의 요점이다 — gofmt·prettier 출력을 손으로 고치지 않고
붙여 넣는다.

### `lang` 값은 닫힌 목록이다

| 값 | 무엇 |
|---|---|
| `go` | Go |
| `xml` | XML·HTML 계열 |
| `soml` | **`xml`의 별칭** — 이 저장소의 덱이 자기 문법을 띄우는 것이 가장 흔한 코드 블록이다 |
| `json` | JSON (`balpyo.json`·`theme.json`) |
| `bash` | 셸 (`balpyo build …`) |

- **목록 밖 이름은 에러다** — `variant`처럼 경고 + 폴백하지 않는다. `variant` 어휘는 테마의
  사유물이라 다른 테마로 만든 덱도 열려야 하지만 `lang`은 **엔진 어휘**이므로, 목록 밖 이름은
  오타이거나 아직 없는 언어다. 조용히 하이라이팅을 끄면 저자는 자기 색이 왜 안 나오는지 알
  길이 없다.
- **`lang`은 생략할 수 있다.** 없으면 하이라이팅만 없고 나머지는 같다 (코드 역할의 mono 폰트,
  공백 보존, 탭 펴기).
- `soml`을 언어로 세지 않는 것이 값이다 — SOML 전용 토크나이저를 들이면 어휘가 자랄 때마다
  렉서를 함께 고쳐야 한다.

### 강조는 저자가 아니라 토크나이저가 만든다

`lang`이 붙으면 값 층이 `<pre>`의 문자 데이터를 토큰 스트림으로 읽어 **색이 하나인 조각마다
합성 인라인 런**을 만든다. 저자가 `<pre>` 안에 `<span>`을 쓰는 길은 열리지 않는다 (콘텐츠
모델이 문자 데이터뿐이다 — `<pre><code>` 관용도 없다): 런의 주인이 둘이면 "저자가 칠한 조각"과
"토크나이저가 칠한 조각"이 같은 줄에서 겹치는 문제가 생기고, 그 문제는 열지 않는 쪽으로
풀었다.

- **색은 테마가 정한다.** 종류(`keyword`·`name`·`attr`·`string`·`number`·`comment`·`punct`)
  → 색 토큰 표가 `components.json`의 `code` 절에 있고, 그 값은
  [외관 속성](../appearance.md)의 코드 하이라이팅 절에 있다. `<pre>`에 색을 고르는 속성은
  없다 — 코드마다 색을 고르면 덱이 팔레트를 잃는다.
- **합성 런도 토큰 참조로 남는다.** 색 이름의 해석이 저자가 쓴 `<span color>`와 **같은 함수**를
  지나므로, PPTX 슬라이드 파트의 코드 런은 `schemeClr` 참조다 (PowerPoint의 테마 색 편집이
  코드 블록만 두고 가지 않는다).
- **색이 없는 조각은 요소가 되지 않는다** — 공백·연산자·평범한 식별자는 `<pre>`의 확정 색을
  그대로 물려받는다. 런은 **색이 바뀌는 자리마다** 끊긴다 (종류가 아니라 색이 경계다): 테마가
  적지 않은 종류는 색이 없으므로 끊을 이유가 없고, 종류마다 끊으면 Go 다섯 줄이 `a:r` 서른
  개가 된다.

### 탭은 `lang`과 무관하게 펴진다

탭은 하이라이팅의 문제가 아니라 **글자의 문제**다. CSS `tab-size`와 OOXML의 절대 탭
정지점(`a:defTabSz`)은 같은 규칙이 아니어서, 탭을 그대로 흘리면 두 산출물의 줄이 갈라진다.
그래서 값 층이 모든 `<pre>`의 탭을 스페이스로 펴고, 두 렌더러는 **같은 한 문자열**을 받는다.

- 정지점은 "다음 배수까지"지 "N칸"이 아니다 — 글자 하나 뒤의 탭은 한 칸만 먹는다. 그 규칙이
  gofmt 출력의 들여쓰기 계단을 고르게 만든다.
- 폭은 테마가 정한다 (`components.json`의 `code.tabSize` — `minimal`은 **2**다. 슬라이드는
  폭이 좁아 4칸이면 세 번째 중첩이 화면을 벗어난다).
- 칼럼은 **글자 수**로 센다 — 동아시아 문자의 폭 2를 반영하지 않는다. 폭을 재는 규칙을 값 층에
  두면 같은 계단이 폰트마다 달라지고, 두 렌더러가 받는 글자는 어차피 하나라 산출물이 갈라질
  자리는 없다.
- 탭을 쓰지 않은 덱의 산출물은 이 패스 때문에 **한 바이트도 움직이지 않는다**.

## 문자 참조

`<`와 `&`는 마크업의 글자이므로 내용으로 쓸 길이 필요하다. SOML의 문자 참조는 **닫힌
어휘**이고 `;`를 요구한다:

| 표기 | 문자 |
|---|---|
| `&amp;` | `&` |
| `&apos;` | `'` |
| `&gt;` | `>` |
| `&lt;` | `<` |
| `&quot;` | `"` |
| `&#10진수;` (`&#8594;`) | 코드포인트 |
| `&#x16진수;` (`&#x27E8;`, `&#X27E8;`) | 코드포인트 |

- **참조로 읽는 자리는 `&이름;`·`&#수;`·`&#x수;` 모양일 때뿐이다.** 그 모양이 아니면 `&`는
  그냥 글자다 — `R&D 예산`, `?a=1&b=2`, `100% & 그 외`를 참조로 쓰지 않아도 된다. HTML5는
  `;` 없는 `&amp`도 받지만 SOML은 받지 않는다: 받으면 `R&ampere`가 조용히 `R&ere`가 된다.
- **모양은 맞고 어휘 밖이면 에러다.** `&nbsp;`를 리터럴로 흘리면 화면에 `&nbsp;`가 그대로
  뜨는 것으로만 드러난다 — 그건 빌드가 잡을 수 있는 실패다. 코드포인트도 같다: XML이
  금지한 문자(`&#0;`), 서로게이트(`&#xD800;`), 범위를 넘는 값(`&#x110000;`)은 에러다
  (PowerPoint가 파일 열기를 거부하는 XML이 나가지 않게).
- **마크업이 아닌 `<`는 내용이다.** `<`가 태그를 여는 것은 **영문자**·`/`·`<!--`·`<![CDATA[`
  앞일 때뿐이므로 `a < b`·`x <= y`·`1<2`·`a <- b`를 그냥 쓴다. `&lt;`로 써도 결과는 같다.
  이 규칙이 오타를 놓치지 않는 근거는 태그 이름이 항상 영문자로 시작한다는 것이다 —
  `<txt>`는 여전히 `unknown tag <txt>`로 잡힌다.
- **접기 다음에 참조를 푼다.** `&#32;`로 쓴 공백은 접히지 않으므로 `가&#32;&#32;나`는 두
  칸을 남긴다 — 접는 모델에서 공백을 의도대로 넣는 유일한 길이다.
- **참조는 한 번만 풀린다.** `&amp;lt;`는 `&lt;`라는 **글자 다섯**이고, 산출물에서 웹은
  `&amp;lt;`로, PPTX는 `&amp;lt;`로 다시 감싸진다 — `&amp;amp;lt;`가 되지 않는다.
- **속성 값도 같은 규칙이다.** `alt="a &lt; b"`가 동작하고 `href="…?a=1&b=2"`는 그대로다.
- **`<pre>`·`<code>`도 같은 규칙이다** — 참조는 공백 보존과 무관하게 풀린다. 코드에
  `&`가 흔하다는 이유로 예외를 두면 "여기서는 참조가 되고 저기서는 안 된다"를 저자가
  외워야 한다. 원문을 **하나도** 건드리지 않고 싣고 싶으면 CDATA를 쓴다.

## CDATA

`<![CDATA[ … ]]>` 안의 내용은 마크업도 참조도 아닌 **글자 그대로**다. 문자 데이터를 쓸 수
있는 자리라면 어디든 쓸 수 있다.

```html
<slide id="syntax">
  <h2>SOML 문법</h2>
  <pre><![CDATA[<slide id="title">
  <text role="title">2026 아키텍처 리뷰</text>
</slide>]]></pre>
  <p>비교 연산자: <code><![CDATA[a < b && c]]></code></p>
</slide>
```

- **공백 접기는 부모의 내용 모델을 따른다** — `<pre>` 안에서는 개행과 들여쓰기가 보존되고,
  `<p>` 안에서는 접힌다. CDATA는 "참조를 끄는 표기"이지 "공백 규칙을 바꾸는 표기"가
  아니다.
- CDATA와 일반 텍스트는 이어 붙는다 — `식: <![CDATA[a < b]]> 끝`은 한 문단이다.
- `]]>`로 닫지 않으면 에러다 (``CDATA section is not closed with `]]>` ``). 닫히지 않은 CDATA는
  파일의 나머지를 삼키므로 열리지 않은 태그 진단이 함께 난다 — 닫히지 않은 주석과 같다.
- **CDATA 안에 `]]>`는 담을 수 없다** (XML과 같다) — 첫 `]]>`가 섹션을 닫는다. 그 글자가
  내용이어야 하면 섹션을 둘로 쪼개거나(`]]]]><![CDATA[>`) `&gt;`를 쓴다.

## `<br/>`와 `<a href>`

- `<br/>`는 같은 블록 안의 줄바꿈이다 — 문단을 쪼개지 않는다 (문단 사이 간격은 줄간격과
  다른 축이다). 서식도 외관도 갖지 않는 흐름의 분절점이고, 자기 닫힘으로만 쓴다.
- `<a href="…">`의 href는 열린 값이다 — URL 문법을 검증하지 않는다 (파일·앵커·상대 경로
  전부 유효할 수 있고, 잘못된 URL은 클릭이 실패할 뿐 빌드가 막을 수 있는 실패가 아니다).
  href가 없거나 비어 있으면 에러다. `?a=1&b=2`의 `&`는 참조 모양이 아니므로 글자 그대로
  실린다 — 쿼리 문자열을 위해 `&amp;`로 바꿔 쓸 필요가 없다 (위 문자 참조 절).
- 링크 런은 밑줄과 `accent1` 토큰 색을 받는다 — 웹과 PPTX의 링크 색이 같은 토큰 표에서
  나온다.

## 순차 등장 — `reveal`

`<li reveal>`은 그 항목을 **발표자 뷰에서 한 스텝씩** 나타나게 한다. 값을 생략하면
그 슬라이드에서 자기 차례를 받고, `reveal="2"`는 그 자리를 덮어쓴다.

```html
<slide id="changes">
  <h2>올해의 세 가지 변화</h2>
  <ul>
    <li>모놀리스 분리 완료</li>            <!-- 처음부터 보인다 -->
    <li reveal>배포 파이프라인 자동화</li>  <!-- 1번째 → -->
    <li reveal>관측성 스택 도입</li>        <!-- 2번째 → -->
  </ul>
  <ol>
    <li reveal="3">먼저 잰다</li>          <!-- 목록을 넘어 이어진 3번째 -->
    <li reveal="3">그 다음 고친다</li>      <!-- 같은 번호는 함께 나타난다 -->
  </ol>
</slide>
```

규칙은 다섯이다:

- **번호는 슬라이드 안에서 1부터다.** 값 없는 `reveal`은 그 슬라이드에서 `reveal`을 가진 항목
  중 **자기 차례**(문서 순서 1, 2, 3…)를 받고, 차례는 `<ul>`에서 `<ol>`로 넘어가도 이어진다.
  다음 슬라이드의 첫 스텝은 다시 1이다 — 발표는 슬라이드 단위로 진행하므로 덱을 관통하는
  번호는 쓸 자리가 없다.
- **같은 번호를 받은 항목들은 함께 나타난다.** 한 스텝에 여러 줄을 띄우는 유일한 길이다.
- **값이 있으면 1 이상의 정수다.** `reveal=""`·`0`·`-1`·`1.5`·낱말은 진단이고, 그 항목은
  순서를 잃는다 (전개된 상태로 남는다). 스텝 0은 "처음부터 보인다"는 뜻이 되어 `reveal`을
  쓰지 않은 것과 구별되지 않는다.
- **숨는 곳은 발표자 뷰 하나다.** 문서 뷰(파일을 열어 JS가 꺼진 상태)·오버뷰·인쇄(PDF)·PPTX는
  **전부 전개된 상태**를 본다. 그래서 발표 중에 Cmd+P를 눌러도 아직 말하지 않은 항목이 종이에서
  빠지지 않고, PPTX 산출물은 `reveal`을 쓰기 전과 한 바이트도 다르지 않다 (PPTX 진입
  애니메이션으로 옮기지 않는다).
- **나타나도 배치가 움직이지 않는다.** 숨김은 자리를 남기는 방식이므로(`visibility`) 저자가 짠
  한 배치가 발표 중에 여러 배치가 되지 않는다 — 웹과 PPTX가 같은 좌표를 쓰는 근거다.

조작은 [발표 조작](../cli.md#웹-산출물에서의-발표-조작)과 같다: **다음**(→·Space·›)은 스텝을
먼저 먹고 남은 스텝이 없을 때 다음 슬라이드로 가고, **이전**(←·‹)은 스텝을 하나 되감는다.
뒤로 들어온 슬라이드는 전부 전개된 상태이고, 딥링크·오버뷰 타일·Home/End로 곧장 들어간
슬라이드는 스텝 0에서 시작한다.

## 예제

`testdata/fixtures/02_text_subset/slides/001_headings.slide` — 제목 위계:

```html
<slide id="headings">
  <h1>2026 아키텍처 리뷰</h1>
  <h2>상반기 회고</h2>
  <h3>플랫폼 팀</h3>
  <p>모놀리스 분리를 마쳤고, 배포 파이프라인을 자동화했으며,
  관측성 스택을 도입했다.</p>
  <blockquote>측정하지 않으면 개선할 수 없다.</blockquote>
</slide>
```

`002_lists.slide` — 리스트와 인라인 서식:

```html
<slide id="lists">
  <h2>올해의 세 가지 변화</h2>
  <ul>
    <li>모놀리스 분리 완료</li>
    <li reveal>배포 파이프라인 <strong>자동화</strong></li>
    <li reveal>관측성 스택 도입 (<code>otel</code> + Grafana)</li>
  </ul>
  <ol>
    <li>먼저 잰다</li>
    <li reveal="3">그 다음 고친다</li>
  </ol>
</slide>
```

`003_inline.slide` — 링크·줄바꿈·span:

```html
<slide id="inline">
  <h2>인라인 서식</h2>
  <p>배포 파이프라인 <strong>자동화</strong>와 <em>관측성</em>이 핵심이다.</p>
  <p>빌드는 <code>make test</code>로 돌리고, 문서는
  <a href="https://example.com/docs">레퍼런스</a>를 본다.</p>
  <p>첫 줄<br/>둘째 줄</p>
  <p><span color="accent1" weight="bold">강조된 런</span>과 보통 런이 한 문단에 있다.</p>
</slide>
```

`004_pre.slide` — 공백 보존:

```html
<slide id="pre">
  <h2>코드 블록</h2>
  <pre>
func main() {
	fmt.Println("balpyo")  // 연속  공백도 내용이다
}
</pre>
</slide>
```

`lang` 어휘는 `go`·`soml`·`json`·`bash`이고, 어느 언어든 웹의 `<span>`과 PPTX의 `a:r`는 같은
경계에서 끊긴다.

## 렌더 결과

**웹** — 블록과 `<li>`는 저자가 쓴 이름 그대로 **시맨틱 태그**로 나간다 (`<h2>`→`<h2>`,
`<p>`→`<p>` …, 클래스는 전부 `balpyo-text`). 스크린 리더가 제목 위계와 리스트를 읽을 수
있되, 텍스트 메트릭은 확정 스타일의 인라인 `style`이 전부 명시하고 margin·padding은 엔진
CSS가 0으로 명시하므로 **브라우저 기본 스타일에 기대는 값은 없다** — 메트릭은 어느 태그를
골라도 같다. `<text>`는 대응 시맨틱 태그가 없는 중립 요소라 `<div>`로 남는다.

- 인라인 태그는 태그 그대로 나가되 확정 런 스타일을 **완전 명시**한다 — 기본 6선언에
  `font-style`(em)·`text-decoration`(a)까지 8선언. UA 시트의 `em { font-style: italic }`
  같은 규칙이 개입할 자리가 없다. 인라인에는 지오메트리 식별자(`data-balpyo-id`)가 붙지
  않는다 — 측정 대상은 블록 종류뿐이다.
- 리스트는 `<ul>`/`<ol>`(클래스 `balpyo-list`) + `<li>`다. 마커는 UA의 `::marker`가
  아니라 엔진이 그린다: ul은 `li::before`의 글리프 `•`, ol은 엔진 카운터의 `1.` 형식 —
  마커가 li의 확정 폰트·색을 상속한다. 들여쓰기 값은 [외관 속성](../appearance.md)의
  리스트 매핑 표에 있다.
- **컨테이너에 쓴 외관은 `<ul>` 태그가 아니라 `<li>`의 인라인 `style`로 나간다** — 확정 값을
  갖는 것은 항목이고 컨테이너는 상속의 출처일 뿐이다. PPTX도 같다: 문단
  (`a:pPr@algn`)과 런(`a:rPr@sz`)이 항목마다 값을 싣고, 도형은 하나 그대로다.
- `<pre>`는 `white-space: pre`로 줄바꿈하지 않는다 — 긴 줄은 오버플로 허용이다 (박스
  고정 정책).
- **하이라이팅의 합성 런은 저자가 쓴 인라인과 같은 경로로 나간다** — 웹은 `<span style>`,
  PPTX는 `a:r` + `a:rPr`다. 렌더러에 코드 전용 경로는 없다: 값 층이 색이 확정된 조각을 만들어
  이미 있던 런 경로에 넣는다. 합성 런에는 지오메트리 식별자가 붙지 않으므로(인라인은 측정
  대상이 아니다) 잰 박스는 하이라이팅이 없을 때와 **같다** — 줄 수도 같다 (측정은 식별자를
  갖지 않는 자손의 글리프까지 읽는다).

**PPTX** — 블록 하나가 `p:sp` 텍스트 박스 하나다 (좌표·외관 매핑은 [`<text>`](text.md)와
같다). 인라인 서식은 **텍스트 런 분할**이 된다: 문자 데이터 잎마다 `a:r` 하나, 스타일은
잎을 감싼 가장 안쪽 요소의 확정 스타일이다.

- 인라인 런의 `a:rPr`는 `b`(굵기)·`i`(이탤릭)·`u`(밑줄)·색·폰트를 전부 명시한다 — 뷰어의
  자동 밑줄·자동 색에 기대지 않는다.
- `<a>`는 런의 `a:hlinkClick r:id`와 슬라이드 rels의 외부(`TargetMode="External"`) 관계가
  된다. 같은 URL은 관계 하나를 재사용한다.
- `<br/>`와 `<pre>`의 보존 개행은 `a:br`이다 — 문단을 쪼개지 않는다. `<pre>`의 텍스트
  박스는 `bodyPr@wrap="none"`으로 웹의 `white-space: pre`와 짝을 맞춘다.
- `<ul>`/`<ol>` 하나가 텍스트 박스 **하나**이고 `<li>` 하나가 문단(`a:p`) 하나다 —
  항목마다 도형을 내면 PowerPoint에서 리스트가 낱개 텍스트 박스로 조각나 편집 단위가
  깨진다. 문단은 행잉 인덴트(`marL` + 음수 `indent`)와 마커(ul `buChar`, ol
  `buAutoNum type="arabicPeriod"`), li의 확정 폰트를 딴 `buFont`를 받는다 — 값은 웹
  리스트 CSS와 같은 상수에서 나온다 ([외관 속성](../appearance.md)의 리스트 매핑 표).

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 배치·문법:

| 조건 | 진단 메시지 |
|---|---|
| 서브셋 밖의 HTML 태그 (`<h6>` 등) | `unknown tag <h6> — did you mean <h3>? (tags available inside <slide>: …)` — 목록은 그 자리(부모)에 쓸 수 있는 태그다. `<b>`·`<i>`·`<img>`·`<h4>`–`<h6>` 같은 HTML 습관과 철자가 한두 글자 다른 이름은 후보를 하나 말한다 |
| `<li>`가 리스트 밖에 있다 | `<li> can only be a child of <ul>/<ol>` |
| 중첩 리스트 | `<ul> cannot be nested inside <li> — nested lists are not supported yet` |
| `<ul>`/`<ol>` 직속 문자 데이터 | `character data cannot appear directly inside <ul> (wrap it in <li>)` |
| `<ul>`/`<ol>`의 자식이 `<li>`가 아니다 | `<ul> cannot contain <p> (available tags: <li>)` |
| 블록 안의 블록·리스트 | `block tag <h2> is not allowed inside <p> (blocks do not nest)` |
| `<a>` 안의 `<a>` (중첩 포함) | `<a> cannot be nested inside <a>` |
| `<br>`을 여닫아 썼다 | `void tag <br> cannot have content (write <br/>)` |
| `<pre>` 안에 요소를 썼다 | `only character data is allowed inside <pre> (<strong> is not allowed)` |
| `<pre>` 아닌 블록에 `lang`을 썼다 | `unknown attribute lang on <p> (available attributes: variant, size, …)` — `lang`은 `<pre>` 고유 속성이다 |
| 어휘 밖의 문자 참조 (`&nbsp;`) | `` unsupported character reference &nbsp; (available: &amp;, &apos;, &gt;, &lt;, &quot;, `&#decimal;`, `&#xhex;`) `` |
| 쓸 수 없는 코드포인트 (`&#0;` `&#xD800;` `&#x110000;`) | 같은 진단 — 참조 모양이지만 해석할 수 없다 |
| CDATA가 닫히지 않았다 | ``CDATA section is not closed with `]]>` `` |
| 요소만 담는 자리에 CDATA를 썼다 | `character data cannot appear directly inside <slide> (wrap it in <text>)` — 표기가 달라도 내용이다 |
| 블록에 `role`을 썼다 | `unknown attribute role on <h1> (available attributes: variant, size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot)` |
| `<span size>` | `unknown attribute size on <span> (available attributes: color, weight)` |
| `<li>`에 외관 속성을 썼다 | `unknown attribute size on <li> (available attributes: reveal)` — 외관은 `<ul>`/`<ol>`에 쓴다 |
| `<ul variant>` | `unknown attribute variant on <ul> (available attributes: size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot)` |

외관 해석(`core.Resolve`) — 값 어휘는 이 층이 소유한다. 블록의 `size`/`color`/`align`
진단은 [`<text>`](text.md)와 같고, 서브셋 고유의 것은:

| 조건 | 진단 메시지 |
|---|---|
| 어휘 밖의 `lang` | `unknown lang "typescript" (available values: go, xml, soml, json, bash)` |
| `lang`에 값이 없다 (`<pre lang>`) | `the lang attribute needs a value (lang="go")` |
| `lang=""` | `the lang attribute is empty (remove it to use the default)` — 하이라이팅 없는 `<pre>`가 기본값이다 |
| `<a>`에 `href`가 없다 | `<a> needs an href — a link without a destination has no meaning` |
| `href`에 값이 없다 (`<a href>`) | `the href attribute needs a value (href="https://example.com")` |
| `href=""` | `href is empty — a link without a destination has no meaning` |
| `reveal=""` | `the reveal attribute is empty (remove the value to take the next step in order)` |
| `reveal`이 1 이상 정수가 아니다 (`0`·`-1`·`1.5`·낱말) | `reveal is an integer of at least 1 ("0")` — 그 항목만 순서를 잃고 나머지 항목의 번호는 그대로다 |
| `weight`에 400/700 수치를 썼다 | `weight takes a name — instead of "700", use "bold"` |
| `weight`에 그 외 수치를 썼다 | `weight "600" is not allowed — there are two weights, normal (400) and bold (700) (OOXML bold is on/off, so intermediate weights would only work on the web)` |
| 어휘 밖의 `weight` | `unknown weight "…" (available values: normal, bold)` |

모르는 `variant`는 에러가 아니라 **경고 + 기본 외관 폴백**이다 — 규칙은
[외관 속성](../appearance.md)의 캐스케이드 절에 있다.
