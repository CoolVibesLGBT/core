#!/bin/sh

set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
binary="$project_dir/build/core"
source_stamp="$project_dir/build/core.sources"
needs_build=0

# A full Xcode installation may be selected before its license is accepted,
# even though the standalone Command Line Tools are already usable. Go only
# needs those tools for this project's CGO/WebP build.
if [ "$(uname -s)" = "Darwin" ] && [ -x /Library/Developer/CommandLineTools/usr/bin/clang ]; then
	export DEVELOPER_DIR=/Library/Developer/CommandLineTools
fi

source_fingerprint=$(
	{
		go version
		go env GOOS GOARCH CGO_ENABLED
		find "$project_dir" \
			\( -path "$project_dir/.git" -o -path "$project_dir/build" -o -path "$project_dir/tmp" -o -path "$project_dir/vendor" \) -prune -o \
			-type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -print0 |
			LC_ALL=C sort -z |
			xargs -0 cksum
	} | cksum | awk '{print $1 ":" $2}'
)

if [ ! -x "$binary" ] || [ ! -f "$source_stamp" ]; then
	needs_build=1
elif [ "$(sed -n '1p' "$source_stamp")" != "$source_fingerprint" ]; then
	needs_build=1
fi

if [ "$needs_build" -eq 1 ]; then
	mkdir -p "$project_dir/build"
	printf '%s\n' 'Building core...'
	(
		cd "$project_dir"
		go build -trimpath -ldflags="-s -w" -o "$binary" .
	)
	printf '%s\n' "$source_fingerprint" >"$source_stamp"
fi

cd "$project_dir"
exec "$binary" "$@"
