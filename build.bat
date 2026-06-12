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

REM Ensure JetBrains-style CJK UI fonts are present. The .ttf files are not
REM committed to the repo; they are fetched on first build via a China mirror.
set "FONT_DIR=%~dp0assets\fonts"
set "FONT_REGULAR=%FONT_DIR%\MapleMono-CN-Regular.ttf"
set "FONT_MEDIUM=%FONT_DIR%\MapleMono-CN-Medium.ttf"
if not exist "%FONT_REGULAR%" (
    echo Maple Mono CN Regular font missing; downloading from China mirror...
    set "FONT_ZIP=%TOOLS_DIR%\MapleMono-CN.zip"
    set "FONT_URL=https://gh-proxy.com/https://github.com/subframe7536/maple-font/releases/download/v7.9/MapleMono-CN.zip"
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $url='%FONT_URL%'; $zip='%FONT_ZIP%'; Write-Host ('Downloading ' + $url); Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $zip; if (Test-Path '%FONT_DIR%\tmp-fonts') { Remove-Item -Recurse -Force '%FONT_DIR%\tmp-fonts' }; $null=New-Item -ItemType Directory -Path '%FONT_DIR%\tmp-fonts'; Expand-Archive -Path $zip -DestinationPath '%FONT_DIR%\tmp-fonts' -Force; Copy-Item '%FONT_DIR%\tmp-fonts\MapleMono-CN-Regular.ttf' '%FONT_REGULAR%' -Force; Copy-Item '%FONT_DIR%\tmp-fonts\MapleMono-CN-Medium.ttf' '%FONT_MEDIUM%' -Force; Remove-Item -Recurse -Force '%FONT_DIR%\tmp-fonts'"
    if errorlevel 1 (
        echo ERROR: failed to download Maple Mono CN fonts from %FONT_URL%
        echo Please place the following files manually in %FONT_DIR%:
        echo   MapleMono-CN-Regular.ttf
        echo   MapleMono-CN-Medium.ttf
        exit /b 1
    )
    echo Fonts downloaded to %FONT_DIR%
)
if not exist "%FONT_MEDIUM%" (
    echo ERROR: Maple Mono CN Medium font missing: %FONT_MEDIUM%
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
