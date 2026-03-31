@chcp 65001 >nul
@echo off
setlocal

rem Check if running as administrator
net session >nul 2>&1
if errorlevel 1 goto :notadmin

rem install.bat: add explorer right-click menu
set INSTALL_DIR=%~dp0
set INSTALL_DIR=%INSTALL_DIR:~0,-1%

echo installing mdify: %INSTALL_DIR%

reg add "HKLM\Software\Classes\Directory\Background\shell\mdify" /ve /d "クリップボードをMarkdownに変換(mdify)" /f >nul 2>&1
reg add "HKLM\Software\Classes\Directory\Background\shell\mdify" /v Icon /d "%INSTALL_DIR%\\mdify.exe" /f >nul
reg add "HKLM\Software\Classes\Directory\Background\shell\mdify\command" /ve /d "\"%INSTALL_DIR%\\mdify.exe\"" /f >nul

echo Added mdify to explorer right-click menu.
echo Completed installing mdify.
pause
exit /b 0

:notadmin
echo.
echo You need administrator privileges to install mdify.
echo To add mdify to the explorer right-click menu, you need to run the installer with administrator privileges.
echo.
set /p confirm=Do you want to elevate privileges? (Y/n) [Y]: 
if /i "%confirm%"=="" goto :elevate
if /i "%confirm%"=="Y" goto :elevate
if /i "%confirm%"=="N" (
    echo Installation cancelled.
    pause
    exit /b 1
)
goto :elevate

:elevate
powershell -NoProfile -Command "Start-Process '%~f0' -Verb RunAs"
exit /b 0

endlocal