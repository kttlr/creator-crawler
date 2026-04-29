set shell := ["bash", "-uc"]

dev:
	#!/usr/bin/env bash
	set -euo pipefail
	mkdir -p .dev-bin
	GOBIN="$PWD/.dev-bin" go install github.com/a-h/templ/cmd/templ@v0.3.1001
	GOBIN="$PWD/.dev-bin" go install github.com/air-verse/air@v1.63.0

	cleanup() {
		kill "$templ_pid" "$css_pid" "$air_pid" 2>/dev/null || true
	}
	trap cleanup INT TERM EXIT

	.dev-bin/templ generate -watch -watch-pattern '(.+\.templ$)' -cmd 'pnpm build:css' &
	templ_pid=$!

	pnpm watch:css &
	css_pid=$!

	.dev-bin/air -c .air.toml &
	air_pid=$!

	printf 'Dev server: http://localhost:8090\n'
	printf 'App server: http://localhost:8080\n'
	wait
