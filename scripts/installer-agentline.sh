#!/bin/bash
# Installe « agentline » sur Linux ou macOS depuis les versions GitHub du projet : télécharge l'archive de ce
# système, vérifie sa somme SHA-256 (SHA256SUMS), la décompresse, puis lance « agentline install » (binaire
# dans ~/.local/bin, statusline dans ~/.claude/settings.json avec une copie .bak, Nerd Font proposée).
# Usage :
#   curl -fsSL https://raw.githubusercontent.com/adrien-ln/agentline/main/scripts/installer-agentline.sh | bash
#   bash installer-agentline.sh [version] [source]
#     version : par exemple 0.1.1 (« v » permis) ; par défaut la dernière
#     source  : une URL de base (par défaut la version GitHub du tag), ou un dossier local (dist/<version>)
# Les messages sont en anglais : le script est lu par tout le monde avant d'être lancé.
set -euo pipefail

depot=https://github.com/adrien-ln/agentline
version=${1:-}
version=${version#v}
if [[ -z $version ]]; then
  # /releases/latest redirige vers /releases/tag/vX.Y.Z
  derniere=$(curl --fail --silent --show-error --location --output /dev/null --write-out '%{url_effective}' "$depot/releases/latest")
  version=${derniere##*/v}
  [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "agentline: could not find the latest release ($derniere)" >&2; exit 1; }
fi
source=${2:-$depot/releases/download/v$version}

case $(uname -s) in Linux) os=linux ;; Darwin) os=darwin ;; *) echo "agentline: unsupported system: $(uname -s) (on Windows: scoop install agentline)" >&2; exit 1 ;; esac
case $(uname -m) in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "agentline: unsupported processor: $(uname -m)" >&2; exit 1 ;; esac
archive=agentline_${version}_${os}_${arch}.tar.gz
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

recuperer() { # nom -> $tmp/nom
  if [[ -d $source ]]; then cp "$source/$1" "$tmp/$1"; return; fi
  curl --fail --silent --show-error --location -o "$tmp/$1" "$source/$1"
}
echo "agentline $version ($os/$arch)"
recuperer "$archive"
recuperer SHA256SUMS

attendu=$(awk -v f="$archive" '{ n = $2; sub(/^\*/, "", n) } n == f { print $1 }' "$tmp/SHA256SUMS")
if command -v sha256sum >/dev/null; then obtenu=$(sha256sum "$tmp/$archive" | awk '{ print $1 }')
else obtenu=$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }'); fi # macOS
if [[ -z $attendu || $attendu != "$obtenu" ]]; then
  echo "agentline: wrong SHA-256 checksum for $archive: nothing was installed" >&2
  exit 1
fi
echo "SHA-256 checksum verified: $archive"

tar -xzf "$tmp/$archive" -C "$tmp" agentline
chmod +x "$tmp/agentline"
# Lancé par « curl | bash », l'entrée standard est le script : la question de la police passe par le terminal.
if [[ -t 0 ]] || ! { : </dev/tty; } 2>/dev/null; then
  "$tmp/agentline" install
else
  "$tmp/agentline" install </dev/tty
fi
