#!/usr/bin/env sh

set -eu

cd "$(dirname "$0")/.."

if [ ! -f apps/web/out/index.html ]; then
  echo "apps/web/out/index.html is missing; run 'pnpm --filter @stackit/web build' first." >&2
  exit 1
fi

mkdir -p apps/server/static
find apps/server/static -mindepth 1 -maxdepth 1 ! -name .gitignore -exec rm -rf {} +
cp -R apps/web/out/. apps/server/static/
