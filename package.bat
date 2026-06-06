@echo off
setlocal

echo Packaging go-chrome for Windows...

REM Ensure go-chrome.exe exists
if not exist go-chrome.exe (
    echo ERROR: go-chrome.exe not found. Run build.bat first.
    exit /b 1
)

REM Create package directory
set PKGDIR=go-chrome-release
if exist %PKGDIR% rmdir /s /q %PKGDIR%
mkdir %PKGDIR%

REM Copy executable
copy go-chrome.exe %PKGDIR%\

REM Copy documentation
copy README.md %PKGDIR%\
copy FAQ.md %PKGDIR%\
copy USER_GUIDE.md %PKGDIR%\

REM Create data directory structure
mkdir %PKGDIR%\data\flows
mkdir %PKGDIR%\logs

REM Create chrome directory placeholder
mkdir %PKGDIR%\chrome

REM Package as zip
set ZIPNAME=go-chrome-release.zip
if exist %ZIPNAME% del %ZIPNAME%
powershell -Command "Compress-Archive -Path %PKGDIR%\* -DestinationPath %ZIPNAME%"

echo Package created: %ZIPNAME%
endlocal
