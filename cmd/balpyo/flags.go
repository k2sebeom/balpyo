package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// reportedError는 이미 사람에게 보인 에러다. FlagSet(ContinueOnError)은 모르는 플래그나 빠진 값을
// 사용법과 함께 스스로 찍으므로, main이 같은 문구를 한 번 더 찍지 않게 표시해 둔다.
type reportedError struct{ err error }

func (e *reportedError) Error() string { return e.err.Error() }
func (e *reportedError) Unwrap() error { return e.err }

// parseFlags는 fs.Parse를 감싼다. `-h`/`--help`는 flag.ErrHelp로 올려 cli가 성공으로 끝내게 하고,
// 그 밖의 파싱 에러는 FlagSet이 이미 찍었다는 표시를 붙인다.
func parseFlags(fs *flag.FlagSet, args []string) error {
	err := fs.Parse(args)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, flag.ErrHelp):
		return flag.ErrHelp
	}
	return &reportedError{err}
}

// parseInterleaved는 플래그와 위치 인자가 섞인 인자를 읽는다 (`install <URL> --force`,
// `init --author A my-talk --theme ink`). flag 패키지는 첫 위치 인자에서 멈추므로 남은 것을 다시 읽는다.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := parseFlags(fs, args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

// cli는 run을 돌려 종료 코드를 정한다 — main이 os.Exit에 넘기는 값을 테스트가 그대로 본다.
// 도움말 요청은 실패가 아니므로 0이고, 이미 찍힌 에러는 다시 찍지 않는다.
func cli(args []string, stdout, stderr io.Writer) int {
	err := run(args, stdout, stderr)
	var reported *reportedError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return 0
	case errors.As(err, &reported):
		return 1
	}
	fmt.Fprintln(stderr, err)
	return 1
}
