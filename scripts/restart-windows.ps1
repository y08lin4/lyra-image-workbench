param(
  [int]$Port = 8787,
  [switch]$BuildWeb
)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Resolve-Path (Join-Path $ScriptDir '..')
$Exe = Join-Path $Root 'bin\image-workbench-local-server.exe'
$OutLog = Join-Path $Root 'local-server.out.log'
$ErrLog = Join-Path $Root 'local-server.err.log'

function Write-Step($Message) {
  Write-Host "[image-workbench] $Message"
}

function Get-RouteStatus($Uri, $Method, $Body) {
  try {
    $headers = @{ 'Content-Type' = 'application/json' }
    $resp = Invoke-WebRequest -UseBasicParsing -Uri $Uri -Method $Method -Headers $headers -Body $Body -TimeoutSec 5
    return [int]$resp.StatusCode
  } catch {
    if ($_.Exception.Response) {
      return [int]$_.Exception.Response.StatusCode
    }
    throw
  }
}

Set-Location $Root
New-Item -ItemType Directory -Force -Path (Join-Path $Root 'bin') | Out-Null

$targets = @()
$named = Get-Process -Name 'image-workbench-local-server' -ErrorAction SilentlyContinue
if ($named) {
  $targets += $named
}

try {
  $listeners = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  foreach ($listener in $listeners) {
    if ($listener.OwningProcess -and $listener.OwningProcess -ne 0) {
      $oldProc = Get-Process -Id $listener.OwningProcess -ErrorAction SilentlyContinue
      if ($oldProc) {
        $targets += $oldProc
      }
    }
  }
} catch {
  Write-Step "Could not read listening process on port $Port; continuing with process-name restart."
}

$targets = $targets | Sort-Object Id -Unique
foreach ($oldProc in $targets) {
  Write-Step "Stopping old backend: PID $($oldProc.Id) $($oldProc.ProcessName)"
  Stop-Process -Id $oldProc.Id -Force
}

if ($BuildWeb) {
  Write-Step 'Building frontend web/dist'
  Push-Location (Join-Path $Root 'web')
  npm run build
  Pop-Location
}

Write-Step 'Building latest Go backend'
go build -o $Exe ./cmd/local-server

Write-Step "Starting backend: http://127.0.0.1:$Port"
$proc = Start-Process -FilePath $Exe -WorkingDirectory $Root -WindowStyle Hidden -RedirectStandardOutput $OutLog -RedirectStandardError $ErrLog -PassThru

$healthUrl = "http://127.0.0.1:$Port/api/health"
$promptUrl = "http://127.0.0.1:$Port/api/prompt-tools/text-to-prompt"
$ready = $false
for ($i = 0; $i -lt 20; $i++) {
  Start-Sleep -Milliseconds 500
  try {
    $health = Invoke-WebRequest -UseBasicParsing -Uri $healthUrl -TimeoutSec 3
    if ([int]$health.StatusCode -eq 200) {
      $ready = $true
      break
    }
  } catch {
  }
}

if (-not $ready) {
  Write-Step "Backend was not ready in time. Check log: $ErrLog"
  exit 1
}

$status = Get-RouteStatus -Uri $promptUrl -Method 'POST' -Body '{"input":"route-check","style":"auto","ratio":"auto","language":"zh","target":"image-2"}'
if ($status -eq 405) {
  Write-Step 'Prompt tools still returned HTTP 405; current port may still be served by an old backend.'
  exit 1
}

Write-Step "Restart complete: PID $($proc.Id)"
Write-Step "Prompt tools route check: HTTP $status (anything except 405 means the POST route is active)"
Write-Step "Log: $ErrLog"
