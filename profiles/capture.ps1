param(
    [Parameter(Mandatory = $true)]
    [string]$OutFile
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
if (-not $root) { $root = Get-Location }
Set-Location $root

$addr = "127.0.0.1:18080"
$base = "http://$addr"
$batch = Join-Path $root "profiles\testdata\batch.json"
$bin = Join-Path $root "profiles\server.exe"

Write-Host "building server..."
go build -o $bin ./cmd/server

Write-Host "starting server on $addr"
$server = Start-Process -FilePath $bin -ArgumentList "-a=$addr","-f=","-i=300" -PassThru -WindowStyle Hidden

try {
    $ready = $false
    foreach ($i in 1..50) {
        try {
            Invoke-WebRequest -Uri "$base/" -UseBasicParsing -TimeoutSec 1 | Out-Null
            $ready = $true
            break
        } catch {
            Start-Sleep -Milliseconds 200
        }
    }
    if (-not $ready) { throw "server did not start" }

    Write-Host "warmup: 100 batch posts"
    go run github.com/rakyll/hey@latest -n 100 -c 10 -m POST -T "application/json" -D $batch "$base/updates/" | Out-Null

    Write-Host "load: POST /updates/ and GET /"
    $post = Start-Process -FilePath "go" -ArgumentList @(
        "run","github.com/rakyll/hey@latest",
        "-z","8s","-c","20","-m","POST","-T","application/json","-D",$batch,"$base/updates/"
    ) -PassThru -NoNewWindow
    $get = Start-Process -FilePath "go" -ArgumentList @(
        "run","github.com/rakyll/hey@latest",
        "-z","8s","-c","10","$base/"
    ) -PassThru -NoNewWindow

    Start-Sleep -Seconds 2
    Write-Host "capturing heap -> $OutFile"
    curl.exe -s -H "Accept-Encoding: identity" "$base/debug/pprof/heap" -o $OutFile

    Wait-Process -Id $post.Id, $get.Id -ErrorAction SilentlyContinue
} finally {
    if ($server -and -not $server.HasExited) {
        Stop-Process -Id $server.Id -Force
    }
}

Write-Host "saved $OutFile"
