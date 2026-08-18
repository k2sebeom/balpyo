package core

import (
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 우선순위 표: 같은 필드를 여러 단이 정할 때 인라인 > variant > 컴포넌트 기본값 >
// 역할 기본값 순으로 이긴다. 단마다 하나씩 빼 가며 단정한다 — 병합이 mergeText 한 곳에서만
// 일어나므로 여기서 순서를 못 박으면 캐스케이드 전체의 우선순위가 못 박힌다.
func TestMergeTextCascadeOrder(t *testing.T) {
	component := textPatch{size: ptr(30.0)}
	variant := textPatch{size: ptr(40.0)}
	inline := textPatch{size: ptr(50.0)}
	empty := textPatch{}

	cases := []struct {
		name                       string
		component, variant, inline textPatch
		want                       float64
	}{
		{"인라인이 전부를 이긴다", component, variant, inline, 50},
		{"인라인이 빠지면 variant", component, variant, empty, 40},
		{"variant까지 빠지면 컴포넌트 기본값", component, empty, empty, 30},
		{"전부 빠지면 역할 기본값", empty, empty, empty, 24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := textRoleDefaults(soml.RoleBody) // size 24
			mergeText(&style, tc.component)
			mergeText(&style, tc.variant)
			mergeText(&style, tc.inline)
			if style.Size != tc.want {
				t.Errorf("Size = %v, want %v", style.Size, tc.want)
			}
		})
	}
}

// 부분성: 패치는 정한 필드만 얹고, "정하지 않음"(nil)은 아래 단의 값을 지우지 않는다.
func TestMergeTextAppliesOnlySetFields(t *testing.T) {
	base := textRoleDefaults(soml.RoleBody)

	// 빈 패치는 아무것도 바꾸지 않는다.
	got := base
	mergeText(&got, textPatch{})
	if got != base {
		t.Errorf("빈 패치가 외관을 바꿨다: %+v → %+v", base, got)
	}

	// 필드 전부를 정한 패치는 전부를 바꾼다 — 병합이 다루는 필드 목록의 회귀 감시다.
	// TextStyle에 필드가 늘었는데 textPatch가 따라가지 않으면 이 단정이 잡는다.
	full := textPatch{
		font:       ptr(soml.FontMono),
		size:       ptr(99.0),
		weight:     ptr(700),
		lineHeight: ptr(2.0),
		color:      ptr(soml.LiteralColor("ABCDEF")),
		align:      ptr(soml.AlignJustify),
	}
	got = base
	mergeText(&got, full)
	want := soml.TextStyle{Role: soml.RoleBody, Font: soml.FontMono, Size: 99, Weight: 700,
		LineHeight: 2.0, Color: soml.LiteralColor("ABCDEF"), Align: soml.AlignJustify}
	if got != want {
		t.Errorf("전체 패치 = %+v, want %+v", got, want)
	}

	// 일부만 정한 패치는 나머지를 아래 단에 남긴다.
	got = base
	mergeText(&got, textPatch{size: ptr(32.0)})
	if got.Size != 32 {
		t.Errorf("Size = %v, want 32", got.Size)
	}
	if got.Color != base.Color || got.Align != base.Align || got.Font != base.Font ||
		got.Weight != base.Weight || got.LineHeight != base.LineHeight {
		t.Errorf("정하지 않은 필드가 아래 단을 잃었다: %+v", got)
	}
}

// "정하지 않음"과 0값의 구별: 0값을 **명시적으로** 정한 패치는 0값을 얹는다.
// 이 구별이 없으면 (완성 스타일끼리 덮어쓰면) "weight를 0으로" 같은 지정과 "weight를
// 말하지 않음"이 같은 코드가 되어 어느 단이 이겼는지 알 수 없다.
func TestMergeTextDistinguishesUnsetFromZero(t *testing.T) {
	style := textRoleDefaults(soml.RoleBody)
	mergeText(&style, textPatch{weight: ptr(0), size: ptr(0.0)})
	if style.Weight != 0 || style.Size != 0 {
		t.Errorf("명시된 0값이 적용되지 않았다: %+v", style)
	}
}

// minimal 테마의 variant 표: 이름은 결정적 순서로 나열되고 (경고 메시지의 근거),
// 각 variant는 역할 기본값과 **구별되는** 값만 정한다 — 같은 값을 정하는 variant는
// 우선순위 테스트가 아무것도 증명하지 못한다.
func TestThemeTextVariantsAreDistinctFromRoleDefaults(t *testing.T) {
	variants := textVariants()
	if len(variants) < 2 {
		t.Fatalf("variant %d개 — 캐스케이드를 증명하려면 2개 이상이 필요하다", len(variants))
	}
	seen := map[string]bool{}
	for _, v := range variants {
		if seen[v.name] {
			t.Errorf("variant 이름이 중복됐다: %q", v.name)
		}
		seen[v.name] = true
		if v.patch == (textPatch{}) {
			t.Errorf("variant %q가 아무 필드도 정하지 않는다", v.name)
		}
		// 어느 역할 기본값과도 겹치지 않아야 하는 것은 아니다 — 기본 역할(body) 기준으로
		// variant가 정한 필드가 실제로 값을 바꾸는지만 본다.
		base := textRoleDefaults(soml.DefaultTextRole)
		merged := base
		mergeText(&merged, v.patch)
		if merged == base {
			t.Errorf("variant %q가 기본 역할의 외관을 하나도 바꾸지 않는다", v.name)
		}
	}
}

func TestTextVariantLookup(t *testing.T) {
	for _, v := range textVariants() {
		patch, ok := textVariantPatch(v.name)
		if !ok {
			t.Errorf("textVariantPatch(%q)를 찾지 못했다", v.name)
			continue
		}
		// 패치는 포인터 필드라 값 비교가 안 된다 — 같은 바닥에 얹은 결과로 비교한다.
		wantStyle, gotStyle := textRoleDefaults(soml.DefaultTextRole), textRoleDefaults(soml.DefaultTextRole)
		mergeText(&wantStyle, v.patch)
		mergeText(&gotStyle, patch)
		if gotStyle != wantStyle {
			t.Errorf("textVariantPatch(%q) = %+v, want %+v", v.name, gotStyle, wantStyle)
		}
	}
	if _, ok := textVariantPatch("없는-이름"); ok {
		t.Error("없는 variant가 찾아졌다")
	}
}
