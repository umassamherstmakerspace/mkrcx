param(
    [string]$Version = "v1.3.0"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($Version -ne "v1.3.0") {
    throw "Only the reviewed NSQ version v1.3.0 is allowed by this script."
}

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$go = Join-Path $repositoryRoot ".tools\go1.26.5\go\bin\go.exe"
if (-not (Test-Path -LiteralPath $go -PathType Leaf)) {
    throw "Verified Go 1.26.5 is missing at $go"
}
$goVersion = (& $go version).Trim()
if ($goVersion -ne "go version go1.26.5 windows/amd64") {
    throw "Unexpected Go toolchain: $goVersion"
}

$nsqRoot = Join-Path $repositoryRoot ".tools\nsq\v1.3.0"
$bin = Join-Path $nsqRoot "bin"
$moduleCache = Join-Path $repositoryRoot ".tools\nsq\gomodcache"
$buildCache = Join-Path $repositoryRoot ".tools\nsq\gocache"
New-Item -ItemType Directory -Force -Path $bin, $moduleCache, $buildCache | Out-Null

$env:GOTOOLCHAIN = "local"
$env:GOPROXY = "https://proxy.golang.org"
$env:GOSUMDB = "sum.golang.org"
$env:GONOSUMDB = ""
$env:GOPRIVATE = ""
$env:GONOPROXY = ""
$env:GOVCS = "*:off"
$env:GOMODCACHE = $moduleCache
$env:GOCACHE = $buildCache

$module = (& $go mod download -json "github.com/nsqio/nsq@$Version" | ConvertFrom-Json)
if ($LASTEXITCODE -ne 0) {
    throw "NSQ module download failed"
}
if ($module.Sum -ne "h1:v7NtyO844ieTIOCQEqQ7IUSSi1ImhgrTTto1rgIYGEU=") {
    throw "Unexpected NSQ source checksum: $($module.Sum)"
}
if ($module.GoModSum -ne "h1:RxNr6UC0kSkNF44LnJrlN3U3CQnQGTXk+QKfSZLzqvc=") {
    throw "Unexpected NSQ go.mod checksum: $($module.GoModSum)"
}
if ($module.Origin.URL -ne "https://github.com/nsqio/nsq" -or
    $module.Origin.Hash -ne "f580340fd5be61a94d0fef388458f027925c7fc0" -or
    $module.Origin.Ref -ne "refs/tags/v1.3.0") {
    throw "Unexpected NSQ source origin"
}

Push-Location $module.Dir
try {
    & $go mod verify
    if ($LASTEXITCODE -ne 0) {
        throw "NSQ dependency verification failed"
    }
    & $go build -mod=readonly -trimpath -buildvcs=false -o (Join-Path $bin "nsqd.exe") ./apps/nsqd
    if ($LASTEXITCODE -ne 0) {
        throw "nsqd build failed"
    }
    & $go build -mod=readonly -trimpath -buildvcs=false -o (Join-Path $bin "nsqlookupd.exe") ./apps/nsqlookupd
    if ($LASTEXITCODE -ne 0) {
        throw "nsqlookupd build failed"
    }
}
finally {
    Pop-Location
}

$nsqd = Join-Path $bin "nsqd.exe"
$lookupd = Join-Path $bin "nsqlookupd.exe"
if ((& $nsqd --version).Trim() -notmatch "^nsqd v1\.3\.0 \(built w/go1\.26\.5\)$") {
    throw "Unexpected nsqd version"
}
if ((& $lookupd --version).Trim() -notmatch "^nsqlookupd v1\.3\.0 \(built w/go1\.26\.5\)$") {
    throw "Unexpected nsqlookupd version"
}

$receipt = [ordered]@{
    module = "github.com/nsqio/nsq"
    version = $Version
    origin = $module.Origin.URL
    origin_commit = $module.Origin.Hash
    source_sum = $module.Sum
    go_mod_sum = $module.GoModSum
    go = $goVersion
    nsqd_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $nsqd).Hash
    nsqlookupd_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $lookupd).Hash
    built_at_utc = [DateTime]::UtcNow.ToString("o")
}
$receipt | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $nsqRoot "receipt.json") -Encoding utf8
$receipt | Format-List
