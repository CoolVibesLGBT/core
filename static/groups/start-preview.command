#!/bin/zsh
cd -- "$(dirname -- "$0")"
exec python3 scripts/serve.py
