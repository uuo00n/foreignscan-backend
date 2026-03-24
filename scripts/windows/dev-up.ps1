param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ExtraUpArgs
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/common.ps1"

try {
  Invoke-DockerUp -Mode "dev" -ExtraUpArgs $ExtraUpArgs
  Invoke-ContractChecks
} catch {
  Write-Stderr $_.Exception.Message
  exit 1
}
