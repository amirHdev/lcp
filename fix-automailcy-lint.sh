#!/usr/bin/env bash
set -euo pipefail

CONFIG=".golangci.yml"
[ -f ".golangci.yaml" ] && CONFIG=".golangci.yaml"

cp "$CONFIG" "$CONFIG.bak2"

python3 - <<'PY'
from pathlib import Path

p = Path(".golangci.yml")
if not p.exists():
    p = Path(".golangci.yaml")

text = p.read_text()

# Remove duplicate block added by previous script
marker = "\n# Added by fix"
if marker in text:
    text = text[:text.index(marker)].rstrip() + "\n"

p.write_text(text)
print(f"Cleaned {p}")
PY

gofumpt -w .
golangci-lint run
