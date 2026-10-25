#!/bin/bash
# Notes d'une version (Markdown, en anglais), tirées des messages de commit Conventional Commits depuis le tag
# précédent, groupées par type. Lancé par .github/workflows/release.yml ; à la main pour relire avant un tag.
# Usage : bash scripts/notes-de-version.sh <tag> [tag précédent]
#   <tag>            par exemple v0.1.0 (ou HEAD pour voir ce que donnerait la prochaine version)
#   [tag précédent]  par défaut le dernier tag vX.Y.Z avant <tag> ; s'il n'y en a pas, tout l'historique
# Pas de dépendance : bash 4 et git.
set -euo pipefail
cd "${BASH_SOURCE[0]%/*}/.."

tag=${1:?usage: bash scripts/notes-de-version.sh <tag> [previous tag]}
precedent=${2:-$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' "$tag^" 2>/dev/null || true)}
plage=$tag
[[ -n $precedent ]] && plage=$precedent..$tag
depot=${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:-adrien-ln/agentline}

# Rubriques, dans l'ordre d'affichage ; « OTHER » reçoit les messages hors convention
types=(BREAKING feat fix perf refactor docs test build ci chore revert OTHER)
declare -A titres=([BREAKING]="Breaking changes" [feat]=Features [fix]=Fixes [perf]=Performance
  [refactor]=Refactoring [docs]=Documentation [test]=Tests [build]=Build [ci]=CI [chore]=Chores
  [revert]=Reverts [OTHER]=Other)
declare -A rubriques=()

motif='^([a-z]+)(\(([a-z0-9-]+)\))?(!)?: (.+)$'
while IFS=$'\t' read -r sha sujet; do
  type=OTHER ligne="- $sujet ($sha)"
  if [[ $sujet =~ $motif && -n ${titres[${BASH_REMATCH[1]}]:-} ]]; then
    type=${BASH_REMATCH[1]} portee=${BASH_REMATCH[3]} resume=${BASH_REMATCH[5]}
    ligne="- ${portee:+**$portee:** }$resume ($sha)"
    rupture='(^|'$'\n'')BREAKING[ -]CHANGE:'
    if [[ -n ${BASH_REMATCH[4]} || $(git log -1 --format=%b "$sha") =~ $rupture ]]; then
      type=BREAKING
    fi
  fi
  rubriques[$type]+=$ligne$'\n'
done < <(git log --no-merges --reverse --format='%h%x09%s' "$plage")

for t in "${types[@]}"; do
  [[ -n ${rubriques[$t]:-} ]] && printf '## %s\n\n%s\n' "${titres[$t]}" "${rubriques[$t]}"
done
if [[ -n $precedent ]]; then
  echo "**Full changelog**: $depot/compare/$precedent...$tag"
else
  echo "**Full changelog**: $depot/commits/$tag"
fi
