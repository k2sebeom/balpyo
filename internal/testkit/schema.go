package testkit

import (
	"embed"
	"fmt"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/knroy/go-xml/xdm"
	"github.com/knroy/go-xml/xsd"
)

// 스키마 검증: 패키지의 XML 파트마다 ECMA-376 Transitional 스키마로 검증한다.
// PowerPoint가 복구를 묻는 흔한 원인 — 자식 순서 · 필수 속성 · 열거값 · 타입 · 범위 — 는 전부 스키마
// 위반이고, LibreOffice는 그것을 조용히 받아 주었다. 스키마 원본은 ooxml/에 고치지 않고 둔다.

//go:embed ooxml/*.xsd
var ooxmlFS embed.FS

// ooxmlSchemas는 스키마 파일과 그 이름공간이다. 파일마다 이름공간이 하나라 import는 이름공간으로
// 풀린다 — `dc.xsd`처럼 원격 URL로 적힌 schemaLocation도 네트워크를 타지 않는다.
var ooxmlSchemas = []xsd.CatalogEntry{
	{Path: "ooxml/pml.xsd", Namespace: "http://schemas.openxmlformats.org/presentationml/2006/main"},
	{Path: "ooxml/dml-main.xsd", Namespace: "http://schemas.openxmlformats.org/drawingml/2006/main"},
	{Path: "ooxml/dml-chart.xsd", Namespace: "http://schemas.openxmlformats.org/drawingml/2006/chart"},
	{Path: "ooxml/dml-chartDrawing.xsd", Namespace: "http://schemas.openxmlformats.org/drawingml/2006/chartDrawing"},
	{Path: "ooxml/dml-diagram.xsd", Namespace: "http://schemas.openxmlformats.org/drawingml/2006/diagram"},
	{Path: "ooxml/dml-lockedCanvas.xsd", Namespace: "http://schemas.openxmlformats.org/drawingml/2006/lockedCanvas"},
	{Path: "ooxml/dml-picture.xsd", Namespace: "http://schemas.openxmlformats.org/drawingml/2006/picture"},
	{Path: "ooxml/shared-commonSimpleTypes.xsd", Namespace: "http://schemas.openxmlformats.org/officeDocument/2006/sharedTypes"},
	{Path: "ooxml/shared-relationshipReference.xsd", Namespace: "http://schemas.openxmlformats.org/officeDocument/2006/relationships"},
	{Path: "ooxml/shared-documentPropertiesExtended.xsd", Namespace: "http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"},
	{Path: "ooxml/shared-documentPropertiesVariantTypes.xsd", Namespace: "http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"},
	{Path: "ooxml/opc-contentTypes.xsd", Namespace: "http://schemas.openxmlformats.org/package/2006/content-types"},
	{Path: "ooxml/opc-relationships.xsd", Namespace: "http://schemas.openxmlformats.org/package/2006/relationships"},
	{Path: "ooxml/opc-coreProperties.xsd", Namespace: "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"},
	{Path: "ooxml/dc.xsd", Namespace: "http://purl.org/dc/elements/1.1/"},
	{Path: "ooxml/dcterms.xsd", Namespace: "http://purl.org/dc/terms/"},
	{Path: "ooxml/dcmitype.xsd", Namespace: "http://purl.org/dc/dcmitype/"},
	{Path: "ooxml/xml.xsd", Namespace: "http://www.w3.org/XML/1998/namespace"},
}

var (
	ooxmlOnce   sync.Once
	ooxmlSchema *xsd.Schema
	ooxmlErr    error
)

// OOXMLSchema는 적재된 스키마 집합이다 — 실행마다 한 번 읽는다 (~80ms).
func OOXMLSchema(t testing.TB) *xsd.Schema {
	t.Helper()
	ooxmlOnce.Do(func() { ooxmlSchema, ooxmlErr = loadOOXMLSchema() })
	if ooxmlErr != nil {
		t.Fatalf("ECMA-376 스키마를 적재할 수 없다: %v", ooxmlErr)
	}
	return ooxmlSchema
}

func loadOOXMLSchema() (*xsd.Schema, error) {
	catalog := xsd.NewCatalogResolver()
	entries := make([]xsd.CatalogEntry, len(ooxmlSchemas))
	for i, entry := range ooxmlSchemas {
		// 같은 디렉토리의 상대 schemaLocation("dml-main.xsd")도 같은 항목이 답한다.
		entry.Aliases = []string{path.Base(entry.Path)}
		entries[i] = entry
	}
	if err := catalog.AddFromFS(ooxmlFS, entries); err != nil {
		return nil, err
	}
	// 루트는 이름공간마다 import 하나인 빈 스키마다 — 파트의 뿌리 요소가 어느 이름공간이든 선언이 있다.
	var root strings.Builder
	root.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`)
	for _, entry := range ooxmlSchemas {
		fmt.Fprintf(&root, `<xs:import namespace="%s"/>`, entry.Namespace)
	}
	root.WriteString(`</xs:schema>`)
	tree, err := xdm.ParseString(root.String(), xdm.ParseOptions{})
	if err != nil {
		return nil, err
	}
	return xsd.Load(tree.Root, "ooxml/root.xsd", xsd.Options{Resolver: catalog})
}

// ValidatePart는 XML 파트 하나를 스키마로 검증한다. 뿌리 요소에 선언이 없는 파트(모르는 이름공간)도
// 실패다 — 검증을 건너뛴 파트가 초록으로 보이면 안 된다.
func ValidatePart(t testing.TB, data []byte) error {
	t.Helper()
	tree, err := xdm.ParseString(string(data), xdm.ParseOptions{})
	if err != nil {
		return err
	}
	return OOXMLSchema(t).Validate(tree.Root, xsd.ValidateOptions{MaxErrors: 5})
}

// AssertSchema는 패키지의 모든 XML 파트(`.xml` · `.rels`)가 스키마에 맞는지 본다.
func AssertSchema(t testing.TB, parts map[string][]byte) {
	t.Helper()
	for _, name := range sortedParts(parts) {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".rels") {
			continue
		}
		if err := ValidatePart(t, parts[name]); err != nil {
			t.Errorf("%s가 ECMA-376 스키마에 맞지 않는다 — PowerPoint가 복구를 묻는다:\n%v", name, err)
		}
	}
}
