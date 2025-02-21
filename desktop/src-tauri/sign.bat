@echo off
REM Check if the binary path parameter was provided
if "%~1"=="" (
    echo Error: No binary path provided.
    echo Usage: %~nx0 "path_to_binary"
    exit /b 1
)

REM Execute the signing command using CodeSignTool.bat with the required environment variables
CodeSignTool.bat sign ^
    -username "%CODESIGNTOOL_USERNAME%" ^
    -password "%CODESIGNTOOL_PASSWORD%" ^
    -totp_secret "%CODESIGNTOOL_TOTP_SECRET%" ^
    -credential_id "%CODESIGNTOOL_CREDENTIAL_ID%" ^
    -input_file_path "%~1" ^
    -override

exit /b %errorlevel%
