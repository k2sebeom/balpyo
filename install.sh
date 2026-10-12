#!/bin/sh
# balpyo 설치 스크립트 — GitHub Releases의 아카이브를 받아 체크섬을 맞춘 뒤 설치한다.
#
#   curl -fsSL https://raw.githubusercontent.com/k2sebeom/balpyo/main/install.sh | sh
#
# 환경 변수:
#   BALPYO_VERSION       설치할 태그 (예: v0.1.0). 없으면 최신 릴리스.
#   BALPYO_INSTALL_DIR   설치할 디렉토리. 기본은 $HOME/.local/bin.
#   BALPYO_DOWNLOAD_URL  릴리스 기준 URL. 기본은 https://github.com/k2sebeom/balpyo/releases.
#                        그 아래의 `download/<태그>/<파일>`과 `latest/download/<파일>`을 받는다.
#
# POSIX sh만 쓴다 (`curl … | sh`의 sh는 dash일 수도, macOS의 sh일 수도 있다). jq나 GitHub API를
# 쓰지 않는다: 최신 버전은 `latest/download/SHA256SUMS`에 적힌 아카이브 이름에서 읽는다 —
# API는 비인증 호출 수에 상한이 있고, 체크섬 파일은 어차피 받아야 한다.
set -eu

BALPYO_RELEASES="${BALPYO_DOWNLOAD_URL:-https://github.com/k2sebeom/balpyo/releases}"
BALPYO_RELEASES="${BALPYO_RELEASES%/}"
INSTALL_DIR="${BALPYO_INSTALL_DIR:-${HOME}/.local/bin}"

say() { printf 'balpyo-install: %s\n' "$*" >&2; }
warn() { printf 'balpyo-install: warning: %s\n' "$*" >&2; }
die() {
	printf 'balpyo-install: error: %s\n' "$*" >&2
	exit 1
}

# fetch URL 파일 — curl이 없으면 wget을 쓴다. 둘 다 리다이렉트를 따라가고(GitHub의 다운로드는
# 리다이렉트다) HTTP 오류에서 실패한다.
fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --retry 3 -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		die "neither curl nor wget is installed"
	fi
}

# sha256 파일 — 리눅스는 sha256sum, macOS는 shasum이다. 해시를 낼 도구가 없으면 검증을 건너뛰지
# 않고 멈춘다: 확인하지 않은 바이너리를 설치하는 것보다 설치하지 않는 편이 낫다.
sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{print $1}'
	elif command -v openssl >/dev/null 2>&1; then
		openssl dgst -sha256 "$1" | awk '{print $NF}'
	else
		die "no SHA-256 tool found (sha256sum, shasum or openssl) — refusing to install an unverified binary"
	fi
}

detect_os() {
	case "$(uname -s)" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	MINGW* | MSYS* | CYGWIN* | Windows_NT)
		die "this installer supports Linux and macOS only — on Windows, download the archive from ${BALPYO_RELEASES}"
		;;
	*) die "unsupported operating system: $(uname -s) (supported: Linux, macOS)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
	x86_64 | amd64) echo amd64 ;;
	aarch64 | arm64) echo arm64 ;;
	*) die "unsupported architecture: $(uname -m) (supported: x86_64/amd64, aarch64/arm64)" ;;
	esac
}

# 크롬 탐색은 빌드가 찾는 순서와 같다: BALPYO_CHROMIUM → PATH의 chromium · chromium-browser ·
# google-chrome → macOS의 알려진 설치 경로. 없어도 설치는 끝낸다 — 웹 타깃은 브라우저 없이 된다.
has_chromium() {
	if [ -n "${BALPYO_CHROMIUM:-}" ]; then
		[ -f "$BALPYO_CHROMIUM" ] && [ -x "$BALPYO_CHROMIUM" ]
		return
	fi
	for candidate in chromium chromium-browser google-chrome; do
		if command -v "$candidate" >/dev/null 2>&1; then
			return 0
		fi
	done
	for candidate in \
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
		"/Applications/Chromium.app/Contents/MacOS/Chromium"; do
		if [ -x "$candidate" ]; then
			return 0
		fi
	done
	return 1
}

main() {
	os="$(detect_os)"
	arch="$(detect_arch)"
	suffix="_${os}_${arch}.tar.gz"

	tmp="$(mktemp -d 2>/dev/null || mktemp -d -t balpyo-install)"
	trap 'rm -rf "$tmp"' EXIT
	trap 'exit 1' HUP INT TERM

	if [ -n "${BALPYO_VERSION:-}" ]; then
		version="$BALPYO_VERSION"
		archive="balpyo_${version}${suffix}"
		say "fetching checksums for ${version}"
		fetch "${BALPYO_RELEASES}/download/${version}/SHA256SUMS" "$tmp/SHA256SUMS" ||
			die "could not download SHA256SUMS for ${version} — does the release exist?"
	else
		say "resolving the latest release"
		fetch "${BALPYO_RELEASES}/latest/download/SHA256SUMS" "$tmp/SHA256SUMS" ||
			die "could not download the latest release's SHA256SUMS — set BALPYO_VERSION to pick a release"
		archive="$(awk -v s="$suffix" '{
			name = $2; sub(/^\*/, "", name)
			if (substr(name, 1, 7) == "balpyo_" && length(name) > length(s) &&
				substr(name, length(name) - length(s) + 1) == s) { print name; exit }
		}' "$tmp/SHA256SUMS")"
		[ -n "$archive" ] || die "the latest release has no archive for ${os}/${arch}"
		version="${archive#balpyo_}"
		version="${version%"$suffix"}"
	fi

	expected="$(awk -v n="$archive" '{ name = $2; sub(/^\*/, "", name) } name == n { print $1; exit }' "$tmp/SHA256SUMS")"
	[ -n "$expected" ] || die "SHA256SUMS of ${version} has no entry for ${archive} (${os}/${arch} may not be released)"

	# 최신을 고른 경우에도 아카이브는 **태그가 박힌 URL**에서 받는다 — 두 요청 사이에 새 릴리스가
	# 나와도 체크섬과 아카이브가 다른 릴리스에서 오지 않는다.
	say "downloading ${archive}"
	fetch "${BALPYO_RELEASES}/download/${version}/${archive}" "$tmp/$archive" ||
		die "could not download ${archive}"

	actual="$(sha256 "$tmp/$archive")"
	if [ "$actual" != "$expected" ]; then
		die "checksum mismatch for ${archive} (expected ${expected}, got ${actual}) — nothing was installed"
	fi
	say "checksum verified"

	mkdir "$tmp/extract"
	tar -xzf "$tmp/$archive" -C "$tmp/extract" balpyo || die "could not extract ${archive}"
	[ -f "$tmp/extract/balpyo" ] || die "${archive} does not contain a balpyo binary"

	mkdir -p "$INSTALL_DIR" || die "could not create ${INSTALL_DIR}"
	# 같은 디렉토리의 임시 이름에 놓은 뒤 mv한다 — 실행 중인 balpyo를 덮어도 반쯤 쓴 파일이
	# 보이지 않는다.
	staged="$INSTALL_DIR/.balpyo.install.$$"
	cp "$tmp/extract/balpyo" "$staged" || die "could not write to ${INSTALL_DIR}"
	chmod 755 "$staged"
	mv -f "$staged" "$INSTALL_DIR/balpyo" || {
		rm -f "$staged"
		die "could not install into ${INSTALL_DIR}"
	}

	say "installed ${INSTALL_DIR}/balpyo"
	"$INSTALL_DIR/balpyo" version

	case ":${PATH}:" in
	*":${INSTALL_DIR}:"* | *":${INSTALL_DIR%/}:"*) ;;
	*) warn "${INSTALL_DIR} is not on your PATH — add it, e.g. export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
	esac

	if ! has_chromium; then
		warn "no Chromium/Chrome found — the PPTX target and --check-layout need one (set BALPYO_CHROMIUM to its path); web builds work without it"
	fi
}

main "$@"
