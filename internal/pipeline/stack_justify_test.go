package pipeline_test

import (
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
)

// `<stack justify>`는 단일 셀 안에서 자식을 가로로 정렬할 뿐 셀 자체를 줄이지 않는다.
// 셀(암시 트랙)이 내용 폭으로 줄어들면 `w="fill"` 자식이 그 좁은 칸의 100%가 되어 쪼그라들고,
// 텍스트가 없으면 폭 0으로 사라진다. 그래서 어떤 justify 값에서도 채우는 자식은 stack 폭
// 전체를 받는다 — 판정은 선언한 px로 하고 글꼴 줄바꿈에 기대지 않는다.
func TestStackJustifyKeepsFillChildrenFull(t *testing.T) {
	testkit.RequireChromium(t)
	for _, justify := range []string{"start", "center", "end", "between", "around", "evenly"} {
		t.Run(justify, func(t *testing.T) {
			dir := project(t, map[string]string{
				"balpyo.json": `{"title": "stack justify"}`,
				"slides/001_with_text.slide": `<slide id="with-text">
  <stack w="400" h="200" justify="` + justify + `">
    <shape type="rect" w="fill" h="fill" fill="accent1"></shape>
    <text>label</text>
  </stack>
</slide>`,
				"slides/002_alone.slide": `<slide id="alone">
  <stack w="400" h="200" justify="` + justify + `">
    <shape type="rect" w="fill" h="fill" fill="accent1"></shape>
  </stack>
</slide>`,
			})
			result, err := pipeline.Build(pipeline.Options{
				ProjectDir:  dir,
				Targets:     []render.Target{render.TargetHTML},
				CheckLayout: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			// 정렬은 여전히 먹는다: 텍스트 자식이 justify가 가리키는 쪽에 붙는다. 셀이 하나라
			// 분배 값은 한 칸일 때의 CSS 대체값(between은 start, around·evenly는 center)을 따른다.
			stack, _ := result.Geometry.Box("s0.0")
			label, ok := result.Geometry.Box("s0.0.1")
			if !ok {
				t.Fatal("s0.0.1 박스가 측정되지 않았다")
			}
			left := label.Rect.X - stack.Rect.X
			right := stack.Rect.X + stack.Rect.W - (label.Rect.X + label.Rect.W)
			var off float64
			switch justify {
			case "start", "between":
				off = left
			case "end":
				off = right
			default:
				off = left - right
			}
			if off < -0.5 || off > 0.5 {
				t.Errorf("텍스트 자식이 justify=%s 자리에 있지 않다: 왼쪽 여백 %v, 오른쪽 여백 %v", justify, left, right)
			}
			for _, id := range []string{"s0.0", "s1.0"} {
				stack, ok := result.Geometry.Box(id)
				if !ok {
					t.Fatalf("%s 박스가 측정되지 않았다", id)
				}
				fill, ok := result.Geometry.Box(id + ".0")
				if !ok {
					t.Fatalf("%s.0 박스가 측정되지 않았다", id)
				}
				if fill.Rect.X != stack.Rect.X || fill.Rect.W != stack.Rect.W {
					t.Errorf("%s: w=\"fill\" 자식 x=%v w=%v, want stack의 x=%v w=%v",
						id, fill.Rect.X, fill.Rect.W, stack.Rect.X, stack.Rect.W)
				}
			}
		})
	}
}
