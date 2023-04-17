@echo off

set GOOS=windows
set GOARCH=amd64

go build -ldflags "-s -w" -o test/devpod-windows-amd64

xcopy /F /Y test\devpod-windows-amd64 desktop\bin\*