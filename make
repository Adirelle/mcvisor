#!/usr/bin/env bash
set -eu -o pipefail

: ${BIN_SUFFIX:=}
SRC="./cmd/... ./pkg/..."
export GOARCH=amd64

target() {
	echo "./$(basename $1)${BIN_SUFFIX}_${GOARCH}_${GOOS}$(env GOOS="${GOOS}" go env GOEXE)"
}

cmd_lint() {
	golangci-lint run ${SRC}
}

cmd_lintfix() {
	golangci-lint run --fix ${SRC}
}

cmd_test() {
	go test ./{cmd,pkg}/...
}

cmd_build() {
	for CMD in ./cmd/*; do
		for GOOS in windows linux; do
			ENV="env GOOS=${GOOS}"
			TARGET="$(target ${CMD})"
			echo "$TARGET"
			$ENV go build -v -o "${TARGET}" "${CMD}"
			sha512sum "${TARGET}" >"${TARGET}.sha512"
		done
	done
}

cmd_clean() {
	for CMD in ./cmd/*; do
		rm -fv "$(basename ${CMD})_"*
	done
}

case "${1:-all}" in
	lint|lintfix|test|build|clean) cmd_${1} ;;
	all)
		cmd_clean
		cmd_lint
		cmd_build
		cmd_test
	;;
esac
