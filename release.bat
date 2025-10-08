@echo off
setlocal enabledelayedexpansion

REM Windows batch script for creating and pushing Git tags with auto-versioning

REM Check if git is installed
where git >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Git is not installed. Please install git first.
    exit /b 1
)

REM Check if we're in a git repository
git rev-parse --git-dir >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Not a git repository. Please run this script from the project root.
    exit /b 1
)

REM Check for uncommitted changes
git diff-index --quiet HEAD -- >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] You have uncommitted changes. Please commit or stash them first.
    git status --short
    exit /b 1
)

REM Get the latest tag (using git tag with sorting instead of git describe)
for /f "delims=" %%i in ('git tag --sort^=-v:refname 2^>nul') do (
    set LATEST_TAG=%%i
    goto :tag_found
)
:tag_found
if "!LATEST_TAG!"=="" set LATEST_TAG=v0.0.0
echo [INFO] Latest tag: !LATEST_TAG!

REM Remove 'v' prefix if present
set LATEST_VERSION=!LATEST_TAG:v=!

REM Split version into components
for /f "tokens=1,2,3 delims=." %%a in ("!LATEST_VERSION!") do (
    set MAJOR=%%a
    set MINOR=%%b
    set PATCH=%%c
)

REM Set defaults if not found
if "!MAJOR!"=="" set MAJOR=0
if "!MINOR!"=="" set MINOR=0
if "!PATCH!"=="" set PATCH=0

REM Check if argument is provided
if "%~1"=="" (
    echo [ERROR] Usage: %~nx0 ^<major^|minor^|patch^|x.y.z^>
    echo.
    echo Examples:
    echo   %~nx0 patch   # Increment patch version ^(e.g., 1.7.0 -^> 1.7.1^)
    echo   %~nx0 minor   # Increment minor version ^(e.g., 1.7.0 -^> 1.8.0^)
    echo   %~nx0 major   # Increment major version ^(e.g., 1.7.0 -^> 2.0.0^)
    echo   %~nx0 1.8.5   # Set specific version
    exit /b 1
)

REM Determine the new version based on input
set INCREMENT_TYPE=%~1

if /i "!INCREMENT_TYPE!"=="major" (
    set /a NEW_MAJOR=!MAJOR!+1
    set NEW_MINOR=0
    set NEW_PATCH=0
) else if /i "!INCREMENT_TYPE!"=="minor" (
    set NEW_MAJOR=!MAJOR!
    set /a NEW_MINOR=!MINOR!+1
    set NEW_PATCH=0
) else if /i "!INCREMENT_TYPE!"=="patch" (
    set NEW_MAJOR=!MAJOR!
    set NEW_MINOR=!MINOR!
    set /a NEW_PATCH=!PATCH!+1
) else (
    REM Check if argument is a valid version number (x.y.z)
    echo !INCREMENT_TYPE! | findstr /r "^[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*$" >nul
    if !errorlevel! equ 0 (
        for /f "tokens=1,2,3 delims=." %%a in ("!INCREMENT_TYPE!") do (
            set NEW_MAJOR=%%a
            set NEW_MINOR=%%b
            set NEW_PATCH=%%c
        )
    ) else (
        echo [ERROR] Invalid argument: !INCREMENT_TYPE!
        echo [ERROR] Use 'major', 'minor', 'patch', or a specific version ^(e.g., 1.8.5^)
        exit /b 1
    )
)

set NEW_VERSION=!NEW_MAJOR!.!NEW_MINOR!.!NEW_PATCH!
set NEW_TAG=v!NEW_VERSION!

echo [INFO] New version: !NEW_TAG!

REM Confirm before proceeding
set /p CONFIRM="Create and push tag !NEW_TAG!? [y/N] "
if /i not "!CONFIRM!"=="y" (
    echo [WARN] Cancelled by user.
    exit /b 0
)

REM Create annotated tag
echo [INFO] Creating annotated tag !NEW_TAG!...
git tag -a "!NEW_TAG!" -m "Release !NEW_TAG!"
if %errorlevel% neq 0 (
    echo [ERROR] Failed to create tag !NEW_TAG!
    exit /b 1
)

REM Push tag to remote
echo [INFO] Pushing tag !NEW_TAG! to origin...
git push origin "!NEW_TAG!"
if %errorlevel% neq 0 (
    echo [ERROR] Failed to push tag !NEW_TAG!
    exit /b 1
)

echo [INFO] Done! GitHub Actions workflow will be triggered automatically.
echo [INFO] Check the release at: https://github.com/TheTitanrain/kerioMirrorGo/releases

endlocal
