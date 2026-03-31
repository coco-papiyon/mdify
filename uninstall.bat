@chcp 65001 >nul
@echo off
setlocal

rem Check if running as administrator
net session >nul 2>&1
if errorlevel 1 goto :notadmin

echo uninstalling mdify

reg delete "HKLM\Software\Classes\Directory\Background\shell\mdify" /f >nul 2>&1

echo Completed uninstalling mdify.
pause
exit /b 0

:notadmin
echo.
echo You need administrator privileges to uninstall mdify.
echo To remove mdify from the explorer right-click menu, you need to run the uninstaller with administrator privileges.
echo.
set /p confirm=Do you want to elevate privileges? (Y/n) [Y]: 
if /i "%confirm%"=="" goto :elevate
if /i "%confirm%"=="Y" goto :elevate
if /i "%confirm%"=="N" (
    echo Uninstallation cancelled.
    pause
    exit /b 1
)
goto :elevate

:elevate
powershell -NoProfile -Command "Start-Process '%~f0' -Verb RunAs"
exit /b 0

endlocal