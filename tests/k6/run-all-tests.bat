@echo off
setlocal enabledelayedexpansion

:: Base URL
if "%BASE_URL%"=="" set BASE_URL=http://localhost:8080

echo ========================================
echo 🚀 k6 Load Testing Suite
echo ========================================
echo Target: %BASE_URL%
echo.

:: Check if k6 is installed
where k6 >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ k6 is not installed!
    echo Please install k6 first:
    echo   - Windows (Chocolatey^): choco install k6
    echo   - Windows (Scoop^): scoop install k6
    echo   - Or download from: https://github.com/grafana/k6/releases
    exit /b 1
)

echo ✓ k6 is installed
echo.

:: Check if service is running
echo Checking if service is running...
curl -s "%BASE_URL%/health" >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Service is not responding at %BASE_URL%
    echo Please start the service first:
    echo   make run
    exit /b 1
)

echo ✓ Service is running
echo.

:: Create results directory
if not exist results mkdir results

:: Get timestamp
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /format:list') do set datetime=%%I
set TIMESTAMP=%datetime:~0,8%_%datetime:~8,6%

echo Starting test suite...
echo.

set FAILED_TESTS=0

:: Test 1: Validate Token
echo ========================================
echo Running: validate-token
echo ========================================
k6 run -e BASE_URL=%BASE_URL% --out json=results/validate-token_%TIMESTAMP%.json validate-token.js
if %errorlevel% neq 0 (
    echo ✗ validate-token failed
    set /a FAILED_TESTS+=1
) else (
    echo ✓ validate-token completed successfully
)
echo.
timeout /t 5 /nobreak >nul

:: Test 2: Login
echo ========================================
echo Running: login
echo ========================================
k6 run -e BASE_URL=%BASE_URL% --out json=results/login_%TIMESTAMP%.json login.js
if %errorlevel% neq 0 (
    echo ✗ login failed
    set /a FAILED_TESTS+=1
) else (
    echo ✓ login completed successfully
)
echo.
timeout /t 5 /nobreak >nul

:: Test 3: Register
echo ========================================
echo Running: register
echo ========================================
k6 run -e BASE_URL=%BASE_URL% --out json=results/register_%TIMESTAMP%.json register.js
if %errorlevel% neq 0 (
    echo ✗ register failed
    set /a FAILED_TESTS+=1
) else (
    echo ✓ register completed successfully
)
echo.
timeout /t 5 /nobreak >nul

:: Test 4: Mixed Load
echo ========================================
echo Running: mixed-load
echo ========================================
k6 run -e BASE_URL=%BASE_URL% --out json=results/mixed-load_%TIMESTAMP%.json mixed-load.js
if %errorlevel% neq 0 (
    echo ✗ mixed-load failed
    set /a FAILED_TESTS+=1
) else (
    echo ✓ mixed-load completed successfully
)
echo.

:: Summary
echo ========================================
echo 📊 TEST SUITE COMPLETE
echo ========================================

if %FAILED_TESTS%==0 (
    echo ✓ All tests passed!
    echo Results saved in: ./results/
    exit /b 0
) else (
    echo ✗ %FAILED_TESTS% test(s^) failed
    echo Check the results in: ./results/
    exit /b 1
)
