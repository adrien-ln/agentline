#!/bin/bash
# Installe « agentline » sur Linux ou macOS : télécharge le binaire de ce système depuis le registre de paquets
# GitLab, vérifie sa somme SHA-256, le place dans ~/.local/bin puis lance « agentline install ».
# Usage : bash installer-agentline.sh <version> [source]
#   source : une URL de base (par défaut le registre du projet), ou un dossier local (dist/<version>)
# Projet privé : exporte GITLAB_TOKEN (jeton personnel ou de déploiement, droit read_package_registry).
set -euo pipefail

version=${1:?usage : bash installer-agentline.sh <version> [source]}
source=${2:-https://gitlab.com/api/v4/projects/adrien-ln%2Fstatusline/packages/generic/agentline/$version}

case $(uname -s) in Linux) os=linux ;; Darwin) os=darwin ;; *) echo "système non pris en charge : $(uname -s)" >&2; exit 1 ;; esac
case $(uname -m) in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "processeur non pris en charge : $(uname -m)" >&2; exit 1 ;; esac
fichier=agentline_${version}_${os}_${arch}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

recuperer() { # nom -> $tmp/nom
  if [[ -d $source ]]; then cp "$source/$1" "$tmp/$1"; return; fi
  local entete=()
  [[ -n ${GITLAB_TOKEN:-} ]] && entete=(--header "PRIVATE-TOKEN: $GITLAB_TOKEN")
  curl --fail --silent --show-error --location "${entete[@]}" -o "$tmp/$1" "$source/$1"
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
