package geom_test

import (
	"os"
	"testing"

	"github.com/k2sebeom/balpyo/internal/testkit"
)

// TestMain은 공유 브라우저 세션을 스위트가 끝난 뒤 닫는다 (testkit.SharedSession).
func TestMain(m *testing.M) {
	code := m.Run()
	testkit.CloseSharedSession()
	os.Exit(code)
}
