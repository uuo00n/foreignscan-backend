param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ExtraUpArgs
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/common.ps1"

try {
  Invoke-DockerRebuild -Mode "prod" -ExtraUpArgs $ExtraUpArgs
} catch {
  Write-Stderr $_.Exception.Message
  exit 1
}
