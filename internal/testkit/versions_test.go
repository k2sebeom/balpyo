package testkit

import "testing"

// 골든 좌표와 픽셀 스냅샷은 이미지 안의 도구 버전에 묶여 있다. 보안 패치로 버전이 움직이면
// 골든이 조용히 깨지는 대신 이 테스트가 먼저 실패한다 (docker/versions.env의 대처 절차 참고).
func TestImageVersionsMatchPins(t *testing.T) {
	pins := PinnedVersions(t)
	actual, inImage := ImageVersions(t)
	if !inImage {
		// 이미지 밖에서는 대조할 대상이 없다. Strict(이미지 안)에서는 파일이 반드시 있다.
		if Strict() {
			t.Fatalf("%s가 없다 — 테스트 이미지가 아닌 곳에서 Strict로 돌고 있다", ImageVersionsFile)
		}
		t.Skipf("%s가 없다 (테스트 이미지 밖) — `make test`로 이미지 안에서 돌린다", ImageVersionsFile)
	}
	for _, mismatch := range CompareVersions(pins, actual) {
		t.Errorf("버전 핀 불일치: %s", mismatch)
	}
}

func TestCompareVersions(t *testing.T) {
	pins := map[string]string{
		"BALPYO_GO":              "go1.26.6",
		"BALPYO_SOFFICE_VERSION": "LibreOffice 7.4.7.2",
		"BALPYO_MISSING":         "1.0",
	}
	actual := map[string]string{
		"BALPYO_GO": "go1.26.7",
		// 접두 일치이므로 뒤쪽 빌드 문자열은 통과해야 한다.
		"BALPYO_SOFFICE_VERSION": "LibreOffice 7.4.7.2 40(Build:2)",
	}
	mismatches := CompareVersions(pins, actual)
	if len(mismatches) != 2 {
		t.Fatalf("불일치 %d건, want 2건: %v", len(mismatches), mismatches)
	}
	// 이름 순이므로 GO가 먼저, MISSING이 다음이다.
	if got := mismatches[0]; got == "" || got[:9] != "BALPYO_GO" {
		t.Errorf("첫 불일치 = %q, want BALPYO_GO", got)
	}
	if len(CompareVersions(map[string]string{"A": "1"}, map[string]string{"A": "1.2"})) != 0 {
		t.Error("접두 일치는 통과해야 한다")
	}
}

func TestPinnedVersionsParsesRepoFile(t *testing.T) {
	// 핀 파일은 주석이 많다 — 파서가 주석을 값으로 오해하면 대조가 무의미해진다.
	pins := PinnedVersions(t)
	for _, key := range []string{"BALPYO_IMAGE_BASE", "BALPYO_GO", "BALPYO_SOFFICE_VERSION",
		"BALPYO_CHROMIUM_VERSION", "BALPYO_PDFTOPPM_VERSION", "BALPYO_FONT_SANS", "BALPYO_FONT_CJK",
		"BALPYO_FONT_MONO"} {
		if pins[key] == "" {
			t.Errorf("%s에 %s 핀이 없다", PinsFile, key)
		}
	}
}
