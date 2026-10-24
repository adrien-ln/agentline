#!/bin/bash
# Publie les manifestes d'une version dans le bucket Scoop (adrien-ln/scoop-agentline, bucket/agentline.json)
# et le tap Homebrew (adrien-ln/homebrew-agentline, Formula/agentline.rb). Lancé par
# .github/workflows/release.yml après la version GitHub, avec un jeton qui peut écrire dans ces deux dépôts.
# Usage : PACKAGING_TOKEN=… bash scripts/publier-manifestes.sh <version>
# Sans jeton : un avertissement, rien n'est publié (les manifestes restent joints à la version GitHub).
set -euo pipefail

version=${1:?usage : bash scripts/publier-manifestes.sh <version>}
dist=$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)/dist/$version
if [[ -z ${PACKAGING_TOKEN:-} ]]; then
  echo "::warning::PACKAGING_TOKEN is not set: update the Scoop bucket and the Homebrew tap by hand"
  exit 0
fi

publier() { # dépôt fichier chemin-dans-le-dépôt
  local d
  d=$(mktemp -d)
  git clone --quiet --depth 1 "https://x-access-token:$PACKAGING_TOKEN@github.com/adrien-ln/$1.git" "$d"
  git -C "$d" rev-parse --verify --quiet HEAD >/dev/null || git -C "$d" symbolic-ref HEAD refs/heads/main # dépôt vide
  mkdir -p "$d/$(dirname "$3")"
  cp "$2" "$d/$3"
  git -C "$d" add "$3"
  if git -C "$d" diff --cached --quiet; then
    echo "$1: $3 already up to date"
    return
  fi
  git -C "$d" -c user.name="github-actions[bot]" -c user.email="41898282+github-actions[bot]@users.noreply.github.com" \
    commit --quiet -m "chore: update agentline to v$version"
  git -C "$d" push --quiet origin HEAD
  echo "$1: $3 updated to v$version"
}

publier scoop-agentline "$dist/agentline.json" bucket/agentline.json
publier homebrew-agentline "$dist/agentline.rb" Formula/agentline.rb
