package soml

import (
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

// DiagnosticsError는 에러 심각도의 진단이 빌드를 멈췄음을 나타내는 error다. 상세 진단은
// 이미 Diagnostics 목록에 있으므로, 이 목록을 따로 출력하는 호출자(예: CLI)는 이 에러를
// 다시 통째로 찍는 대신 Count()로 요약만 낼 수 있다 — 그래야 진단이 두 번 나오지 않는다.
type DiagnosticsError struct {
	Errors Diagnostics // 에러 심각도의 진단만.
}

// Error는 에러 진단을 줄바꿈으로 합친다 — 목록을 따로 출력하지 않는 호출자를 위한 기본 형태다.
func (e *DiagnosticsError) Error() string {
	lines := make([]string, len(e.Errors))
	for i, d := range e.Errors {
		lines[i] = d.String()
	}
	return strings.Join(lines, "\n")
}

// Count는 빌드를 멈춘 에러 진단의 개수다.
func (e *DiagnosticsError) Count() int { return len(e.Errors) }

// Err은 에러가 있으면 *DiagnosticsError를, 없으면 nil을 돌려준다.
func (ds Diagnostics) Err() error {
	if !ds.HasErrors() {
		return nil
	}
	errs := make(Diagnostics, 0, len(ds))
	for _, d := range ds {
		if d.Severity == SeverityError {
			errs = append(errs, d)
		}
	}
	return &DiagnosticsError{Errors: errs}
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
