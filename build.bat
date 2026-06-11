@echo off
setlocal EnableExtensions EnableDelayedExpansion

echo Building go-chrome for Windows...

set "GO_VERSION=1.26.0"
set "GO_ARCHIVE=go%GO_VERSION%.windows-amd64.zip"
set "GO_DOWNLOAD_URL=https://golang.google.cn/dl/%GO_ARCHIVE%"
set "TOOLS_DIR=%~dp0.tools"
set "LOCAL_GO_DIR=%TOOLS_DIR%\go"
set "LOCAL_GO_BIN=%LOCAL_GO_DIR%\bin\go.exe"

:: MinGW-w64 from GitHub (via ghproxy)
set "MINGW_ZIP_FILE=mingw64.7z"
set "MINGW_DOWNLOAD_URL=https://ghfast.top/https://github.com/niXman/mingw-builds-binaries/releases/download/14.2.0-rt_v12-rev0/x86_64-14.2.0-release-posix-seh-ucrt-rt_v12-rev0.7z"
set "LOCAL_MINGW_DIR=%TOOLS_DIR%\mingw64"
set "LOCAL_MINGW_BIN=%LOCAL_MINGW_DIR%\bin"

:: Check and install Go SDK
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

:: Check and install MinGW for CGO
where gcc >nul 2>nul
if errorlevel 1 (
    echo GCC not found in PATH. MinGW is required for building Fyne applications.

    if not exist "%LOCAL_MINGW_DIR%\bin\gcc.exe" (
        echo Installing MinGW from ghproxy mirror...
        echo URL: %MINGW_DOWNLOAD_URL%
        
        :: Download 7zr.exe
        if not exist "%TOOLS_DIR%\7zr.exe" (
            echo Downloading 7zr.exe...
            powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $url='https://www.7-zip.org/a/7zr.exe'; $out='%TOOLS_DIR%\7zr.exe'; Write-Host ('Downloading ' + $url); Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $out"
        )

        :: Download MinGW
        echo Downloading MinGW...
        powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $url='%MINGW_DOWNLOAD_URL%'; $out='%TOOLS_DIR%\%MINGW_ZIP_FILE%'; Write-Host ('Downloading ' + $url); Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $out"
        if errorlevel 1 (
            echo Failed with ghfast, trying ghproxy.com...
            set "MINGW_DOWNLOAD_URL=https://ghproxy.com/https://github.com/niXman/mingw-builds-binaries/releases/download/14.2.0-rt_v12-rev0/x86_64-14.2.0-release-posix-seh-ucrt-rt_v12-rev0.7z"
            echo Trying %MINGW_DOWNLOAD_URL%
            powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $url='%MINGW_DOWNLOAD_URL%'; $out='%TOOLS_DIR%\%MINGW_ZIP_FILE%'; Write-Host ('Downloading ' + $url); Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $out"
        )
        if errorlevel 1 (
            echo ERROR: MinGW download failed from all mirrors
            exit /b 1
        )

        :: Extract MinGW
        echo Extracting MinGW...
        "%TOOLS_DIR%\7zr.exe" x "%TOOLS_DIR%\%MINGW_ZIP_FILE%" -o"%TOOLS_DIR%" -y
        if errorlevel 1 (
            echo ERROR: MinGW extraction failed
            exit /b 1
        )
        
        del "%TOOLS_DIR%\%MINGW_ZIP_FILE%" 2>nul
    )

    if not exist "%LOCAL_MINGW_DIR%\bin\gcc.exe" (
        echo ERROR: MinGW install failed, gcc.exe not found: %LOCAL_MINGW_DIR%\bin\gcc.exe
        exit /b 1
    )

    set "PATH=%LOCAL_MINGW_BIN%;%PATH%"
    echo MinGW installed at: %LOCAL_MINGW_DIR%
) else (
    for /f "delims=" %%v in ('gcc --version 2^>^&1 ^| findstr /i gcc') do echo Found %%v
)

where go >nul 2>nul
if errorlevel 1 (
    echo ERROR: Go SDK is still unavailable
    exit /b 1
)

where gcc >nul 2>nul
if errorlevel 1 (
    echo ERROR: GCC is still unavailable
    exit /b 1
)

set GO111MODULE=on
set GOPROXY=https://goproxy.cn,direct
set GOSUMDB=sum.golang.google.cn
set CGO_ENABLED=1

echo Using GOPROXY=%GOPROXY%
echo CGO_ENABLED=%CGO_ENABLED%
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
