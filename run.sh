#!/usr/bin/env bash
#
# run.sh — iteration 하나를 **항목 단위**로 돌린다.
#
# 항목마다 `claude -p`를 새 세션으로 한 번 띄운다. 세션을 이어 쓰지 않는 이유는 항목이
# 서로의 맥락을 물려받으면 앞 항목의 판단이 뒤 항목에 검증 없이 실려 가기 때문이다 —
# 각 항목은 TASKS.md와 PRD.md, 그리고 **이미 커밋된 코드**만 근거로 삼는다.
#
# 게이트는 에이전트의 보고를 믿지 않고 **이 스크립트가 직접 돌린다** (gofmt·vet·go test·
# make test). 보고와 실제가 갈라지는 것이 이 방식에서 가장 비싼 실패라서다: 초록이라고
# 적힌 커밋이 쌓이면 어느 항목에서 깨졌는지 나중에 찾아야 한다.
#
# 사용법:
#   ./run.sh <iteration> [item…]        # ./run.sh 1        → iterations/01/*.md 전부
#                                       # ./run.sh 1 05 06  → 접두가 맞는 항목만
#   ./run.sh --list 1                   # 항목 목록과 완료 여부만 본다
#
# 환경 변수:
#   MODEL=opus              claude --model (기본: 설정된 세션 모델)
#   PERMISSION_MODE=…       기본 acceptEdits. 샌드박스에서는 bypassPermissions.
#   STREAM=1                진행 상황을 실시간으로 흘린다 (jq 필요)
#   DRY_RUN=1               프롬프트만 출력하고 claude를 부르지 않는다
#   SKIP_GATES=1            게이트를 건너뛴다 (커밋 검사는 그대로 한다)
#   FORCE=1                 NOTES.md에 완료로 적힌 항목도 다시 돌린다
#   CLAUDE_BIN=…            claude 실행 파일 경로
#
set -euo pipefail

readonly ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ITER_DIR="$ROOT/iterations"
readonly PREAMBLE="$ITER_DIR/_preamble.md"
readonly NOTES="$ROOT/NOTES.md"
readonly LOG_DIR="$ROOT/.run"

CLAUDE_BIN="${CLAUDE_BIN:-claude}"
PERMISSION_MODE="${PERMISSION_MODE:-acceptEdits}"

# 에이전트가 프롬프트 없이 쓸 수 있어야 하는 도구다. 게이트(go·gofmt·make)와 커밋에
# 필요한 git 서브커맨드만 연다 — `git push`·`git reset`·`rm`은 일부러 빼 두었다.
# 되돌릴 수 없는 동작이 자동 루프 안에서 조용히 일어나지 않게 하는 것이 목적이다.
readonly ALLOWED_TOOLS=(
	"Bash(go:*)" "Bash(gofmt:*)" "Bash(make:*)"
	"Bash(git add:*)" "Bash(git commit:*)" "Bash(git status:*)"
	"Bash(git diff:*)" "Bash(git log:*)" "Bash(git show:*)"
	"Bash(grep:*)" "Bash(rg:*)" "Bash(find:*)" "Bash(ls:*)" "Bash(wc:*)"
	"Bash(cat:*)" "Bash(head:*)" "Bash(tail:*)" "Bash(sed:*)" "Bash(awk:*)"
	"Bash(mkdir:*)" "Bash(cp:*)" "Bash(diff:*)" "Bash(unzip:*)" "Bash(xmllint:*)"
)

log()  { printf '\033[36m▸\033[0m %s\n' "$*"; }
warn() { printf '\033[33m!\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# usage는 파일 맨 위 주석 블록을 그대로 도움말로 쓴다 (설명이 두 곳에 있으면 갈라진다).
usage() {
	awk 'NR>1 { if (!/^#/) exit; sub(/^# ?/, ""); print }' "${BASH_SOURCE[0]}"
	exit "${1:-0}"
}

# ── 준비 ─────────────────────────────────────────────────────────────────────

# preflight는 항목을 돌리기 전에 되돌리기 어려운 상태를 걸러낸다.
#
# 워킹 트리가 더러우면 시작하지 않는다: 이 스크립트는 "새 커밋이 생겼는가"로 항목의 성공을
# 판정하므로, 손으로 만든 변경이 섞여 있으면 에이전트의 커밋에 그것이 함께 실려 들어가고
# 어느 줄이 누구의 것인지 잃는다.
preflight() {
	command -v "$CLAUDE_BIN" >/dev/null 2>&1 || die "claude를 찾을 수 없다 (CLAUDE_BIN으로 지정한다)"
	git -C "$ROOT" rev-parse --git-dir >/dev/null 2>&1 || die "git 저장소가 아니다: $ROOT"
	[[ -f "$PREAMBLE" ]] || die "공통 프롬프트가 없다: $PREAMBLE"
	# DRY_RUN은 프롬프트만 찍고 끝나므로 트리 상태와 무관하다 (이 러너 자체를 커밋하기 전에도
	# 프롬프트를 읽어 볼 수 있어야 한다). 추적되지 않은 파일도 더러운 것으로 센다 —
	# 에이전트가 `git add -A`를 하면 그것까지 커밋에 실린다.
	if [[ "${DRY_RUN:-}" != 1 && -n "$(git -C "$ROOT" status --porcelain)" ]]; then
		git -C "$ROOT" status --short >&2
		die "워킹 트리가 깨끗해야 한다 — 커밋하거나 치워 두고 다시 돌린다"
	fi
	mkdir -p "$LOG_DIR"
}

# item_files는 iteration 디렉토리의 항목 프롬프트를 사전순으로 준다 (파일 이름의 번호가
# 곧 실행 순서다 — 항목은 앞 항목의 커밋 위에서 돈다).
item_files() {
	local iter="$1"; shift
	local dir="$ITER_DIR/$iter"
	[[ -d "$dir" ]] || die "iteration $iter의 항목이 없다: $dir"
	local file found=0
	for file in "$dir"/*.md; do
		[[ -e "$file" ]] || continue
		if (($# == 0)) || matches "$(basename "$file" .md)" "$@"; then
			printf '%s\n' "$file"
			found=1
		fi
	done
	((found)) || die "고른 항목이 없다 (필터: $*)"
}

# matches는 항목 이름이 필터 중 하나로 시작하는지 본다 (`06` → `06-fixtures`).
matches() {
	local name="$1"; shift
	local filter
	for filter in "$@"; do
		[[ "$name" == "$filter"* ]] && return 0
	done
	return 1
}

# done_marker는 NOTES.md가 이 항목을 완료로 적었는지 볼 때 쓰는 표시다. 상태를 별도 파일에
# 두지 않는 이유: 완료의 근거가 사람이 읽는 문서에 있어야 나중에 "이 항목이 왜 초록인가"를
# 커밋 로그와 노트만으로 되짚을 수 있다.
done_marker() { printf '<!-- item: %s/%s -->' "$1" "$2"; }

is_done() {
	[[ -f "$NOTES" ]] && grep -qF "$(done_marker "$1" "$2")" "$NOTES"
}

# ── 프롬프트 ─────────────────────────────────────────────────────────────────

render_prompt() {
	local iter="$1" slug="$2" file="$3"
	local body
	body="$(cat "$PREAMBLE"; printf '\n\n---\n\n'; cat "$file")"
	body="${body//\{\{ITERATION\}\}/$iter}"
	body="${body//\{\{ITEM\}\}/$slug}"
	body="${body//\{\{MARKER\}\}/$(done_marker "$iter" "$slug")}"
	printf '%s' "$body"
}

# ── 검증 ─────────────────────────────────────────────────────────────────────

# gates는 게이트를 순서대로 돌린다 — 싼 것부터. make test는 컨테이너를 띄우므로 마지막이다.
gates() {
	local out
	log "게이트: gofmt"
	out="$(cd "$ROOT" && gofmt -l .)" || die "gofmt 실패"
	[[ -z "$out" ]] || { printf '%s\n' "$out" >&2; return 1; }
	log "게이트: go vet"
	(cd "$ROOT" && go vet ./...) || return 1
	log "게이트: go test"
	(cd "$ROOT" && go test ./...) || return 1
	log "게이트: make test (컨테이너)"
	(cd "$ROOT" && make test) || return 1
}

# verify_commit은 항목이 남긴 커밋이 규약대로인지 본다: 새 커밋이 있고, 워킹 트리가
# 깨끗하고, NOTES.md 항목이 **그 커밋 안에** 들어 있다.
#
# 노트를 별도 커밋으로 받지 않는 이유: 코드와 판단 근거가 다른 커밋에 있으면 나중에
# `git show`로 한 항목을 통째로 읽을 수 없고, 그렇게 갈라진 노트는 곧 갱신되지 않는다.
verify_commit() {
	local iter="$1" slug="$2" before="$3"
	local after; after="$(git -C "$ROOT" rev-parse HEAD)"
	[[ "$after" != "$before" ]] || die "커밋이 생기지 않았다 — 항목이 끝나지 않았거나 커밋을 잊었다"

	if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
		git -C "$ROOT" status --short >&2
		die "커밋되지 않은 변경이 남았다 — 항목의 산출물 전체가 한 커밋에 들어가야 한다"
	fi

	local range="$before..$after"
	local count; count="$(git -C "$ROOT" rev-list --count "$range")"
	((count == 1)) || warn "커밋이 $count개다 (항목당 하나를 기대한다)"

	git -C "$ROOT" diff --name-only "$range" | grep -qx 'NOTES.md' \
		|| die "NOTES.md가 커밋에 없다 — 리뷰 노트는 코드와 같은 커밋에 들어간다"
	is_done "$iter" "$slug" \
		|| die "NOTES.md에 항목 표시가 없다: $(done_marker "$iter" "$slug")"

	log "커밋 $count개:"
	git -C "$ROOT" log --oneline "$range" | sed 's/^/    /'
}

# ── 실행 ─────────────────────────────────────────────────────────────────────

run_item() {
	local iter="$1" file="$2"
	local slug; slug="$(basename "$file" .md)"
	local prompt; prompt="$(render_prompt "$iter" "$slug" "$file")"

	if [[ "${DRY_RUN:-}" == 1 ]]; then
		printf '\n===== %s/%s =====\n%s\n' "$iter" "$slug" "$prompt"
		return 0
	fi

	local logfile="$LOG_DIR/$iter-$slug.log"
	local before; before="$(git -C "$ROOT" rev-parse HEAD)"
	log "항목 $iter/$slug 시작 (로그: ${logfile#$ROOT/})"

	local args=(--print --permission-mode "$PERMISSION_MODE" --allowedTools "${ALLOWED_TOOLS[@]}")
	[[ -n "${MODEL:-}" ]] && args+=(--model "$MODEL")

	# STREAM=1은 진행 상황을 흘린다. 기본이 아닌 이유는 jq 의존과 로그 부피이고, 켜는
	# 이유는 항목 하나가 길어서(컨테이너 테스트까지 돈다) 끝날 때까지 아무 출력이 없으면
	# 멈춘 것과 구별되지 않기 때문이다.
	if [[ "${STREAM:-}" == 1 ]] && command -v jq >/dev/null 2>&1; then
		(cd "$ROOT" && "$CLAUDE_BIN" "${args[@]}" --output-format stream-json --verbose "$prompt") \
			| tee "$logfile" \
			| jq -rj --unbuffered 'select(.type=="assistant") | .message.content[]? | select(.type=="text") | .text' \
			|| die "claude가 실패했다 (로그: $logfile)"
		printf '\n'
	else
		[[ "${STREAM:-}" == 1 ]] && warn "jq가 없어 스트리밍 없이 돈다"
		(cd "$ROOT" && "$CLAUDE_BIN" "${args[@]}" "$prompt") | tee "$logfile" \
			|| die "claude가 실패했다 (로그: $logfile)"
	fi

	verify_commit "$iter" "$slug" "$before"

	if [[ "${SKIP_GATES:-}" == 1 ]]; then
		warn "게이트를 건너뛴다 (SKIP_GATES=1)"
	elif ! gates; then
		die "게이트가 깨졌다 — 커밋 $(git -C "$ROOT" rev-parse --short HEAD)를 손으로 고치거나 되돌린다
    되돌리기: git reset --soft $before && git restore --staged ."
	fi
	log "항목 $iter/$slug 완료"
}

list_items() {
	local iter="$1"; shift
	local file slug
	while IFS= read -r file; do
		slug="$(basename "$file" .md)"
		if is_done "$iter" "$slug"; then
			printf '  \033[32m✓\033[0m %s\n' "$slug"
		else
			printf '  \033[90m·\033[0m %s\n' "$slug"
		fi
	done < <(item_files "$iter" "$@")
}

main() {
	local list=0
	while (($#)); do
		case "$1" in
			-h|--help) usage 0 ;;
			-l|--list) list=1; shift ;;
			-*) die "알 수 없는 옵션: $1" ;;
			*) break ;;
		esac
	done
	(($# >= 1)) || usage 1

	# iteration은 한 자리로 주어도 디렉토리 이름(두 자리)으로 맞춘다: `./run.sh 1` = `01`.
	# 숫자가 아니면 그대로 둔다 — printf에 넘기면 `00abc` 같은 이름으로 조용히 뒤틀린다.
	local iter="$1"
	[[ "$iter" =~ ^[0-9]+$ ]] && iter="$(printf '%02d' "$iter")"
	shift

	if ((list)); then
		list_items "$iter" "$@"
		return 0
	fi

	preflight

	local file slug ran=0 skipped=0
	while IFS= read -r file; do
		slug="$(basename "$file" .md)"
		if is_done "$iter" "$slug" && [[ "${FORCE:-}" != 1 ]]; then
			log "항목 $iter/$slug 건너뜀 (NOTES.md에 완료로 적혀 있다 — FORCE=1로 다시 돌린다)"
			((++skipped))
			continue
		fi
		run_item "$iter" "$file"
		((++ran))
	done < <(item_files "$iter" "$@")

	log "iteration $iter: 항목 $ran개 실행, $skipped개 건너뜀"
	if ((ran)); then
		printf '\n다음: `make review`로 갤러리를 보고, macOS에서 deck.pptx를 Keynote/PowerPoint로 직접 연다\n'
		printf '(TASKS.md 반복 절차 ③ — 사람이 눈으로 보는 단계는 자동화하지 않는다)\n'
	fi
}

main "$@"
