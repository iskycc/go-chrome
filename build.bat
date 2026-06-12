@echo off
setlocal EnableExtensions EnableDelayedExpansion

echo Building go-chrome for Windows...

set "TOOLS_DIR=%~dp0.tools"
set "MINGW_BIN=%TOOLS_DIR%\mingw64\bin"
if exist "%MINGW_BIN%\gcc.exe" set "PATH=%MINGW_BIN%;%PATH%"

set "GO_VERSION=1.26.0"
set "GO_ARCHIVE=go%GO_VERSION%.windows-amd64.zip"
set "GO_DOWNLOAD_URL=https://golang.google.cn/dl/%GO_ARCHIVE%"
set "TOOLS_DIR=%~dp0.tools"
set "LOCAL_GO_DIR=%TOOLS_DIR%\go"
set "LOCAL_GO_BIN=%LOCAL_GO_DIR%\bin\go.exe"

where go >nul 2>nul
if errorlevel 1 (
    echo Go SDK not found in PATH.
    echo Installing Go SDK %GO_VERSION% from China mirror: %GO_DOWNLOAD_URL%

    if not exist "%TOOLS_DIR%" mkdir "%TOOLS_DIR%"

    powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $url='%GO_DOWNLOAD_URL%'; $zip=Join-Path '%TOOLS_DIR%' '%GO_ARCHIVE%'; Write-Host ('Downloading ' + $url); Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $zip; if (Test-Path '%LOCAL_GO_DIR%') { Remove-Item -Recurse -Force '%LOCAL_GO_DIR%' }; Expand-Archive -Path $zip -DestinationPath '%TOOLS_DIR%' -Force"
    if errorlevel 1 (
        echo ERROR: Go SDK download or extraction failed
        exit /b 1
    )

    if not exist "%LOCAL_GO_BIN%" (
        echo ERROR: Go SDK install failed, go.exe not found: %LOCAL_GO_BIN%
        exit /b 1
    )

    set "PATH=%LOCAL_GO_DIR%\bin;%PATH%"
) else (
    for /f "delims=" %%v in ('go version') do echo Found %%v
)

where go >nul 2>nul
if errorlevel 1 (
    echo ERROR: Go SDK is still unavailable
    exit /b 1
)

set GO111MODULE=on
set GOPROXY=https://goproxy.cn,direct
set GOSUMDB=sum.golang.google.cn

echo Using GOPROXY=%GOPROXY%
echo Downloading Go module dependencies from China mirror...
go mod download
if errorlevel 1 (
    echo ERROR: go mod download failed
    exit /b 1
)

echo Building executable...
go build -mod=readonly -ldflags "-H=windowsgui -s -w" -o go-chrome.exe ./cmd/go-chrome
if errorlevel 1 (
    echo ERROR: Build failed
    exit /b 1
)

echo Build complete: go-chrome.exe
endlocal
