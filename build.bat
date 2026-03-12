@echo off
REM Сборка Wails-приложения с правильными build tags (без установки wails CLI)
go build -tags "desktop,production" -ldflags "-w -s -H windowsgui" -o "FCS AutoReport.exe" .
if %ERRORLEVEL% equ 0 (
    echo OK: FCS AutoReport.exe
) else (
    echo Build failed
    exit /b 1
)
