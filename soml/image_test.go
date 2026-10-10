package soml

import "testing"

// 이미지의 정상 트리 — 속성 넷(고유·외관·박스·자식 배치)과 두 자리(슬라이드 직속·컨테이너
// 안)가 파싱을 그대로 통과한다. 없는 파일·어휘 밖의 fit·src 없음의 진단은 값 층의 몫이다.
func TestParseImage(t *testing.T) {
	runTreeCases(t, []treeCase{
		{
			name: "고유·외관·박스 속성",
			src: `<slide><image src="assets/wide.png" fit="cover" alt="아키텍처 개요" variant="framed" border="1 solid muted" ` +
				`radius="lg" shadow="soft" opacity="0.8" rotate="45" flip="h" pad="md" w="fill" h="220" ratio="16:9"/></slide>`,
			want: `image[src=assets/wide.png fit=cover alt=아키텍처 개요 variant=framed border=1 solid muted ` +
				`radius=lg shadow=soft opacity=0.8 rotate=45 flip=h pad=md w=fill h=220 ratio=16:9]`,
		},
		{
			// 캡션은 자식이 아니라 형제 <text>다.
			name: "컨테이너 안 그림과 형제 캡션",
			src:  `<slide><col gap="xs"><image src="assets/wide.png" span="2" style="order: 2"/><text role="caption">그림 1</text></col></slide>`,
			want: `col[gap=xs](image[src=assets/wide.png span=2 style=order: 2] text[role=caption]("그림 1"))`,
		},
		{
			name: "값 검사는 파서 밖 — 없는 파일, 어휘 밖 fit, src 없음, 값 없는 src",
			src:  `<slide><image src="assets/없는파일.png" fit="fill"/><image/><image src/></slide>`,
			want: `image[src=assets/없는파일.png fit=fill] image image[src=]`,
		},
	})
}

// 값 없는 속성은 HasValue로 구별된다 — `<image src>`와 `<image src="">`가 모델에서 달라야
// 값 층이 "경로가 없다"와 "경로가 비었다"를 다른 진단으로 낸다.
func TestParseImageAttrHasValue(t *testing.T) {
	doc := parseOK(t, `<slide><image src/><image src=""/></slide>`)
	if attr, ok := elementAt(t, doc.Content, 0).Attr("src"); !ok || attr.HasValue {
		t.Errorf("<image src>의 HasValue = %v, want false", attr.HasValue)
	}
	if attr, ok := elementAt(t, doc.Content, 1).Attr("src"); !ok || !attr.HasValue || attr.Value != "" {
		t.Errorf(`<image src="">의 HasValue = %v Value = %q, want true ""`, attr.HasValue, attr.Value)
	}
}

// void 위반은 **여는 태그 하나**에서만 진단된다 — 안쪽 내용이나 짝 닫는 태그가 진단을
// 부풀리지 않는다.
func TestParseImageErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{
			name: "이미지 안의 내용",
			src:  `<slide><image src="assets/wide.png">캡션<text>x</text></image></slide>`,
			want: []string{`1:9 void tag <image> cannot have content (write <image/>)`},
		},
		{
			// 세 진단이 모두 저자의 `<image>` 이름을 가리킨다 (들여쓰기가 아니라 태그).
			name: "닫히지 않은 이미지 태그",
			src:  "<slide>\n  <image src=\"assets/wide.png\"\n</slide>",
			want: []string{
				"2:4 tag <image> is not closed with `>`",
				"2:4 void tag <image> cannot have content",
				"2:4 tag <image> must be closed before </slide>",
			},
		},
		{
			// 도형 안 그림은 컨테이너를 거친다 — 진단 목록이 그 길을 보여 준다.
			name: "도형 안의 이미지",
			src:  `<slide><shape type="rect"><image src="assets/wide.png"/></shape></slide>`,
			want: []string{"1:28 <shape> cannot contain <image>"},
		},
		{
			// 그림 뒤의 칠이 보이는 자리는 contain의 여백뿐이고 PPTX에서 pic 박스 밖이다.
			name: "이미지에 fill은 없다",
			src:  `<slide><image src="assets/wide.png" fill="surface"/></slide>`,
			want: []string{`1:37 unknown attribute fill on <image> (available attributes: src, fit, alt, variant, border, radius, shadow, opacity, rotate, flip, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)`},
		},
	})
}

// `fit` 값 어휘는 문서의 순서 그대로다 — 목록은 soml에
// 있고 검사는 core다. 기본값은 **언어**의 것이다 (테마를 바꿨을 때 얼굴이 잘리기 시작하면 안 된다).
func TestImageFits(t *testing.T) {
	if DefaultImageFit != FitContain {
		t.Errorf("DefaultImageFit = %q, want %q", DefaultImageFit, FitContain)
	}
	got := ImageFits()
	if len(got) != 2 || got[0] != FitCover || got[1] != FitContain || FitCover != "cover" || FitContain != "contain" {
		t.Errorf("ImageFits() = %v, want [cover contain]", got)
	}
	got[0] = "짓밟기"
	if ImageFits()[0] != FitCover {
		t.Error("ImageFits()가 내부 슬라이스를 그대로 준다")
	}
}
