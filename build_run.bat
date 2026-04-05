@echo off
set GOARCH=amd64
echo Building...
go build -o .\build\x-tymus.exe -mod=vendor && cls && .\build\x-tymus.exe -p ./phishlets -t ./redirectors -developer -debug
