package themestore

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DefaultBaseURL은 GitHub의 tarball 주소다. `git`을 요구하지 않으려고 저장소를 압축 하나로 받는다.
const DefaultBaseURL = "https://codeload.github.com"

// 받는 압축의 한도다. 테마는 JSON · CSS · 그림 몇 장이다 — 한도는 실수로 거대한 저장소를 가리켰을 때
// 디스크를 채우지 않기 위한 것이다.
const (
	maxArchiveBytes = 512 << 20
	maxArchiveFiles = 50000
)

// Client는 저장소를 받는 쪽이다. 0값은 GitHub에서 기본 http.Client로 받는다.
type Client struct {
	// BaseURL이 비면 DefaultBaseURL이다 (테스트가 로컬 서버를 가리킨다).
	BaseURL string
	HTTP    *http.Client
}

// snapshot은 받은 저장소 하나다 — 디스크에 푼 디렉토리와, 압축이 알려 준 것들.
type snapshot struct {
	dir string
	// commit은 압축의 pax 전역 헤더가 담은 커밋이다 (GitHub tarball이 적는다 — 없으면 빈 문자열).
	commit string
	// links는 풀지 않은 심볼릭 · 하드 링크의 저장소 상대 경로다. 테마 안에 있으면 설치가 거부한다.
	links []string
}

// fetch는 src의 저장소를 dest 아래에 푼다. 파일은 정규 파일과 디렉토리만 만든다.
func (c Client) fetch(ctx context.Context, src Source, dest string) (snapshot, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	ref := src.Ref
	if ref == "" {
		ref = "HEAD"
	}
	u := strings.TrimRight(base, "/") + "/" + url.PathEscape(src.Owner) + "/" + url.PathEscape(src.Repo) +
		"/tar.gz/" + escapeRef(ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return snapshot{}, err
	}
	httpc := c.HTTP
	if httpc == nil {
		httpc = http.DefaultClient
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return snapshot{}, fmt.Errorf("cannot download %s: %w", src.RepoID(), err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return snapshot{}, fmt.Errorf("%s@%s not found on GitHub — check the repository name and ref (private repositories are not supported)",
			src.RepoID(), ref)
	case resp.StatusCode != http.StatusOK:
		return snapshot{}, fmt.Errorf("cannot download %s: HTTP %s", src.RepoID(), resp.Status)
	}
	snap, err := extract(resp.Body, dest)
	if err != nil {
		return snapshot{}, fmt.Errorf("cannot extract %s: %w", src.RepoID(), err)
	}
	return snap, nil
}

// escapeRef는 ref의 경로 조각을 하나씩 이스케이프한다 (`feature/x`의 슬래시는 남긴다).
func escapeRef(ref string) string {
	parts := strings.Split(ref, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// extract는 gzip tar를 dest에 푼다. GitHub tarball은 최상위 디렉토리 하나(`<repo>-<ref>/`) 아래에
// 저장소를 담으므로 첫 경로 조각을 떼어 낸다.
func extract(r io.Reader, dest string) (snapshot, error) {
	snap := snapshot{dir: dest}
	gz, err := gzip.NewReader(r)
	if err != nil {
		return snap, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	files := 0
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return snap, err
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			snap.commit = hdr.PAXRecords["comment"]
			continue
		}
		_, rel, ok := strings.Cut(hdr.Name, "/")
		if !ok || strings.Trim(rel, "/") == "" {
			continue
		}
		rel = path.Clean(rel)
		if !fs.ValidPath(rel) {
			return snap, fmt.Errorf("archive path %q points outside the repository", hdr.Name)
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return snap, err
			}
		case tar.TypeReg:
			files++
			total += hdr.Size
			if files > maxArchiveFiles || total > maxArchiveBytes {
				return snap, fmt.Errorf("repository is too large (over %d files · %d bytes)", maxArchiveFiles, maxArchiveBytes)
			}
			if err := writeFile(target, tr, hdr.Size); err != nil {
				return snap, err
			}
		case tar.TypeSymlink, tar.TypeLink:
			snap.links = append(snap.links, rel)
		}
	}
	return snap, nil
}

func writeFile(target string, r io.Reader, size int64) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.CopyN(f, r, size); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
