#!/bin/bash
# Vérifie que des messages de commit (et le titre d'une pull request) suivent la convention du projet :
# « type(portée)?: résumé », 72 caractères au plus, sans point final (voir CONTRIBUTING.md).
# Lancé par .github/workflows/commits.yml ; à la main avant d'ouvrir une pull request, ou en hook commit-msg.
# Usage :
#   bash scripts/verifier-commits.sh origin/main..HEAD     chaque commit de la plage (commits de fusion compris)
#   bash scripts/verifier-commits.sh --title "<titre>"     un titre de pull request
#   bash scripts/verifier-commits.sh --file <fichier>      un message dans un fichier (.git/hooks/commit-msg : --file "$1")
# Les options se combinent. Code de sortie : 0 tout est conforme, 1 un message ne l'est pas, 2 usage.
# Pas de dépendance : bash et git.
set -uo pipefail
export LC_ALL=C.UTF-8 # longueur en caractères, pas en octets

types='feat|fix|perf|refactor|test|docs|chore|ci|build|revert'
portees='render|selector|style|theme|install|cli|i18n|colors|docs|ci|deps'
max=72
motif="^($types)(\\(($portees)\\))?!?: (.*)$"
erreurs=0

# signaler <où> <sujet> <raison>
signaler() {
  erreurs=$((erreurs + 1))
  echo "✗ $1: \"$2\""
  echo "    $3"
  if [[ ${GITHUB_ACTIONS:-} == true ]]; then echo "::error title=$1::$3: $2"; fi
}

# verifier_sujet <où> <sujet>
verifier_sujet() {
  local ou=$1 s=$2
  if (( ${#s} > max )); then
    signaler "$ou" "$s" "${#s} characters, $max at most"
  fi
  if [[ ! $s =~ $motif ]]; then
    local t='' entre='' p=''
    if [[ $s =~ ^([A-Za-z]+)(\(([^\)]*)\))?!?: ]]; then
      t=${BASH_REMATCH[1]} entre=${BASH_REMATCH[2]} p=${BASH_REMATCH[3]}
    fi
    if [[ -n $t && ! $t =~ ^($types)$ ]]; then
      signaler "$ou" "$s" "unknown type \"$t\" (allowed: ${types//|/, })"
    elif [[ -n $entre && ! $p =~ ^($portees)$ ]]; then
      signaler "$ou" "$s" "unknown scope \"$p\" (allowed: ${portees//|/, }, or none)"
    elif [[ -n $t ]]; then
      signaler "$ou" "$s" "one space after the colon, then the summary"
    else
      signaler "$ou" "$s" "expected \"type(scope): summary\" or \"type: summary\""
    fi
    return
  fi
  local resume=${BASH_REMATCH[4]}
  if [[ -z $resume || $resume =~ ^[[:space:]] || $resume =~ [[:space:]]$ ]]; then
    signaler "$ou" "$s" "one space after the colon, then the summary, no trailing space"
  elif [[ $resume =~ ^[[:upper:]] ]]; then
    signaler "$ou" "$s" "the summary starts with a lowercase letter"
  elif [[ $resume == *. ]]; then
    signaler "$ou" "$s" "no final period"
  fi
}

# verifier_message <où> <message complet> : le sujet, puis une ligne vide avant le corps
verifier_message() {
  local sujet reste
  sujet=${2%%$'\n'*}
  sujet=${sujet%$'\r'}
  verifier_sujet "$1" "$sujet"
  if [[ $2 == *$'\n'* ]]; then
    reste=${2#*$'\n'}
    reste=${reste%%$'\n'*}
    if [[ -n ${reste%$'\r'} ]]; then
      signaler "$1" "$sujet" "leave a blank line between the subject and the body"
    fi
  fi
}

usage() {
  cat >&2 <<'FIN'
Usage:
  bash scripts/verifier-commits.sh origin/main..HEAD     every commit of the range (merge commits included)
  bash scripts/verifier-commits.sh --title "<title>"     a pull request title
  bash scripts/verifier-commits.sh --file <file>         a message in a file (.git/hooks/commit-msg: --file "$1")
Options combine. Exit code: 0 all good, 1 a message does not follow the convention, 2 usage.
FIN
  exit 2
}
(( $# )) || usage

while (( $# )); do
  case $1 in
    --title)
      (( $# >= 2 )) || usage
      verifier_sujet "pull request title" "$2"
      shift 2 ;;
    --file)
      if (( $# < 2 )) || [[ ! -f $2 ]]; then usage; fi
      # Comme git : lignes de commentaire retirées, lignes vides de tête ignorées
      message=$(grep -v '^#' "$2" | sed '/./,$!d')
      verifier_message "commit message" "$message"
      shift 2 ;;
    -h|--help) usage ;;
    -*) usage ;;
    *)
      if ! liste=$(git rev-list --reverse "$1"); then
        echo "invalid range: $1" >&2
        exit 2
      fi
      for sha in $liste; do
        verifier_message "commit ${sha:0:7}" "$(git log -1 --format=%B "$sha")"
      done
      shift ;;
  esac
done

if (( erreurs )); then
  cat <<EOF

$erreurs problem(s). Commit messages and pull request titles follow Conventional Commits:

    type(scope): imperative summary in lowercase

  types:  ${types//|/, }
  scopes: ${portees//|/, } (optional)
  at most $max characters, no final period, a blank line before the body.

See "Commits and pull requests" in CONTRIBUTING.md. To fix the last commit: git commit --amend;
older ones: git rebase -i, then reword.
EOF
  exit 1
fi
echo "Commit messages: OK"
