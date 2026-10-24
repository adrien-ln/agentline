#!/bin/bash
# Construit les binaires d'« agentline » pour Windows, Linux et macOS, leurs sommes SHA-256, et les manifestes
# Scoop et Homebrew qui pointent vers eux. Usage : bash scripts/construire.sh <version> [url de base]
#   <version>      par exemple 1.0.0 (sans « v »)
#   [url de base]  où les fichiers seront publiés ; par défaut le registre de paquets du projet GitLab
# Résultat dans dist/<version>/. Lancé par la CI sur chaque tag v*, et à la main pour vérifier.
set -euo pipefail

version=${1:?usage : bash scripts/construire.sh <version> [url de base]}
depot=$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)
projet=${CI_PROJECT_ID:-adrien-ln%2Fstatusline}
base=${2:-https://gitlab.com/api/v4/projects/$projet/packages/generic/agentline/$version}
dist=$depot/dist/$version
rm -rf "$dist" && mkdir -p "$dist"

cibles=(windows/amd64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64)
for c in "${cibles[@]}"; do
  os=${c%/*} arch=${c#*/} ext=
  [[ $os == windows ]] && ext=.exe
  f=agentline_${version}_${os}_${arch}$ext
  (cd "$depot" && CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
     -ldflags "-s -w -X main.version=$version" -o "$dist/$f" ./cmd/agentline)
  printf '%-36s %s\n' "$f" "$(wc -c < "$dist/$f") octets"
done

(cd "$dist" && sha256sum agentline_* > SHA256SUMS)
somme() { awk -v f="agentline_${version}_$1" '{ n = $2; sub(/^\*/, "", n) } n == f { print $1 }' "$dist/SHA256SUMS"; } # « *nom » : mode binaire (Git Bash)

# Manifestes à partir des modèles de packaging/ : @VERSION@, @BASE@ et @SHA_<os>_<arch>@
remplir() {
  sed -e "s|@VERSION@|$version|g" -e "s|@BASE@|$base|g" \
      -e "s|@SHA_WINDOWS_AMD64@|$(somme windows_amd64.exe)|g" \
      -e "s|@SHA_LINUX_AMD64@|$(somme linux_amd64)|g" -e "s|@SHA_LINUX_ARM64@|$(somme linux_arm64)|g" \
      -e "s|@SHA_DARWIN_AMD64@|$(somme darwin_amd64)|g" -e "s|@SHA_DARWIN_ARM64@|$(somme darwin_arm64)|g" "$1"
}
remplir "$depot/packaging/scoop/agentline.json.modele" > "$dist/agentline.json"
remplir "$depot/packaging/homebrew/agentline.rb.modele" > "$dist/agentline.rb"
grep -qE '@|""' "$dist/agentline.json" "$dist/agentline.rb" && { echo "manifeste incomplet (champ ou somme manquante)" >&2; exit 1; }
echo "dist/$version : $(ls "$dist" | wc -l) fichiers (binaires, SHA256SUMS, agentline.json pour Scoop, agentline.rb pour Homebrew)"
