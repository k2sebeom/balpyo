package soml

import "testing"

// 표와 차트의 정상 트리 — 셀 병합·외관, 셀 안 인라인, 차트의 점과 고유 속성이 파싱을 그대로
// 통과한다. 값 어휘(병합 수·숫자·색)는 값 층이 본다.
func TestParseTableAndChart(t *testing.T) {
	runTreeCases(t, []treeCase{
		{
			name: "헤더 행 + 본문 행",
			src:  `<slide><table><tr><th>지역</th><th>Q1</th></tr><tr><td>남미</td><td>18%</td></tr></table></slide>`,
			want: `table(tr(th("지역") th("Q1")) tr(td("남미") td("18%")))`,
		},
		{
			name: "셀의 병합·외관 속성과 인라인 서식",
			src:  `<slide><table><tr><td colspan="2" rowspan="2" fill="accent1" border="1 solid muted" color="bg" align="center" anchor="middle">매출 <strong>+18%</strong></td></tr></table></slide>`,
			want: `table(tr(td[colspan=2 rowspan=2 fill=accent1 border=1 solid muted color=bg align=center anchor=middle]("매출 " strong("+18%"))))`,
		},
		{
			name: "표 프레임의 variant·배치·슬롯",
			src:  `<slide layout="content"><col><table variant="striped" grow="1"><tr><td>x</td></tr></table></col><table slot="body"><tr><td>y</td></tr></table></slide>`,
			want: `col(table[variant=striped grow=1](tr(td("x")))) table[slot=body](tr(td("y")))`,
		},
		{
			name: "차트의 고유 속성과 점",
			src:  `<slide><chart type="bar" unit="분" max="25" series="accent2" alt="월별 탐지 시간" h="420"><point label="5월" value="22"/><point label="9월" value="12" fill="accent4"/></chart></slide>`,
			want: `chart[type=bar unit=분 max=25 series=accent2 alt=월별 탐지 시간 h=420](point[label=5월 value=22] point[label=9월 value=12 fill=accent4])`,
		},
		{
			name: "차트는 컨테이너의 자식이고 슬롯을 채운다",
			src:  `<slide layout="content"><row><chart type="bar" grow="1"><point label="a" value="1"/></chart></row><chart slot="body"/></slide>`,
			want: `row(chart[type=bar grow=1](point[label=a value=1])) chart[slot=body]`,
		},
	})
}

// 표·차트의 배치 위반 — 문자 데이터의 조언은 **부모가** 정한다 (`<text>로 감싼다`는 표 안에서
// 따를 수 없는 말이다).
func TestParseTableAndChartErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{name: "tr은 table 밖에 올 수 없다", src: `<slide><tr><td>x</td></tr></slide>`, want: []string{"1:9 <tr> can only be a child of <table>"}},
		{name: "td는 tr 밖에 올 수 없다", src: `<slide><table><td>x</td></table></slide>`, want: []string{"1:16 <th>/<td> can only be children of <tr>"}},
		{name: "table의 자식은 tr뿐", src: `<slide><table><p>본문</p></table></slide>`, want: []string{"1:16 <table> cannot contain <p> (available tags: <tr>)"}},
		{name: "셀 안의 블록", src: `<slide><table><tr><td><p>본문</p></td></tr></table></slide>`, want: []string{"1:24 block tag <p> is not allowed inside <td>"}},
		{name: "table 안의 문자 데이터", src: `<slide><table>지역</table></slide>`, want: []string{"1:15 character data cannot appear directly inside <table> (wrap it in <tr>)"}},
		{name: "tr 안의 문자 데이터", src: `<slide><table><tr>지역</tr></table></slide>`, want: []string{"1:19 character data cannot appear directly inside <tr> (wrap it in <td>)"}},
		{name: "point는 chart 밖에 올 수 없다", src: `<slide><point label="a" value="1"/></slide>`, want: []string{"1:9 <point> can only be a child of <chart>"}},
		{name: "chart의 자식은 point뿐", src: `<slide><chart type="bar"><p>본문</p></chart></slide>`, want: []string{"1:27 <chart> cannot contain <p> (available tags: <point>)"}},
		{name: "chart 안의 문자 데이터", src: `<slide><chart type="bar">22</chart></slide>`, want: []string{"1:26 character data cannot appear directly inside <chart> (wrap it in <point>)"}},
		{name: "point는 void", src: `<slide><chart><point label="a">22</point></chart></slide>`, want: []string{"1:16 void tag <point> cannot have content (write <point/>)"}},
		{name: "chart는 도형의 직속 자식이 아니다", src: `<slide><shape type="rect"><chart/></shape></slide>`, want: []string{"1:28 <shape> cannot contain <chart>"}},
	})
}
