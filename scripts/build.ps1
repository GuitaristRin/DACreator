# DACreator v3 构建脚本
# 用法：scripts\build.ps1 [-Version x.y.z]
# 产物：build/DACreator_v<Version>_Windows_x64.zip（内含二合一 DACreator.exe 与资产）
param(
    [string]$Version = ""
)
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if (-not $Version) {
    $Version = (git describe --tags --always 2>$null)
    if (-not $Version) { $Version = "0.0.0-dev" }
}
Write-Host "==> DACreator build $Version" -ForegroundColor Cyan

New-Item -ItemType Directory -Force -Path build | Out-Null

Write-Host "==> [1/3] 构建引擎（Go）" -ForegroundColor Cyan
go build -trimpath -ldflags "-s -w -X main.version=$Version" -o build/dac.exe ./cmd/dac
if ($LASTEXITCODE -ne 0) { throw "go build 失败" }

Write-Host "==> [2/3] 构建 GUI（Rust，内嵌引擎）" -ForegroundColor Cyan
$env:DAC_EMBED_ENGINE = (Resolve-Path build/dac.exe).Path
$env:DAC_EMBED_VERSION = $Version
cargo build --release --manifest-path gui/Cargo.toml
if ($LASTEXITCODE -ne 0) { throw "cargo build 失败" }
Remove-Item Env:DAC_EMBED_ENGINE, Env:DAC_EMBED_VERSION

Write-Host "==> [3/3] 组装便携包" -ForegroundColor Cyan
$stage = "build/stage"
if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
New-Item -ItemType Directory -Force -Path $stage | Out-Null
Copy-Item gui/target/release/dacreator-gui.exe "$stage/DACreator.exe"
Copy-Item build/dac.exe $stage
Copy-Item -Recurse assets $stage/assets
$zip = "build/DACreator_v${Version}_Windows_x64.zip"
if (Test-Path $zip) { Remove-Item -Force $zip }
Compress-Archive -Path "$stage/*" -DestinationPath $zip
Remove-Item -Recurse -Force $stage

Write-Host "==> 完成：$zip" -ForegroundColor Green
Write-Host "    DACreator.exe = GUI + CLI 二合一（引擎已内嵌，亦可直接使用 dac.exe）"
