param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ExtraDownArgs
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/common.ps1"

try {
  Invoke-DockerDown -Mode "dev" -ExtraDownArgs $ExtraDownArgs
} catch {
  Write-Stderr $_.Exception.Message
  exit 1
}
