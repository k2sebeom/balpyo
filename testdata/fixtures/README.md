# 픽스처

픽스처 하나(`<NN>_<이름>/`)는 **완전한 balpyo 프로젝트**다. 그래서 사람이 그 디렉토리를
`balpyo build --project testdata/fixtures/00_empty --target html,pptx`로 직접 빌드해
결과를 볼 수 있다.

```
00_empty/
├── balpyo.json          # 덱 설정 (없으면 기본값)
├── slides/*.slide       # 입력
└── golden/              # 산출물 스냅샷 (사람이 커밋한다)
    ├── index.html       #   웹 타깃
    ├── diagnostics.txt  #   경고가 있을 때만
    └── pptx/**          #   PPTX 파트별 XML (zip을 푼 것)
```

규칙:

- 접두 숫자는 iteration 순서이자 스위트 실행 순서다.
- `_`로 시작하는 디렉토리는 수집되지 않는다 (작업 중 픽스처).
- 골든은 손으로 쓰지 않고 `make golden`으로 갱신한다. 골든 변경은 구현 변경과 같은
  커밋에 담아 "무엇이 왜 움직였는지"를 기록에 남긴다.
- 산출물은 임시 디렉토리에 쓰인다 — 픽스처 디렉토리는 입력과 골든만 담는다.

자세한 규칙은 [`docs/internals/testing.md`](../../docs/internals/testing.md)에 있다.
