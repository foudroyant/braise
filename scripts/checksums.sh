#!/usr/bin/env bash
# Produit checksums.txt pour les binaires d'un dossier.
#
# Usage :
#   ./scripts/checksums.sh [dossier]
#
# Par défaut : bin/
# Si GPG_KEY_ID est défini, produit aussi checksums.txt.asc.

set -euo pipefail

cd "$(dirname "$0")/.."
DIR="${1:-bin}"

if [[ ! -d "$DIR" ]]; then
  echo "Dossier introuvable : $DIR" >&2
  exit 1
fi

cd "$DIR"

if command -v sha256sum >/dev/null 2>&1; then
  HASH=(sha256sum)
else
  HASH=(shasum -a 256)
fi

FILES=()
for file in braise-*; do
  [[ -f "$file" ]] && FILES+=("$file")
done
if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "Aucun binaire braise-* dans $DIR" >&2
  exit 1
fi

: > checksums.txt
for file in "${FILES[@]}"; do
  "${HASH[@]}" "$file" >> checksums.txt
done

echo "Écrit $DIR/checksums.txt"
cat checksums.txt

if [[ -n "${GPG_KEY_ID:-}" ]]; then
  gpg --batch --yes --local-user "$GPG_KEY_ID" \
    --armor --detach-sign --output checksums.txt.asc checksums.txt
  echo "Écrit $DIR/checksums.txt.asc"
fi
