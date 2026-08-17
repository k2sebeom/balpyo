package soml

import (
	"errors"
	"fmt"
	"strings"
)

// Severity는 진단의 심각도다.
type Severity int

const (
	// SeverityError는 빌드를 실패시킨다.
	SeverityError Severity = iota
	// SeverityWarning은 빌드를 통과시킨다. --strict가 에러로 승격한다.
	SeverityWarning
)

// String은 진단 메시지에 찍히는 이름이다.
func (s Severity) String() string {
	if s == SeverityWarning {
		return "warning"
	}
	return "error"
}

// Diagnostic은 위치를 가진 진단 하나다. 파서·검증기·렌더러가 같은 타입을 쓴다 —
// 에디터에서 본 경고와 CI가 막는 경고가 어긋날 수 없게 하는 최소 장치다.
type Diagnostic struct {
	Severity Severity
	// File은 프로젝트 루트 기준 상대 경로다.
	File    string
	Span    Span
	Message string
}

// Errorf는 에러 진단을 만든다.
func Errorf(file string, span Span, format string, args ...any) Diagnostic {
	return Diagnostic{Severity: SeverityError, File: file, Span: span, Message: fmt.Sprintf(format, args...)}
}

// Warningf는 경고 진단을 만든다.
func Warningf(file string, span Span, format string, args ...any) Diagnostic {
	return Diagnostic{Severity: SeverityWarning, File: file, Span: span, Message: fmt.Sprintf(format, args...)}
}

// String은 `file:line:col: severity: message` 형태로 포맷한다 — 에디터가 클릭할 수 있는
// 관례적 형태다. 위치가 없으면 위치 부분을 생략한다.
func (d Diagnostic) String() string {
	var b strings.Builder
	if d.File != "" {
		b.WriteString(d.File)
		if d.Span.Start.IsValid() {
			b.WriteString(":")
			b.WriteString(d.Span.Start.String())
		}
		b.WriteString(": ")
	}
	b.WriteString(d.Severity.String())
	b.WriteString(": ")
	b.WriteString(d.Message)
	return b.String()
}

// Diagnostics는 진단 목록이다.
type Diagnostics []Diagnostic

// HasErrors는 에러 심각도의 진단이 하나라도 있는지 보고한다.
func (ds Diagnostics) HasErrors() bool {
	for _, d := range ds {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Err은 에러가 있으면 전부를 줄바꿈으로 합친 error를, 없으면 nil을 돌려준다.
func (ds Diagnostics) Err() error {
	if !ds.HasErrors() {
		return nil
	}
	lines := make([]string, 0, len(ds))
	for _, d := range ds {
		if d.Severity == SeverityError {
			lines = append(lines, d.String())
		}
	}
	return errors.New(strings.Join(lines, "\n"))
}

// Strict는 경고를 에러로 승격한 새 목록을 돌려준다 (`--strict`).
func (ds Diagnostics) Strict() Diagnostics {
	out := make(Diagnostics, len(ds))
	copy(out, ds)
	for i := range out {
		out[i].Severity = SeverityError
	}
	return out
}
