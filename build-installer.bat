@echo off
REM Полноценный установщик: один exe (FCS_AutoReport_Setup.exe). Запустил — всё ставится, включая WebView2.
setlocal
set WAILS=wails
where wails >nul 2>nul
if %ERRORLEVEL% neq 0 (
    if exist "%USERPROFILE%\go\bin\wails.exe" set WAILS=%USERPROFILE%\go\bin\wails.exe
    if not exist "%WAILS%" (
        echo Ошибка: wails не найден. Запустите setup-build-env.bat
        exit /b 1
    )
)
where makensis >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Ошибка: NSIS не найден. Запустите setup-build-env.bat
    exit /b 1
)

echo [1/3] Сборка приложения и загрузка WebView2...
"%WAILS%" build -webview2 embed -nsis
if %ERRORLEVEL% neq 0 (
    echo Сборка завершилась с ошибкой.
    exit /b 1
)

echo [2/3] Подмена на полноценный установщик (тихая установка WebView2)...
copy /Y "%~dp0installer\project.nsi" "build\windows\installer\project.nsi" >nul
if not exist "build\windows\installer\tmp\MicrosoftEdgeWebview2Setup.exe" (
    echo Ошибка: WebView2 bootstrapper не найден в build\windows\installer\tmp\
    exit /b 1
)

echo [3/3] Сборка итогового установщика...
cd build\windows\installer
makensis -DAPP_EXE="..\..\bin\FCS AutoReport.exe" project.nsi
cd ..\..\..
if %ERRORLEVEL% neq 0 (
    echo Ошибка makensis.
    exit /b 1
)

echo.
echo Готово. Для распространения используйте ОДИН файл:
echo   build\bin\FCS_AutoReport_Setup.exe
echo.
echo Пользователь: запускает его — Далее, Установить — всё ставится, ничего дополнительно не нужно.
dir /b build\bin\FCS_AutoReport_Setup.exe 2>nul
endlocal
