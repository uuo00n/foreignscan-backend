Set-StrictMode -Version Latest

function Write-Stderr {
  param([Parameter(Mandatory = $true)][string]$Message)
  [Console]::Error.WriteLine($Message)
}

function Get-RepoRoot {
  $root = Join-Path $PSScriptRoot "..\.."
  return (Resolve-Path -LiteralPath $root).Path
}

function Resolve-EnvFilePath {
  param(
    [Parameter(Mandatory = $true)][string]$RootDir,
    [string]$EnvFileInput
  )

  $name = if ([string]::IsNullOrWhiteSpace($EnvFileInput)) { ".env.docker" } else { $EnvFileInput }
  if ([System.IO.Path]::IsPathRooted($name)) {
    return $name
  }
  return (Join-Path $RootDir $name)
}

function Ensure-EnvFile {
  param(
    [Parameter(Mandatory = $true)][string]$RootDir,
    [string]$EnvFileInput
  )

  $envFile = Resolve-EnvFilePath -RootDir $RootDir -EnvFileInput $EnvFileInput
  if (-not (Test-Path -LiteralPath $envFile)) {
    $example = Join-Path $RootDir ".env.docker.example"
    if (Test-Path -LiteralPath $example) {
      Copy-Item -LiteralPath $example -Destination $envFile
      Write-Host "Created $envFile from .env.docker.example"
    } else {
      throw "Error: $envFile not found and .env.docker.example is missing."
    }
  }
  return $envFile
}

function Get-ComposeOverrideFile {
  param([Parameter(Mandatory = $true)][ValidateSet("dev", "prod")][string]$Mode)
  if ($Mode -eq "prod") {
    return "compose.prod.yml"
  }
  return "compose.dev.yml"
}

function Get-ComposeArgs {
  param(
    [Parameter(Mandatory = $true)][string]$EnvFilePath,
    [Parameter(Mandatory = $true)][string]$OverrideFile
  )
  return @("compose", "--env-file", $EnvFilePath, "-f", "compose.yml", "-f", $OverrideFile)
}

function Assert-DockerComposeAvailable {
  if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Error: docker command not found."
  }

  $null = & docker compose version 2>$null
  if ($LASTEXITCODE -ne 0) {
    throw "Error: docker compose v2 is required."
  }
}

function Invoke-DockerCompose {
  param(
    [Parameter(Mandatory = $true)][string[]]$ComposeArgs,
    [Parameter(Mandatory = $true)][string[]]$CommandArgs
  )

  & docker @ComposeArgs @CommandArgs
  if ($LASTEXITCODE -ne 0) {
    $full = @("docker") + $ComposeArgs + $CommandArgs
    throw "Error: command failed: $($full -join ' ')"
  }
}

function Get-DockerComposeOutput {
  param(
    [Parameter(Mandatory = $true)][string[]]$ComposeArgs,
    [Parameter(Mandatory = $true)][string[]]$CommandArgs
  )

  $output = & docker @ComposeArgs @CommandArgs
  if ($LASTEXITCODE -ne 0) {
    $full = @("docker") + $ComposeArgs + $CommandArgs
    throw "Error: command failed: $($full -join ' ')"
  }
  return (($output -join "`n").Trim())
}

function Get-PostgresHealthTimeout {
  $raw = $env:POSTGRES_HEALTH_TIMEOUT
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return 90
  }

  $parsed = 0
  if (-not [int]::TryParse($raw, [ref]$parsed) -or $parsed -le 0) {
    throw "Error: POSTGRES_HEALTH_TIMEOUT must be a positive integer, got: $raw"
  }
  return $parsed
}

function Get-ContainerHealthStatus {
  param([Parameter(Mandatory = $true)][string]$ContainerId)

  $inspect = & docker inspect -f "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}" $ContainerId 2>$null
  if ($LASTEXITCODE -ne 0) {
    return ""
  }
  return (($inspect -join "`n").Trim())
}

function Get-ResponseBody {
  param([Parameter(Mandatory = $true)][System.Net.WebResponse]$Response)

  $stream = $null
  $reader = $null
  try {
    $stream = $Response.GetResponseStream()
    if ($null -eq $stream) {
      return ""
    }
    $reader = New-Object System.IO.StreamReader($stream)
    return $reader.ReadToEnd()
  } finally {
    if ($null -ne $reader) {
      $reader.Dispose()
    }
    if ($null -ne $stream) {
      $stream.Dispose()
    }
    $Response.Dispose()
  }
}

function Invoke-HttpRequest {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [int]$TimeoutSeconds = 3
  )

  try {
    $request = [System.Net.HttpWebRequest]::Create($Url)
    $request.Method = "GET"
    $request.Timeout = $TimeoutSeconds * 1000
    $request.ReadWriteTimeout = $TimeoutSeconds * 1000
    $request.UserAgent = "foreignscan-scripts-windows"

    try {
      $response = [System.Net.HttpWebResponse]$request.GetResponse()
      $status = [int]$response.StatusCode
      $body = Get-ResponseBody -Response $response
      return @{
        Reachable = $true
        StatusCode = $status
        Body = $body
        ErrorMessage = ""
      }
    } catch [System.Net.WebException] {
      $webEx = $_.Exception
      if ($null -ne $webEx.Response) {
        $status = [int]([System.Net.HttpWebResponse]$webEx.Response).StatusCode
        $body = Get-ResponseBody -Response $webEx.Response
        return @{
          Reachable = $true
          StatusCode = $status
          Body = $body
          ErrorMessage = $webEx.Message
        }
      }
      return @{
        Reachable = $false
        StatusCode = -1
        Body = ""
        ErrorMessage = $webEx.Message
      }
    }
  } catch {
    return @{
      Reachable = $false
      StatusCode = -1
      Body = ""
      ErrorMessage = $_.Exception.Message
    }
  }
}

function Test-SuccessStatusCode {
  param([int]$StatusCode)
  return ($StatusCode -ge 200 -and $StatusCode -lt 300)
}

function Get-EnvValue {
  param(
    [Parameter(Mandatory = $true)][string]$FilePath,
    [Parameter(Mandatory = $true)][string]$Key
  )

  if (-not (Test-Path -LiteralPath $FilePath)) {
    return ""
  }

  $pattern = "^\s*" + [regex]::Escape($Key) + "\s*=(.*)$"
  foreach ($line in [System.IO.File]::ReadLines($FilePath)) {
    if ($line -match "^\s*#") {
      continue
    }
    if ($line -match $pattern) {
      $value = $Matches[1].Trim()
      $value = $value.Trim('"').Trim("'").Trim()
      return $value
    }
  }

  return ""
}

function Compact-Body {
  param([string]$Body)
  $text = if ($null -eq $Body) { "" } else { $Body }
  $text = ($text -replace "[\r\n]+", " " -replace "\s+", " ").Trim()
  if ($text.Length -gt 180) {
    return $text.Substring(0, 180)
  }
  return $text
}

function Assert-Status200 {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Url
  )

  $resp = Invoke-HttpRequest -Url $Url -TimeoutSeconds 3
  if (-not [bool]$resp.Reachable) {
    throw "[contract][FAIL] ${Name}: 无法访问 ${Url}"
  }

  $statusCode = [int]$resp.StatusCode
  if ($statusCode -ne 200) {
    if ($statusCode -eq 404) {
      Write-Stderr "[contract][FAIL] ${Name}: HTTP 404（后端运行版本可能缺少该接口） ${Url}"
    } else {
      Write-Stderr "[contract][FAIL] ${Name}: HTTP ${statusCode} ${Url}"
    }

    $brief = Compact-Body -Body ([string]$resp.Body)
    if (-not [string]::IsNullOrWhiteSpace($brief)) {
      Write-Stderr "[contract][DETAIL] $brief"
    }
    throw "[contract][FAIL] ${Name}: status ${statusCode}"
  }

  Write-Host "[contract][OK] ${Name}: ${Url}"
}

function Normalize-DetectProbeUrl {
  param([string]$DetectUrl)

  if ([string]::IsNullOrWhiteSpace($DetectUrl)) {
    return ""
  }

  $base = $DetectUrl.Trim().TrimEnd("/")
  if ($base.EndsWith("/api")) {
    return "${base}/room-models"
  }
  return "${base}/api/room-models"
}

function Invoke-HostHealthWarnings {
  param([Parameter(Mandatory = $true)][string]$EnvFilePath)

  Write-Host ""
  Write-Host "Health checks:"

  $health = Invoke-HttpRequest -Url "http://localhost:3000/health" -TimeoutSeconds 3
  if (-not [bool]$health.Reachable -or -not (Test-SuccessStatusCode -StatusCode ([int]$health.StatusCode))) {
    Write-Host "Warning: /health is not reachable yet."
  }

  $ready = Invoke-HttpRequest -Url "http://localhost:3000/ready" -TimeoutSeconds 3
  if (-not [bool]$ready.Reachable -or -not (Test-SuccessStatusCode -StatusCode ([int]$ready.StatusCode))) {
    Write-Host "Warning: /ready is not reachable yet."
  }

  $detectUrl = Get-EnvValue -FilePath $EnvFilePath -Key "FS_DETECT_URL"
  if (-not [string]::IsNullOrWhiteSpace($detectUrl)) {
    $detectResp = Invoke-HttpRequest -Url $detectUrl -TimeoutSeconds 2
    if (-not [bool]$detectResp.Reachable -or -not (Test-SuccessStatusCode -StatusCode ([int]$detectResp.StatusCode))) {
      Write-Host "Warning: FS_DETECT_URL is unreachable from host: $detectUrl"
    }
  }
}

function Invoke-DockerUp {
  param(
    [Parameter(Mandatory = $true)][ValidateSet("dev", "prod")][string]$Mode,
    [string[]]$ExtraUpArgs = @()
  )

  $rootDir = Get-RepoRoot
  Set-Location -LiteralPath $rootDir
  Assert-DockerComposeAvailable

  $envFile = Ensure-EnvFile -RootDir $rootDir -EnvFileInput $env:ENV_FILE
  $override = Get-ComposeOverrideFile -Mode $Mode
  $composeArgs = Get-ComposeArgs -EnvFilePath $envFile -OverrideFile $override
  $timeout = Get-PostgresHealthTimeout

  Write-Host "[1/3] Starting postgres ($Mode)..."
  Invoke-DockerCompose -ComposeArgs $composeArgs -CommandArgs @("up", "-d", "postgres")

  Write-Host "[2/3] Waiting postgres to become healthy (timeout: ${timeout}s)..."
  $startTs = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
  while ($true) {
    $cid = Get-DockerComposeOutput -ComposeArgs $composeArgs -CommandArgs @("ps", "-q", "postgres")
    if (-not [string]::IsNullOrWhiteSpace($cid)) {
      $health = Get-ContainerHealthStatus -ContainerId $cid
      if ($health -eq "healthy") {
        Write-Host "Postgres is healthy."
        break
      }
    }

    $nowTs = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    if (($nowTs - $startTs) -ge $timeout) {
      & docker @composeArgs logs --tail=100 postgres
      throw "Error: Timed out waiting for postgres to become healthy."
    }
    Start-Sleep -Seconds 2
  }

  Write-Host "[3/3] Starting api ($Mode)..."
  $upArgs = @("up", "--build", "-d", "api") + $ExtraUpArgs
  Invoke-DockerCompose -ComposeArgs $composeArgs -CommandArgs $upArgs

  Write-Host ""
  Write-Host "Service status:"
  Invoke-DockerCompose -ComposeArgs $composeArgs -CommandArgs @("ps")

  Invoke-HostHealthWarnings -EnvFilePath $envFile
}

function Invoke-DockerDown {
  param(
    [Parameter(Mandatory = $true)][ValidateSet("dev", "prod")][string]$Mode,
    [string[]]$ExtraDownArgs = @()
  )

  $rootDir = Get-RepoRoot
  Set-Location -LiteralPath $rootDir
  Assert-DockerComposeAvailable

  $envFile = Ensure-EnvFile -RootDir $rootDir -EnvFileInput $env:ENV_FILE
  $override = Get-ComposeOverrideFile -Mode $Mode
  $composeArgs = Get-ComposeArgs -EnvFilePath $envFile -OverrideFile $override

  $downArgs = @("down") + $ExtraDownArgs
  Invoke-DockerCompose -ComposeArgs $composeArgs -CommandArgs $downArgs
}

function Invoke-DockerRebuild {
  param(
    [Parameter(Mandatory = $true)][ValidateSet("dev", "prod")][string]$Mode,
    [string[]]$ExtraUpArgs = @()
  )

  Write-Host "[rebuild] Stopping existing services ($Mode)..."
  Invoke-DockerDown -Mode $Mode -ExtraDownArgs @("--remove-orphans")

  Write-Host ""
  Write-Host "[rebuild] Rebuilding and starting services ($Mode)..."
  Invoke-DockerUp -Mode $Mode -ExtraUpArgs $ExtraUpArgs
}

function Invoke-ContractChecks {
  $rootDir = Get-RepoRoot
  $envFile = Resolve-EnvFilePath -RootDir $rootDir -EnvFileInput $env:ENV_FILE
  $backendBase = if ([string]::IsNullOrWhiteSpace($env:BACKEND_BASE)) { "http://127.0.0.1:3000" } else { $env:BACKEND_BASE }
  $yoloBase = if ([string]::IsNullOrWhiteSpace($env:YOLO_BASE)) { "http://127.0.0.1:8077" } else { $env:YOLO_BASE }

  Write-Host ""
  Write-Host "[contract] 开始联调契约检查..."

  Assert-Status200 -Name "backend health" -Url "$($backendBase.TrimEnd('/'))/health"
  Assert-Status200 -Name "backend room-models api" -Url "$($backendBase.TrimEnd('/'))/api/room-models"
  Assert-Status200 -Name "yolo room-models api" -Url "$($yoloBase.TrimEnd('/'))/api/room-models"

  $detectUrl = Get-EnvValue -FilePath $envFile -Key "FS_DETECT_URL"
  if ([string]::IsNullOrWhiteSpace($detectUrl)) {
    throw "[contract][FAIL] ${envFile} 未配置 FS_DETECT_URL"
  }

  $probeUrl = Normalize-DetectProbeUrl -DetectUrl $detectUrl
  if ([string]::IsNullOrWhiteSpace($probeUrl)) {
    throw "[contract][FAIL] FS_DETECT_URL 无法解析: ${detectUrl}"
  }

  $hostProbeUrl = ($probeUrl -replace "host\.docker\.internal", "127.0.0.1")
  Assert-Status200 -Name "FS_DETECT_URL(host probe)" -Url $hostProbeUrl

  Write-Host "[contract][OK] 联调契约检查通过"
}
