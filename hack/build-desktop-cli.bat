@echo off

set GOOS=windows
set GOARCH=amd64

go build -ldflags "-s -w" -o test/devpod-x86_64-pc-windows-msvc.exe

xcopy /F /Y test\devpod-x86_64-pc-windows-msvc.exe desktop\bin\*
