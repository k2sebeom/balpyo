package testkit

import (
	"os"
	"testing"
)

// TestMain은 공유 브라우저 세션을 스위트가 끝난 뒤 닫는다 (SharedSession).
func TestMain(m *testing.M) {
	code := m.Run()
	CloseSharedSession()
	os.Exit(code)
}
