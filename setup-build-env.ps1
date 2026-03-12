# Автоматическая установка всего нужного для сборки: Go, Wails, NSIS и добавление в PATH.
# Запуск: правый клик -> "Выполнить в PowerShell" или: powershell -ExecutionPolicy Bypass -File setup-build-env.ps1

$ErrorActionPreference = "Stop"

function Add-ToUserPath {
    param([string]$Dir)
    if (-not $Dir -or -not (Test-Path $Dir)) { return }
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -split ";" -contains $Dir) { return }
    [Environment]::SetEnvironmentVariable("Path", $userPath.TrimEnd(";") + ";" + $Dir, "User")
    $env:Path = $env:Path + ";" + $Dir
    Write-Host "  + PATH (пользователь): $Dir" -ForegroundColor Green
}

function Refresh-PathFromRegistry {
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
}

Write-Host "`n=== Установка среды для сборки FCS AutoReport ===`n" -ForegroundColor Cyan

# 1) Winget
if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    Write-Host "Ошибка: winget не найден. Установите App Installer (Windows 10/11) или Visual Studio." -ForegroundColor Red
    exit 1
}

# 2) Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Установка Go (winget)..." -ForegroundColor Yellow
    winget install --id GoLang.Go -e --accept-package-agreements --accept-source-agreements
    Refresh-PathFromRegistry
    $goBinCandidates = @("$env:ProgramFiles\Go\bin", "C:\Program Files\Go\bin", "$env:LOCALAPPDATA\Programs\Go\bin")
    foreach ($p in $goBinCandidates) {
        if (Test-Path $p) {
            $env:Path = $env:Path + ";" + $p
            Add-ToUserPath -Dir $p
            break
        }
    }
}
if (Get-Command go -ErrorAction SilentlyContinue) {
    Write-Host "Go: $(go version)" -ForegroundColor Green
} else {
    Write-Host "Go не найден после установки. Добавьте вручную в PATH и перезапустите скрипт." -ForegroundColor Red
    exit 1
}

# 3) GOPATH / go\bin в PATH (для wails и других go install)
$goBin = Join-Path $env:USERPROFILE "go\bin"
if (-not (Test-Path (Split-Path $goBin))) {
    New-Item -ItemType Directory -Path $goBin -Force | Out-Null
}
Add-ToUserPath -Dir $goBin

# 4) Wails CLI
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    Write-Host "Установка Wails CLI (go install)..." -ForegroundColor Yellow
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    Refresh-PathFromRegistry
    $env:Path = $env:Path + ";" + $goBin
}
if (Get-Command wails -ErrorAction SilentlyContinue) {
    Write-Host "Wails: установлен (wails)" -ForegroundColor Green
} else {
    Write-Host "Wails не в PATH. Добавьте в PATH: $goBin" -ForegroundColor Yellow
}

# 5) NSIS
if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) {
    Write-Host "Установка NSIS (winget)..." -ForegroundColor Yellow
    winget install --id NSIS.NSIS -e --accept-package-agreements --accept-source-agreements
    Refresh-PathFromRegistry
    $nsisPaths = @("$env:ProgramFiles\NSIS", "${env:ProgramFiles(x86)}\NSIS")
    foreach ($p in $nsisPaths) {
        if (Test-Path $p) {
            Add-ToUserPath -Dir $p
            break
        }
    }
}
if (Get-Command makensis -ErrorAction SilentlyContinue) {
    Write-Host "NSIS: установлен (makensis)" -ForegroundColor Green
} else {
    Write-Host "NSIS не найден в PATH. Для установщика добавьте вручную папку NSIS в PATH." -ForegroundColor Yellow
}

Write-Host "`nГотово. Закройте и снова откройте терминал (или Cursor), чтобы PATH обновился.`n" -ForegroundColor Cyan
Write-Host "Дальше: build-installer.bat  — собрать установщик приложения." -ForegroundColor White
