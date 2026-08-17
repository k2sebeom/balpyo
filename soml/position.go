package soml

import "fmt"

// Pos는 소스 안의 한 지점이다.
//
// Offset은 0-based 바이트 오프셋, Line과 Col은 1-based다. Col은 **바이트가 아니라
// 룬(rune) 기준**으로 센다 — 진단 메시지가 사람에게 보이는 좌표계이므로 한글 한 글자가
// 세 칸으로 세어지면 안 된다. LSP가 요구하는 UTF-16 좌표 변환은 lsp 층의 몫이다.
type Pos struct {
	Offset int
	Line   int
	Col    int
}

// StartPos는 소스의 첫 지점이다.
func StartPos() Pos { return Pos{Offset: 0, Line: 1, Col: 1} }

// IsValid는 Pos가 실제 지점을 가리키는지 보고한다. 제로값은 무효다.
func (p Pos) IsValid() bool { return p.Line > 0 && p.Col > 0 }

// String은 "line:col" 형태로 포맷한다.
func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Span은 소스의 반열린 구간 [Start, End)다.
type Span struct {
	Start Pos
	End   Pos
}

// SpanOf는 두 지점으로 Span을 만든다.
func SpanOf(start, end Pos) Span { return Span{Start: start, End: end} }

// Len은 구간의 바이트 길이다.
func (s Span) Len() int { return s.End.Offset - s.Start.Offset }

// IsEmpty는 구간이 비어 있는지 보고한다.
func (s Span) IsEmpty() bool { return s.Len() == 0 }

// Text는 원본 소스에서 이 구간의 문자열을 잘라낸다. 구간이 src 밖이면 빈 문자열이다.
func (s Span) Text(src string) string {
	if s.Start.Offset < 0 || s.End.Offset > len(src) || s.Start.Offset > s.End.Offset {
		return ""
	}
	return src[s.Start.Offset:s.End.Offset]
}

// String은 시작 지점을 포맷한다 — 진단 메시지의 앵커는 항상 시작 지점이다.
func (s Span) String() string { return s.Start.String() }
