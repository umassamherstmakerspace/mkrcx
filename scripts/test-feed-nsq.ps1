$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Some Codex Windows shells contain both `Path` and `PATH`. PowerShell's
# Start-Process treats environment keys case-insensitively and rejects that
# otherwise harmless duplicate, so normalize it within this script process.
$inheritedPath = $env:Path
[Environment]::SetEnvironmentVariable("PATH", $null, [EnvironmentVariableTarget]::Process)
[Environment]::SetEnvironmentVariable("Path", $inheritedPath, [EnvironmentVariableTarget]::Process)

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$backend = Join-Path $repositoryRoot "backend"
$go = Join-Path $repositoryRoot ".tools\go1.26.5\go\bin\go.exe"
$nsqBin = Join-Path $repositoryRoot ".tools\nsq\v1.3.0\bin"
$nsqd = Join-Path $nsqBin "nsqd.exe"
$lookupd = Join-Path $nsqBin "nsqlookupd.exe"
$receiptPath = Join-Path $repositoryRoot ".tools\nsq\v1.3.0\receipt.json"

foreach ($file in @($go, $nsqd, $lookupd, $receiptPath)) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "Missing trusted test prerequisite: $file. Run scripts/bootstrap-nsq.ps1 first."
    }
}
$receipt = Get-Content -LiteralPath $receiptPath -Raw | ConvertFrom-Json
if ((Get-FileHash -Algorithm SHA256 -LiteralPath $nsqd).Hash -ne $receipt.nsqd_sha256 -or
    (Get-FileHash -Algorithm SHA256 -LiteralPath $lookupd).Hash -ne $receipt.nsqlookupd_sha256) {
    throw "NSQ binary hash does not match the local verified-build receipt"
}

$usedPorts = [System.Collections.Generic.HashSet[int]]::new()
function Get-UniqueLoopbackPort {
    do {
        $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
        $listener.Start()
        try {
            $port = ([Net.IPEndPoint]$listener.LocalEndpoint).Port
        }
        finally {
            $listener.Stop()
        }
    } while (-not $usedPorts.Add($port))
    return $port
}

function Wait-NSQReady([string]$Uri, [Diagnostics.Process]$Process) {
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    while ([DateTime]::UtcNow -lt $deadline) {
        if ($Process.HasExited) {
            throw "$($Process.ProcessName) exited before becoming ready"
        }
        try {
            $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 1 -Uri $Uri
            if ($response.StatusCode -eq 200 -and $response.Content.Trim() -eq "OK") {
                return
            }
        }
        catch {
            Start-Sleep -Milliseconds 50
        }
    }
    throw "Timed out waiting for $Uri"
}

function Wait-NSQTopic([string]$Uri) {
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    while ([DateTime]::UtcNow -lt $deadline) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 1 -Uri $Uri
            if ($response.StatusCode -eq 200) {
                return
            }
        }
        catch {
            Start-Sleep -Milliseconds 50
        }
    }
    throw "Timed out waiting for topic discovery at $Uri"
}

$lookupTcp = Get-UniqueLoopbackPort
$lookupHttp = Get-UniqueLoopbackPort
$nsqdTcp = Get-UniqueLoopbackPort
$nsqdHttp = Get-UniqueLoopbackPort
$runRoot = Join-Path ([IO.Path]::GetTempPath()) ("mkrcx-nsq-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $runRoot | Out-Null
$lookupProcess = $null
$nsqdProcess = $null

try {
    $lookupProcess = Start-Process -FilePath $lookupd -PassThru -WindowStyle Hidden `
        -ArgumentList @(
            "--tcp-address=127.0.0.1:$lookupTcp",
            "--http-address=127.0.0.1:$lookupHttp",
            "--broadcast-address=127.0.0.1"
        ) `
        -RedirectStandardOutput (Join-Path $runRoot "lookupd.stdout.log") `
        -RedirectStandardError (Join-Path $runRoot "lookupd.stderr.log")
    Wait-NSQReady "http://127.0.0.1:$lookupHttp/ping" $lookupProcess

    $nsqdProcess = Start-Process -FilePath $nsqd -PassThru -WindowStyle Hidden `
        -ArgumentList @(
            "--tcp-address=127.0.0.1:$nsqdTcp",
            "--http-address=127.0.0.1:$nsqdHttp",
            "--lookupd-tcp-address=127.0.0.1:$lookupTcp",
            "--broadcast-address=127.0.0.1",
            "--data-path=$runRoot"
        ) `
        -RedirectStandardOutput (Join-Path $runRoot "nsqd.stdout.log") `
        -RedirectStandardError (Join-Path $runRoot "nsqd.stderr.log")
    Wait-NSQReady "http://127.0.0.1:$nsqdHttp/ping" $nsqdProcess

    $topic = "mkrcx-feed-items-v1"
    $null = Invoke-WebRequest -UseBasicParsing -Method Post -TimeoutSec 5 `
        -Uri "http://127.0.0.1:$nsqdHttp/topic/create?topic=$topic"
    Wait-NSQTopic "http://127.0.0.1:$lookupHttp/lookup?topic=$topic"

    $env:NSQ_TEST_NSQD = "127.0.0.1:$nsqdTcp"
    $env:NSQ_TEST_LOOKUPD = "127.0.0.1:$lookupHttp"
    $env:NSQ_TEST_NSQD_HTTP = "http://127.0.0.1:$nsqdHttp"
    $env:GOTOOLCHAIN = "local"
    $env:GOPROXY = "off"
    $env:GOMODCACHE = (Resolve-Path (Join-Path $repositoryRoot ".tools\gomodcache")).Path
    $env:GOCACHE = (Resolve-Path (Join-Path $repositoryRoot ".tools\gocache-go1.26.5")).Path

    Push-Location $backend
    try {
        & $go test -mod=readonly -tags=nsq_integration -run "^TestNSQFeedRuntimeFansOutAcrossInstances$" -count=1 -timeout=45s -v ./src/leash/api
        if ($LASTEXITCODE -ne 0) {
            throw "NSQ feed integration test failed"
        }
    }
    finally {
        Pop-Location
    }
}
catch {
    foreach ($log in Get-ChildItem -LiteralPath $runRoot -Filter "*.log" -ErrorAction SilentlyContinue) {
        Write-Host "--- $($log.Name) ---"
        Get-Content -LiteralPath $log.FullName
    }
    throw
}
finally {
    foreach ($process in @($nsqdProcess, $lookupProcess)) {
        if ($null -ne $process -and -not $process.HasExited) {
            Stop-Process -Id $process.Id -Force
            $null = $process.WaitForExit(5000)
        }
    }
    $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $resolvedRun = [IO.Path]::GetFullPath($runRoot)
    if ($resolvedRun.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase) -and
        [IO.Path]::GetFileName($resolvedRun).StartsWith("mkrcx-nsq-", [StringComparison]::Ordinal)) {
        Remove-Item -LiteralPath $resolvedRun -Recurse -Force
    }
}
