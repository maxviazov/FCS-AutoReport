@echo off
REM Один запуск — ставит Go, Wails, NSIS и прописывает их в PATH (через winget + go install).
REM Запуск: двойной клик или из терминала.
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-build-env.ps1"
pause
