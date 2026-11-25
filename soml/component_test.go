package soml

import (
	"strings"
	"testing"
)

const stepDefine = `<define name="step" params="icon label note?">
  <col gap="sm">
    <stack w="84" h="84">
      <shape type="rect" variant="icon-tile"/>
      <image src="{icon}" w="48" alt="{label}"/>
    </stack>
    <text variant="label">{label}</text>
    <text>{note}<slot/></text>
  </col>
</define>
`

const stepsDefine = `<define name="steps">
  <row gap="lg"><slot/></row>
</define>
`

const markerDefine = `<define name="marker" params="year">
  <shape type="ellipse" w="24" h="24"><text>{year}</text></shape>
</define>
`

// mustComponent는 진단이 없어야 하는 정의를 파싱한다.
func mustComponent(t *testing.T, file, src string) *Component {
	t.Helper()
	def, diags := ParseComponent(file, src)
	if len(diags) != 0 {
		t.Fatalf("정의 진단이 없어야 한다:\n%s", diags.Err())
	}
	if def == nil {
		t.Fatal("정의가 nil이다")
	}
	return def
}

// testComponents는 step · steps · marker 세 정의의 어휘다.
func testComponents(t *testing.T) *Components {
	t.Helper()
	comps, diags := NewComponents(
		mustComponent(t, "themes/corp/components/step.slide", stepDefine),
		mustComponent(t, "themes/corp/components/steps.slide", stepsDefine),
		mustComponent(t, "themes/corp/components/marker.slide", markerDefine),
	)
	if len(diags) != 0 {
		t.Fatalf("어휘 진단이 없어야 한다:\n%s", diags.Err())
	}
	return comps
}

func TestParseComponent(t *testing.T) {
	def := mustComponent(t, "themes/corp/components/step.slide", stepDefine)
	if def.Name != "step" || def.NameSpan.Start.String() != "1:15" {
		t.Errorf("이름 = %q @ %s", def.Name, def.NameSpan.Start)
	}
	if got := dumpParams(def.Params); got != "icon label note?" {
		t.Errorf("인자 = %q", got)
	}
	if def.Body == nil || def.Body.Name != "col" {
		t.Fatalf("몸체 = %+v", def.Body)
	}
	if !def.HasSlot() || def.slotParent != "text" {
		t.Errorf("slot 부모 = %q", def.slotParent)
	}
	// 몸체는 `{name}`을 원문 그대로 담는다 — 치환은 펼칠 때 한다.
	if got := dumpNode(def.Body); !strings.Contains(got, "src={icon}") || !strings.Contains(got, `"{label}"`) {
		t.Errorf("몸체 = %s", got)
	}
	comps, _ := NewComponents(def)
	step, _ := comps.Lookup("step")
	if got := strings.Join(step.Attrs(), " "); !strings.HasPrefix(got, "icon label note grow shrink basis") || !strings.HasSuffix(got, "slot enter") {
		t.Errorf("사용처 속성 = %q", got)
	}
}

func dumpParams(params []ComponentParam) string {
	words := make([]string, len(params))
	for i, param := range params {
		words[i] = param.Name
		if param.Optional {
			words[i] += "?"
		}
	}
	return strings.Join(words, " ")
}

func TestParseComponentDiagnostics(t *testing.T) {
	tests := []diagCase{
		{"이름 없음", `<define><col/></define>`, []string{"1:2 <define> needs a name"}},
		{"빈 이름", `<define name=""><col/></define>`, []string{"1:9 the name attribute needs a value"}},
		{"이름 문법", `<define name="Step"><col/></define>`, []string{`1:15 component name "Step" must be lowercase`}},
		{"내장 태그 이름", `<define name="row"><col/></define>`, []string{`1:15 component name "row" is a built-in tag`}},
		{"slot이라는 이름", `<define name="slot"><col/></define>`, []string{`1:15 component name "slot" is a built-in tag`}},
		{"중복 인자", `<define name="card" params="v v"><text>{v}</text></define>`, []string{"1:31 duplicate parameter v"}},
		{"인자 문법", `<define name="card" params="v Y"><text>{v}</text></define>`, []string{`1:31 parameter "Y" must be lowercase`}},
		{"배치 속성과 겹치는 인자", `<define name="card" params="grow"><text>{grow}</text></define>`, []string{"1:29 parameter grow collides with the placement attribute"}},
		{"빈 params", `<define name="card" params=""><col/></define>`, []string{"1:29 the params attribute is empty"}},
		{"모르는 인자 자리 — 후보", `<define name="card" params="label"><text>{lable}</text></define>`, []string{"1:42 unknown parameter {lable} — did you mean {label}? (params: label)"}},
		{"모르는 인자 자리 — 속성 값", `<define name="card" params="icon"><image src="{ikon}"/></define>`, []string{"1:47 unknown parameter {ikon} — did you mean {icon}?"}},
		{"인자를 선언하지 않은 정의", `<define name="card"><text>{v}</text></define>`, []string{"1:27 unknown parameter {v} (this component declares no params"}},
		{"이름 문법 밖의 중괄호는 글자다", `<define name="card"><text>{ x } {} {X}</text></define>`, nil},
		{"몸체 없음", `<define name="card"></define>`, []string{"1:2 <define> needs one element as its body"}},
		{"몸체 둘", `<define name="card"><text>a</text><text>b</text></define>`, []string{"1:36 a component has exactly one root element"}},
		{"slot 둘", `<define name="card"><col><slot/><slot/></col></define>`, []string{"1:34 a component has at most one <slot/>"}},
		{"slot이 몸체", `<define name="card"><slot/></define>`, []string{"1:22 <slot/> cannot be the body of a component"}},
		{"글자만 담는 부모 안의 slot", `<define name="card"><pre><slot/></pre></define>`, []string{"1:27 <slot/> cannot be inside <pre>"}},
		{"slot은 속성을 받지 않는다", `<define name="card"><col><slot name="v"/></col></define>`, []string{"1:32 <slot> takes no attributes"}},
		{"몸체에 노트는 올 수 없다", `<define name="card"><note>x</note></define>`, []string{"1:22 <define> cannot contain <note>"}},
		{"컴포넌트 파일의 루트", `<slide><text>x</text></slide>`, []string{"1:2 the root element must be <define>"}},
		{"루트 둘", `<define name="card"><col/></define><define name="b"><col/></define>`, []string{"1:37 a file must have exactly one root element <define> (one component per file)"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := ParseComponent("themes/corp/components/card.slide", tc.src)
			assertDiags(t, diags, tc.want, SeverityError)
		})
	}
}

func TestParseComponentUnusedParamWarns(t *testing.T) {
	_, diags := ParseComponent("c.slide", `<define name="card" params="label icon"><text>{label}</text></define>`)
	assertDiags(t, diags, []string{"1:35 parameter icon is never used in the body of <card>"}, SeverityWarning)
}

// assertDiags는 진단을 "line:col 메시지조각" 목록과 순서대로 대조한다.
func assertDiags(t *testing.T, diags Diagnostics, want []string, severity Severity) {
	t.Helper()
	for i, w := range want {
		pos, message, _ := strings.Cut(w, " ")
		if i >= len(diags) {
			t.Errorf("진단 %d개뿐이다, want %q", len(diags), w)
			break
		}
		if got := diags[i].Span.Start.String(); got != pos {
			t.Errorf("진단[%d] 위치 = %s, want %s (%s)", i, got, pos, diags[i].Message)
		}
		if !strings.Contains(diags[i].Message, message) {
			t.Errorf("진단[%d] 메시지 = %q, want %q 포함", i, diags[i].Message, message)
		}
		if diags[i].Severity != severity {
			t.Errorf("진단[%d] 심각도 = %v, want %v", i, diags[i].Severity, severity)
		}
	}
	if len(diags) != len(want) {
		t.Errorf("진단 %d개, want %d개:\n%s", len(diags), len(want), diags.Err())
	}
}

func TestParseWithComponents(t *testing.T) {
	comps := testComponents(t)
	src := `<slide>
  <h1>Phases</h1>
  <steps>
    <step icon="theme:icons/a.png" label="Discovery" grow="1">Model-agnostic <em>and</em> evolving</step>
    <step icon="theme:icons/b.png" label="Validation"/>
  </steps>
  <row><marker year="2021"/></row>
</slide>`
	doc, diags := ParseWith("slides/001.slide", src, comps)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다:\n%s", diags.Err())
	}
	want := `h1("Phases") steps(step[icon=theme:icons/a.png label=Discovery grow=1]("Model-agnostic " em("and") " evolving") step[icon=theme:icons/b.png label=Validation]) row(marker[year=2021])`
	if got := dumpNodes(doc.Content); got != want {
		t.Errorf("트리 = %s\nwant %s", got, want)
	}
}

func TestParseWithComponentDiagnostics(t *testing.T) {
	comps := testComponents(t)
	tests := []diagCase{
		{"빠진 필수 인자", `<slide><step icon="a.png"/></slide>`,
			[]string{`1:9 <step> is missing parameter label (write label="…"; defined at themes/corp/components/step.slide:1:15)`}},
		{"빠진 필수 인자 둘", `<slide><step/></slide>`, []string{"1:9 <step> is missing parameters icon, label"}},
		{"값 없는 인자", `<slide><step icon label="x"/></slide>`, []string{"1:14 parameter icon on <step> needs a value"}},
		{"모르는 속성 — 후보", `<slide><step icon="a" label="b" lable="c"/></slide>`,
			[]string{"1:33 unknown attribute lable on <step> — did you mean label? (available attributes: icon, label, note, grow"}},
		{"모르는 태그 — 컴포넌트 후보", `<slide><stpe/></slide>`, []string{"1:9 unknown tag <stpe> — did you mean <step>?"}},
		{"뿌리가 놓일 수 없는 자리", `<slide><ul><marker year="1"/></ul></slide>`, []string{"1:13 <ul> cannot contain <marker> (available tags: <li>)"}},
		{"slot 부모의 문법을 따른다 — 행 안의 글자", `<slide><steps>loose</steps></slide>`, []string{"1:15 character data cannot appear directly inside <steps>"}},
		{"slot 부모의 문법을 따른다 — 인라인 안의 블록", `<slide><step icon="a" label="b"><p>x</p></step></slide>`, []string{"1:34 block tag <p> is not allowed inside <step>"}},
		{"slot 없는 컴포넌트는 void다", `<slide><marker year="1">x</marker></slide>`, []string{"1:9 void tag <marker> cannot have content"}},
		{"정의 파일 밖의 slot", `<slide><slot/></slide>`, []string{"1:9 unknown tag <slot>"}},
		{"목록은 그 자리에 놓일 수 있는 컴포넌트를 말한다", `<slide><row><bogus/></row></slide>`, []string{"1:14 unknown tag <bogus> (tags available inside <row>: <text>"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := ParseWith("slides/001.slide", tc.src, comps)
			assertDiags(t, diags, tc.want, SeverityError)
		})
	}
	_, diags := ParseWith("slides/001.slide", `<slide><row><bogus/></row></slide>`, comps)
	if msg := diags[0].Message; !strings.HasSuffix(msg, "<spacer>, <marker>, <step>, <steps>)") {
		t.Errorf("자리 목록에 컴포넌트가 붙어야 한다: %q", msg)
	}
}

// 컴포넌트를 모르는 파스(Parse)에서 같은 태그는 지금처럼 모르는 태그다 — 어휘는 테마가 연다.
func TestParseWithoutComponentsRejectsThem(t *testing.T) {
	_, diags := Parse("slides/001.slide", `<slide><step icon="a" label="b"/></slide>`)
	assertDiags(t, diags, []string{"1:9 unknown tag <step>"}, SeverityError)
}

func TestNewComponentsDuplicate(t *testing.T) {
	a := mustComponent(t, "themes/a/components/step.slide", `<define name="step"><col/></define>`)
	b := mustComponent(t, "themes/b/components/step.slide", `<define name="step"><row/></define>`)
	comps, diags := NewComponents(a, b)
	assertDiags(t, diags, []string{"1:15 component <step> is already defined at themes/a/components/step.slide:1:15"}, SeverityError)
	if got := diags[0].File; got != "themes/b/components/step.slide" {
		t.Errorf("진단 파일 = %q", got)
	}
	if def, _ := comps.Lookup("step"); def != a {
		t.Error("먼저 정의된 것이 남아야 한다")
	}
}
