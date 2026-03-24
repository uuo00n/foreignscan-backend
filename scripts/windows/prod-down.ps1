param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ExtraDownArgs
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/common.ps1"

try {
  Invoke-DockerDown -Mode "prod" -ExtraDownArgs $ExtraDownArgs
} catch {
  Write-Stderr $_.Exception.Message
  exit 1
}
