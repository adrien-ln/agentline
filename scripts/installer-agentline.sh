#!/bin/bash
# Installe « agentline » sur Linux ou macOS : télécharge le binaire de ce système depuis les versions GitHub
# du projet, vérifie sa somme SHA-256, le place dans ~/.local/bin puis lance « agentline install ».
# Usage : bash installer-agentline.sh <version> [source]
#   <version> : par exemple 0.1.0 (sans « v »)
#   source : une URL de base (par défaut la version GitHub du tag v<version>), ou un dossier local (dist/<version>)
set -euo pipefail

version=${1:?usage : bash installer-agentline.sh <version> [source]}
version=${version#v}
source=${2:-https://github.com/adrien-ln/agentline/releases/download/v$version}

case $(uname -s) in Linux) os=linux ;; Darwin) os=darwin ;; *) echo "système non pris en charge : $(uname -s)" >&2; exit 1 ;; esac
case $(uname -m) in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "processeur non pris en charge : $(uname -m)" >&2; exit 1 ;; esac
fichier=agentline_${version}_${os}_${arch}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

recuperer() { # nom -> $tmp/nom
  if [[ -d $source ]]; then cp "$source/$1" "$tmp/$1"; return; fi
  curl --fail --silent --show-error --location -o "$tmp/$1" "$source/$1"
}
recuperer "$fichier"
recuperer SHA256SUMS

attendu=$(awk -v f="$fichier" '{ n = $2; sub(/^\*/, "", n) } n == f { print $1 }' "$tmp/SHA256SUMS")
if command -v sha256sum >/dev/null; then obtenu=$(sha256sum "$tmp/$fichier" | awk '{ print $1 }')
else obtenu=$(shasum -a 256 "$tmp/$fichier" | awk '{ print $1 }'); fi # macOS
if [[ -z $attendu || $attendu != "$obtenu" ]]; then
  echo "somme SHA-256 incorrecte pour $fichier : rien n'est installé" >&2
  exit 1
fi
echo "Somme SHA-256 vérifiée : $fichier"

chmod +x "$tmp/$fichier"
"$tmp/$fichier" install # copie dans ~/.local/bin/agentline, settings.json (copie .bak avant)
