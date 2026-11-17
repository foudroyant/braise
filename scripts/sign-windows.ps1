# Signe un ou plusieurs exécutables Windows avec signtool.
#
# Usage :
#   pwsh ./scripts/sign-windows.ps1 -Path bin/braise-windows-amd64.exe -PfxPath C:\secrets\braise.pfx
#
# Ou avec les variables d'environnement :
#   WINDOWS_CERT_PFX, WINDOWS_CERT_PASSWORD
#
# Ou avec un certificat déjà installé dans le magasin Windows :
#   WINDOWS_CERT_THUMBPRINT=...
#
# Le mot de passe peut aussi être lu depuis WINDOWS_CERT_PASSWORD pour éviter
# de l'écrire dans l'historique du shell.

[CmdletBinding()]
param(
  [Parameter(Mandatory = $true, Position = 0)]
  [string[]] $Path,

  [string] $PfxPath = $env:WINDOWS_CERT_PFX,
  [string] $PfxPassword = $env:WINDOWS_CERT_PASSWORD,
  [string] $Thumbprint = $env:WINDOWS_CERT_THUMBPRINT,
  [string] $TimestampUrl = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"

function Find-SignTool {
  $cmd = Get-Command signtool.exe -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }

  $kits = "${env:ProgramFiles(x86)}\Windows Kits\10\bin"
  if (Test-Path $kits) {
    $found = Get-ChildItem $kits -Filter signtool.exe -Recurse -ErrorAction SilentlyContinue |
      Sort-Object FullName -Descending |
      Select-Object -First 1
    if ($found) { return $found.FullName }
  }
  throw "signtool.exe introuvable. Installer le Windows SDK ou ajouter signtool au PATH."
}

$signTool = Find-SignTool

if (-not $PfxPath -and -not $Thumbprint) {
  throw "Fournir -PfxPath et -PfxPassword, ou -Thumbprint, ou les variables WINDOWS_CERT_*."
}

foreach ($file in $Path) {
  if (-not (Test-Path $file)) {
    throw "Fichier introuvable : $file"
  }

  Write-Host "Signature de $file"
  if ($PfxPath) {
    & $signTool sign /fd SHA256 /f $PfxPath /p $PfxPassword /tr $TimestampUrl /td SHA256 $file
  } else {
    & $signTool sign /fd SHA256 /sha1 $Thumbprint /tr $TimestampUrl /td SHA256 $file
  }
  if ($LASTEXITCODE -ne 0) {
    throw "signtool sign a échoué pour $file (code $LASTEXITCODE)."
  }

  & $signTool verify /pa /v $file
  if ($LASTEXITCODE -ne 0) {
    throw "signtool verify a échoué pour $file (code $LASTEXITCODE)."
  }
}

Write-Host "Signature et vérification terminées."
