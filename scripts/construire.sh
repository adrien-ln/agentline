#!/bin/bash
# Construit les archives d'« agentline » pour Windows, Linux et macOS, leurs sommes SHA-256, et les manifestes
# Scoop et Homebrew qui pointent vers elles. Usage : bash scripts/construire.sh <version> [url de base]
#   <version>      par exemple 1.0.0 (sans « v »)
#   [url de base]  où les fichiers seront publiés ; par défaut la version GitHub du tag v<version>
# Résultat dans dist/<version>/ : pas de binaire nu (un .exe lancé d'un double-clic s'ouvre dans la vieille
# console, sans Nerd Font) ; .tar.gz pour Linux et macOS (Homebrew, installer-agentline.sh), .zip pour
# Windows (Scoop). Lancé par .github/workflows/release.yml sur chaque tag v*, et à la main pour vérifier.
# Il faut zip (ou 7z) pour les archives Windows.
set -euo pipefail

version=${1:?usage: bash scripts/construire.sh <version> [base url]}
depot=$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)
base=${2:-https://github.com/adrien-ln/agentline/releases/download/v$version}
dist=$depot/dist/$version
rm -rf "$dist" && mkdir -p "$dist"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# zipper archive dossier : le contenu du dossier, sans le dossier lui-même
zipper() {
  if command -v zip >/dev/null; then (cd "$2" && zip -qX "$1" ./*)
  elif command -v 7z >/dev/null; then (cd "$2" && 7z a -bso0 -bsp0 "$1" ./* >/dev/null)
  else echo "zip or 7z not found: cannot build the Windows archives" >&2; exit 1; fi
}

cibles=(windows/amd64 windows/arm64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64)
for c in "${cibles[@]}"; do
  os=${c%/*} arch=${c#*/} exe=agentline
  [[ $os == windows ]] && exe=agentline.exe
  d=$tmp/${os}_$arch
  mkdir -p "$d"
  (cd "$depot" && CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
     -ldflags "-s -w -X main.version=$version" -o "$d/$exe" ./cmd/agentline)
  cp "$depot/LICENSE" "$depot/README.md" "$d/"
  if [[ $os == windows ]]; then
    f=agentline_${version}_${os}_$arch.zip
    zipper "$dist/$f" "$d"
  else
    f=agentline_${version}_${os}_$arch.tar.gz
    tar -C "$d" -czf "$dist/$f" --owner=0 --group=0 agentline LICENSE README.md 2>/dev/null ||
      tar -C "$d" -czf "$dist/$f" agentline LICENSE README.md # bsdtar (macOS) : sans --owner
  fi
  printf '%-40s %s\n' "$f" "$(wc -c < "$dist/$f") bytes"
done

(cd "$dist" && sha256sum agentline_* > SHA256SUMS)
somme() { awk -v f="agentline_${version}_$1" '{ n = $2; sub(/^\*/, "", n) } n == f { print $1 }' "$dist/SHA256SUMS"; } # « *nom » : mode binaire (Git Bash)

# Manifestes à partir des modèles de packaging/ : @VERSION@, @BASE@ et @SHA_<os>_<arch>@
remplir() {
  sed -e "s|@VERSION@|$version|g" -e "s|@BASE@|$base|g" \
      -e "s|@SHA_WINDOWS_AMD64@|$(somme windows_amd64.zip)|g" -e "s|@SHA_WINDOWS_ARM64@|$(somme windows_arm64.zip)|g" \
      -e "s|@SHA_LINUX_AMD64@|$(somme linux_amd64.tar.gz)|g" -e "s|@SHA_LINUX_ARM64@|$(somme linux_arm64.tar.gz)|g" \
      -e "s|@SHA_DARWIN_AMD64@|$(somme darwin_amd64.tar.gz)|g" -e "s|@SHA_DARWIN_ARM64@|$(somme darwin_arm64.tar.gz)|g" "$1"
}
remplir "$depot/packaging/scoop/agentline.json.modele" > "$dist/agentline.json"
remplir "$depot/packaging/homebrew/agentline.rb.modele" > "$dist/agentline.rb"
grep -qE '@[A-Z_]+@|""' "$dist/agentline.json" "$dist/agentline.rb" && { echo "incomplete manifest (missing field or checksum)" >&2; exit 1; }
echo "dist/$version: $(ls "$dist" | wc -l) files (archives, SHA256SUMS, agentline.json for Scoop, agentline.rb for Homebrew)"
