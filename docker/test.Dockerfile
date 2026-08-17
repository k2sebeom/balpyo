# balpyo 테스트 이미지.
#
# 픽셀 비교와 텍스트 메트릭 테스트는 렌더 환경이 결정적일 때만 의미가 있다. 로컬 macOS와
# CI 리눅스가 같은 결과를 내야 하므로 테스트는 항상 이 이미지 안에서 돈다 (`make test`).
#
# 결정성의 근거:
#   - 베이스 이미지를 패치 버전까지 고정한다 (golang:1.26.6-bookworm = Debian bookworm).
#     bookworm의 apt 저장소는 릴리스 내에서 패키지 버전을 고정하므로 LibreOffice와
#     Chromium의 메이저·마이너가 고정된다.
#   - 실제로 설치된 버전을 이미지 안에 기록하고(/etc/balpyo-versions.txt),
#     docker/versions.env의 기대값과 대조하는 테스트를 둔다. 보안 패치로 버전이 움직이면
#     골든이 조용히 깨지는 대신 그 테스트가 먼저 실패한다.
#   - 폰트는 Noto만 설치한다. 텍스트 메트릭이 재현 가능하려면 폰트 집합이 결정적이어야 한다.
FROM golang:1.26.6-bookworm

# LibreOffice는 PPTX 열림 테스트와 PPTX 렌더(pdf 경유)를,
# Chromium은 지오메트리 패스와 웹 스크린샷을, poppler-utils는 pdf → png를 담당한다.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libreoffice-impress \
        chromium \
        poppler-utils \
        fonts-noto-core \
        fonts-noto-cjk \
        fontconfig \
    && rm -rf /var/lib/apt/lists/*

# 렌더 결과에 영향을 주는 폰트 설정을 이미지가 소유한다.
# 힌팅은 픽셀 그리드에 글리프를 맞추느라 글자 폭을 바꾼다 — 끄면 브라우저와 LibreOffice의
# 텍스트 폭이 서로, 그리고 실행마다 더 잘 맞는다.
COPY docker/fonts.conf /etc/fonts/local.conf
RUN fc-cache -f

# 컨테이너 안에서는 root로 돌기 때문에 Chromium 샌드박스를 쓸 수 없다.
# 테스트 하네스는 이 값을 읽어 플래그를 붙인다 (internal/testkit/chromium.go).
ENV BALPYO_CHROMIUM=/usr/bin/chromium \
    BALPYO_CHROMIUM_NO_SANDBOX=1 \
    BALPYO_SOFFICE=/usr/bin/soffice \
    BALPYO_PDFTOPPM=/usr/bin/pdftoppm

# 이미지 안에서 도는 테스트는 외부 도구가 없으면 건너뛰지 말고 실패해야 한다.
ENV BALPYO_TEST_STRICT=1

# LibreOffice는 쓰기 가능한 HOME에 사용자 프로필을 만든다.
ENV HOME=/tmp

# 설치된 버전 기록. versions.env 대조 테스트와 사람의 디버깅이 같은 파일을 본다.
RUN { \
      echo "BALPYO_IMAGE_BASE=golang:1.26.6-bookworm"; \
      echo "BALPYO_DEBIAN=$(cat /etc/debian_version)"; \
      echo "BALPYO_GO=$(go version | cut -d' ' -f3)"; \
      echo "BALPYO_SOFFICE_VERSION=$(soffice --version 2>/dev/null | head -1)"; \
      echo "BALPYO_CHROMIUM_VERSION=$(chromium --version 2>/dev/null | head -1)"; \
      echo "BALPYO_PDFTOPPM_VERSION=$(pdftoppm -v 2>&1 | head -1)"; \
      echo "BALPYO_FONT_SANS=$(fc-match 'Noto Sans' -f '%{file}')"; \
      echo "BALPYO_FONT_CJK=$(fc-match 'Noto Sans CJK KR' -f '%{file}')"; \
    } > /etc/balpyo-versions.txt

WORKDIR /src
