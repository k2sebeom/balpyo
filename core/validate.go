package core

import (
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// Validate는 덱 모델 전체에 걸친 규칙을 검사한다.
//
// 파서가 파일 하나 안에서 볼 수 있는 것(닫힌 어휘, 문법)은 파서가 진단한다. 여기서는
// **파일을 넘어야 알 수 있는 것**만 본다 — 슬라이드 ID 충돌과, 다른 슬라이드를 가리키는
// 링크(`<a href="#…">`)의 목적지가 그것이다.
//
// check·build·lsp가 이 함수를 공유하므로 에디터에서 본 진단과 CI가 막는 진단이 같다.
func Validate(deck *soml.Deck) soml.Diagnostics {
	var diags soml.Diagnostics

	// 슬라이드 ID는 딥링크 앵커다. 충돌하면 링크가 어느 슬라이드를 가리키는지 알 수 없다.
	seen := make(map[string]*soml.Slide, len(deck.Slides))
	for _, slide := range deck.Slides {
		if slide.ID == "" {
			diags = append(diags, soml.Errorf(slide.Source, soml.Span{},
				"slide ID is empty"))
			continue
		}
		if first, dup := seen[slide.ID]; dup {
			diags = append(diags, soml.Errorf(slide.Source, slide.IDSpan,
				"slide ID %q collides with %s (give one of them a different id, or remove the slide you do not need)",
				slide.ID, sourceAt(first.Source, first.IDSpan)))
			continue
		}
		seen[slide.ID] = slide
	}
	for _, slide := range deck.Slides {
		diags = append(diags, validateSlideLinks(deck, slide, slide.Content)...)
	}
	return diags
}

// validateSlideLinks는 덱 안을 가리키는 링크의 목적지가 이 덱에 있는지 본다.
//
// 경고인 이유: 목적지 없는 링크는 클릭이 엉뚱한 곳(첫 슬라이드)으로 갈 뿐 산출물은 온전하다.
// 그리고 가장 흔한 원인이 링크된 슬라이드를 잠시 skip한 것이라, 에러로 막으면 skip 한 줄이
// 빌드를 깬다. `--strict`가 CI에서 이것을 막는다.
func validateSlideLinks(deck *soml.Deck, slide *soml.Slide, nodes []soml.Node) soml.Diagnostics {
	var diags soml.Diagnostics
	for _, node := range nodes {
		elem, ok := node.(*soml.Element)
		if !ok {
			continue
		}
		if elem.Appearance != nil && elem.Appearance.Link != "" {
			href := elem.Appearance.Link
			if target, inDeck := deck.LinkedSlide(href); inDeck && target == nil {
				span := elem.NameSpan
				if attr, ok := elem.Attr("href"); ok {
					span = attr.ValueSpan
				}
				diags = append(diags, soml.Warningf(slide.Source, span,
					"href %q points to no slide in this deck%s — link to a slide id (\"#cover\") or a 1-based slide number (\"#/2\"); skipped slides are not in the deck",
					href, slideLinkHint(deck, href)))
			}
		}
		diags = append(diags, validateSlideLinks(deck, slide, elem.Children)...)
	}
	return diags
}

// slideLinkHint는 링크 목적지 오타의 후보다 (예: `#secnd` → `#second`). 없으면 빈 문자열이다.
func slideLinkHint(deck *soml.Deck, href string) string {
	ids := make([]string, 0, len(deck.Slides))
	for _, s := range deck.Slides {
		ids = append(ids, s.ID)
	}
	raw := strings.TrimPrefix(strings.TrimPrefix(href, "#"), "/")
	if guess, ok := soml.Suggest(raw, ids); ok {
		return " (did you mean " + strconv.Quote("#"+guess) + "?)"
	}
	return ""
}

// sourceAt은 `file:line:col`을 만든다 — 진단의 앞머리와 같은 형태라 에디터가 메시지 안의
// 다른 자리도 클릭할 수 있다. 위치가 없으면 파일만 쓴다.
func sourceAt(file string, span soml.Span) string {
	if !span.Start.IsValid() {
		return file
	}
	return file + ":" + span.Start.String()
}
