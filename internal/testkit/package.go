package testkit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
)

// 패키지 불변식 (TASKS R2): PPTX 패키지의 구조는 스냅샷이 아니라 **어느 덱에서나 성립하는 규칙**으로
// 단정한다. 파트별 XML 골든은 CSS 한 줄에도 수십 파일이 움직였고 리뷰어가 그 diff를 읽지 않았다 —
// 규칙은 움직이지 않고, 깨지면 무엇이 깨졌는지를 말한다.
//
//   - 파트마다 content type이 있다 (Default 확장자 또는 Override) · Override는 있는 파트를 가리킨다.
//   - 모든 내부 관계의 대상이 패키지에 있다 · 관계 파일의 주인 파트가 있다 · rId가 겹치지 않는다.
//   - 관계가 가리키지 않는 파트가 없다 (쓰이지 않는 파트는 파일만 부풀린다).
//   - 슬라이드 파트 수 = 덱의 장 수 · 노트 파트는 노트가 있는 장에만 있다.
//   - 미디어 파트는 에셋 표의 자리 + 포맷의 확장자이고 바이트가 입력 파일과 같다 (웹 산출 에셋도).
//   - zip 타임스탬프는 1980-01-01이다 · 마스터의 `p:hf`는 없다 (크롬은 슬라이드의 도형이다 — ADR 0045).
//   - 모든 XML 파트가 ECMA-376 스키마에 맞는다 (schema.go — ADR 0065).
//   - 파트 사이의 참조가 맞는다 — 도형 id · 슬라이드 id · 관계 종류 · 레이아웃 연결 · 플레이스홀더 (crosspart.go).
//
// 덱과 무관한 규칙(zip · content types · rels · 스키마 · 파트 사이)은 AssertValidPPTX가 맡는다 — 픽스처가 아닌
// 단위 테스트의 덱(render/pptx)도 같은 판정을 받는다.

// contentTypesPart는 패키지의 content type 표다.
const contentTypesPart = "[Content_Types].xml"

// AssertPackage는 빌드 산출물이 패키지 불변식을 지키는지 본다.
func (f Fixture) AssertPackage(t testing.TB, result *pipeline.Result) {
	t.Helper()
	data, err := os.ReadFile(f.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := AssertValidPPTX(t, data)

	slides, notes := 0, 0
	for name, part := range parts {
		switch {
		case strings.HasPrefix(name, "ppt/slides/slide"):
			slides++
		case strings.HasPrefix(name, "ppt/notesSlides/notesSlide"):
			notes++
		}
		if bytes.Contains(part, []byte("<p:hf")) {
			t.Errorf("%s에 p:hf가 있다 — 크롬은 마스터가 아니라 슬라이드의 도형이다", name)
		}
	}
	if slides != len(result.Deck.Slides) {
		t.Errorf("슬라이드 파트 %d개, 덱은 %d장", slides, len(result.Deck.Slides))
	}
	wantNotes := 0
	for i, slide := range result.Deck.Slides {
		part := fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", i+1)
		if _, ok := parts[part]; ok != (slide.Notes != "") {
			t.Errorf("%s(%s): 노트 파트 있음=%v, 노트 있음=%v", part, slide.ID, ok, slide.Notes != "")
		}
		if slide.Notes != "" {
			wantNotes++
		}
	}
	if notes != wantNotes {
		t.Errorf("노트 파트 %d개, 노트가 있는 장 %d장", notes, wantNotes)
	}

	// 미디어: 파트 이름은 에셋 표의 자리 + 포맷의 확장자이고, 엔진은 그림을 고치지 않는다.
	media := map[string]bool{}
	for name := range parts {
		if strings.HasPrefix(name, "ppt/media/") {
			media[name] = true
		}
	}
	for i, asset := range result.Deck.Assets {
		want, err := os.ReadFile(filepath.Join(f.Dir, filepath.FromSlash(asset.Key)))
		if err != nil {
			t.Fatalf("픽스처 에셋을 읽을 수 없다: %v", err)
		}
		if got, err := os.ReadFile(filepath.Join(result.OutDir, filepath.FromSlash(asset.Key))); err != nil {
			t.Errorf("웹 산출에 에셋 %s가 없다: %v", asset.Key, err)
		} else if !bytes.Equal(got, want) {
			t.Errorf("웹 산출 %s의 바이트가 입력과 다르다 — 엔진이 그림을 고쳤다", asset.Key)
		}
		part := fmt.Sprintf("ppt/media/image%d%s", i+1, asset.Format.Ext())
		if !media[part] {
			t.Errorf("미디어 파트 %s(%s)가 없다", part, asset.Key)
		} else if !bytes.Equal(parts[part], want) {
			t.Errorf("%s의 바이트가 %s와 다르다 — 엔진이 그림을 고쳤다", part, asset.Key)
		}
		delete(media, part)
	}
	for name := range media {
		t.Errorf("에셋 표에 없는 미디어 파트가 있다: %s", name)
	}
}

// AssertValidPPTX는 덱과 무관한 패키지 규칙 — zip 타임스탬프 · content types · 관계 그래프 · 스키마 ·
// 파트 사이 참조 —
// 을 보고 파트 표를 돌려준다.
func AssertValidPPTX(t testing.TB, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("pptx를 열 수 없다: %v", err)
	}
	for _, entry := range zr.File {
		if y, m, d := entry.Modified.Date(); y != 1980 || m != 1 || d != 1 {
			t.Errorf("%s의 zip 타임스탬프가 %s다 — 같은 입력이 다른 바이트가 된다", entry.Name, entry.Modified)
		}
	}
	parts := PPTXParts(t, data)
	assertContentTypes(t, parts)
	assertRels(t, parts)
	AssertSchema(t, parts)
	AssertCrossParts(t, parts)
	return parts
}

// AssertSameBytes는 같은 입력의 두 빌드가 같은 바이트를 냈는지 본다 (불변식 6).
func (f Fixture) AssertSameBytes(t testing.TB, first, second *pipeline.Result) {
	t.Helper()
	if strings.Join(first.Written, ",") != strings.Join(second.Written, ",") {
		t.Fatalf("산출 목록이 다르다: %v vs %v", first.Written, second.Written)
	}
	for _, name := range first.Written {
		a, errA := os.ReadFile(filepath.Join(first.OutDir, filepath.FromSlash(name)))
		b, errB := os.ReadFile(filepath.Join(second.OutDir, filepath.FromSlash(name)))
		if errA != nil || errB != nil {
			t.Fatalf("산출물을 읽을 수 없다: %v / %v", errA, errB)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s: 같은 입력의 두 빌드가 다른 바이트를 냈다", name)
		}
	}
}

// assertContentTypes는 파트마다 content type이 있는지 본다 — 없는 파트는 PowerPoint가 복구를 묻는다.
func assertContentTypes(t testing.TB, parts map[string][]byte) {
	t.Helper()
	var types struct {
		Defaults []struct {
			Extension string `xml:"Extension,attr"`
		} `xml:"Default"`
		Overrides []struct {
			PartName string `xml:"PartName,attr"`
		} `xml:"Override"`
	}
	if err := xml.Unmarshal(parts[contentTypesPart], &types); err != nil {
		t.Fatalf("%s를 읽을 수 없다: %v", contentTypesPart, err)
	}
	defaults, overrides := map[string]bool{}, map[string]bool{}
	for _, d := range types.Defaults {
		defaults[strings.ToLower(d.Extension)] = true
	}
	for _, o := range types.Overrides {
		name := strings.TrimPrefix(o.PartName, "/")
		overrides[name] = true
		if _, ok := parts[name]; !ok {
			t.Errorf("content type Override가 없는 파트를 가리킨다: %s", o.PartName)
		}
	}
	for _, name := range sortedParts(parts) {
		if name == contentTypesPart || overrides[name] {
			continue
		}
		if !defaults[strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))] {
			t.Errorf("파트 %s에 content type이 없다", name)
		}
	}
}

// assertRels는 관계 그래프를 본다: 대상이 있고, 주인이 있고, 아무도 가리키지 않는 파트가 없다.
func assertRels(t testing.TB, parts map[string][]byte) {
	t.Helper()
	targeted := map[string]bool{}
	for _, name := range sortedParts(parts) {
		if !strings.HasSuffix(name, ".rels") {
			continue
		}
		// `<dir>/_rels/<part>.rels`의 주인은 `<dir>/<part>`다 (패키지 루트의 `_rels/.rels`는 주인이 없다).
		dir := path.Dir(path.Dir(name))
		if dir == "." {
			dir = ""
		}
		if owner := path.Join(dir, strings.TrimSuffix(path.Base(name), ".rels")); owner != "" {
			if _, ok := parts[owner]; !ok {
				t.Errorf("%s의 주인 파트 %s가 없다", name, owner)
			}
		}
		var rels struct {
			Rels []struct {
				ID         string `xml:"Id,attr"`
				Target     string `xml:"Target,attr"`
				TargetMode string `xml:"TargetMode,attr"`
			} `xml:"Relationship"`
		}
		if err := xml.Unmarshal(parts[name], &rels); err != nil {
			t.Fatalf("%s를 읽을 수 없다: %v", name, err)
		}
		ids := map[string]bool{}
		for _, rel := range rels.Rels {
			if ids[rel.ID] {
				t.Errorf("%s에 rId %s가 두 번 있다", name, rel.ID)
			}
			ids[rel.ID] = true
			if rel.TargetMode == "External" {
				continue
			}
			target := strings.TrimPrefix(rel.Target, "/")
			if !strings.HasPrefix(rel.Target, "/") {
				target = path.Join(dir, rel.Target)
			}
			if _, ok := parts[target]; !ok {
				t.Errorf("%s의 %s가 없는 파트 %s를 가리킨다", name, rel.ID, target)
			}
			targeted[target] = true
		}
	}
	for _, name := range sortedParts(parts) {
		if name != contentTypesPart && !strings.HasSuffix(name, ".rels") && !targeted[name] {
			t.Errorf("아무 관계도 가리키지 않는 파트가 있다: %s", name)
		}
	}
}

// sortedParts는 파트 이름을 정렬해 돌려준다 — 실패 메시지의 순서가 실행마다 같게 한다.
func sortedParts(parts map[string][]byte) []string {
	names := make([]string, 0, len(parts))
	for name := range parts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
