@echo off
cd /d "%~dp0"
set "PATH=%USERPROFILE%\.cargo\bin;%PATH%"

for /f %%v in ('node -v 2^>nul') do set "NODEVER=%%v"
if not defined NODEVER (
	echo [ERROR] Node.js is not installed or not in PATH.
	echo Install Node.js 20+ and try again.
	exit /b 1
)

for /f "tokens=1 delims=." %%m in ("%NODEVER:~1%") do set "NODEMAJOR=%%m"
if %NODEMAJOR% LSS 18 (
	echo [ERROR] Your Node.js version is %NODEVER%, which is too old.
	echo This project needs Node.js 20+.
	echo If you use nvm-windows, run: nvm use 24.12.0
	exit /b 1
)

call "C:\BuildTools\Common7\Tools\VsDevCmd.bat" -arch=x64 -host_arch=x64
npm run dev
