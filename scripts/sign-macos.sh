#!/usr/bin/env bash
# Signe, emballe et notarise un binaire macOS.
#
# Usage :
#   ./scripts/sign-macos.sh <binaire> [version]
#
# Variables obligatoires :
#   APPLE_SIGN_IDENTITY       "Developer ID Application: Nom (TEAMID)"
#   APPLE_INSTALLER_IDENTITY  "Developer ID Installer: Nom (TEAMID)"
#
# Variables d'import du certificat (facultatives si déjà dans le trousseau) :
#   APPLE_CERT_P12_BASE64, APPLE_CERT_PASSWORD
#
# Variables de notarisation :
#   APPLE_ID, APPLE_TEAM_ID, APPLE_APP_PASSWORD
#
# Le binaire est signé, puis placé dans un .pkg signé, notarié et agrafé.
# Un binaire seul ne peut pas être notarié directement par Apple.

set -euo pipefail

BIN="${1:?Usage: $0 <binaire> [version]}"
VERSION="${2:-0.1.0}"
OUT_DIR="${OUT_DIR:-dist}"
IDENTITY="${APPLE_SIGN_IDENTITY:?Définir APPLE_SIGN_IDENTITY}"
INSTALLER_IDENTITY="${APPLE_INSTALLER_IDENTITY:?Définir APPLE_INSTALLER_IDENTITY}"

if [[ ! -f "$BIN" ]]; then
  echo "Binaire introuvable : $BIN" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
NAME="$(basename "$BIN")"
KEYCHAIN=""
ORIGINAL_KEYCHAINS=""

cleanup() {
  if [[ -n "$KEYCHAIN" && -f "$KEYCHAIN" ]]; then
    security delete-keychain "$KEYCHAIN" >/dev/null 2>&1 || true
  fi
  if [[ -n "$ORIGINAL_KEYCHAINS" ]]; then
    # shellcheck disable=SC2086
    security list-keychains -d user -s $ORIGINAL_KEYCHAINS >/dev/null 2>&1 || true
  fi
  [[ -n "${TMP_P12:-}" ]] && rm -f "$TMP_P12"
  [[ -n "${STAGE:-}" ]] && rm -rf "$STAGE"
}
trap cleanup EXIT

if [[ -n "${APPLE_CERT_P12_BASE64:-}" ]]; then
  TMP_ROOT="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
  KEYCHAIN="$TMP_ROOT/braise-signing.keychain-db"
  KEYCHAIN_PASSWORD="${APPLE_KEYCHAIN_PASSWORD:-$(openssl rand -hex 24)}"
  TMP_P12="$(mktemp -t braise-cert).p12"

  security delete-keychain "$KEYCHAIN" >/dev/null 2>&1 || true
  security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
  security set-keychain-settings -lut 21600 "$KEYCHAIN"
  security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
  printf '%s' "$APPLE_CERT_P12_BASE64" | openssl base64 -d -A > "$TMP_P12"
  security import "$TMP_P12" -k "$KEYCHAIN" -P "${APPLE_CERT_PASSWORD:?Définir APPLE_CERT_PASSWORD}" \
    -T /usr/bin/codesign -T /usr/bin/pkgbuild
  security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN" >/dev/null
  # pkgbuild ne prend pas d'option --keychain : le trousseau doit être dans la liste.
  ORIGINAL_KEYCHAINS="$(security list-keychains -d user | sed 's/^[[:space:]]*"//; s/"$//')"
  # shellcheck disable=SC2086
  security list-keychains -d user -s "$KEYCHAIN" $ORIGINAL_KEYCHAINS
fi

CODESIGN_ARGS=(--force --options runtime --timestamp --sign "$IDENTITY")
if [[ -n "$KEYCHAIN" ]]; then
  CODESIGN_ARGS+=(--keychain "$KEYCHAIN")
fi

echo "Signature de $BIN"
codesign "${CODESIGN_ARGS[@]}" "$BIN"
codesign --verify --strict --verbose=2 "$BIN"

STAGE="$(mktemp -d)"
mkdir -p "$STAGE/usr/local/bin"
cp "$BIN" "$STAGE/usr/local/bin/braise"
chmod 0755 "$STAGE/usr/local/bin/braise"

ARCH_SUFFIX="${NAME#braise-darwin-}"
PKG="$OUT_DIR/braise-${VERSION}-darwin-${ARCH_SUFFIX}.pkg"
echo "Construction de $PKG"
pkgbuild \
  --root "$STAGE" \
  --identifier "io.github.foudroyant.braise" \
  --version "$VERSION" \
  --install-location "/" \
  --sign "$INSTALLER_IDENTITY" \
  "$PKG"

if [[ -n "${APPLE_ID:-}" && -n "${APPLE_TEAM_ID:-}" && -n "${APPLE_APP_PASSWORD:-}" ]]; then
  echo "Notarisation de $PKG"
  xcrun notarytool submit "$PKG" \
    --apple-id "$APPLE_ID" \
    --team-id "$APPLE_TEAM_ID" \
    --password "$APPLE_APP_PASSWORD" \
    --wait
  xcrun stapler staple "$PKG"
  spctl -a -vv -t install "$PKG"
else
  echo "Notarisation ignorée : APPLE_ID, APPLE_TEAM_ID ou APPLE_APP_PASSWORD manquant." >&2
fi

echo "Paquet prêt : $PKG"
