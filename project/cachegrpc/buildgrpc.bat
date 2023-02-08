@echo off

REM go install google.golang.org/protobuf
REM go install github.com/golang/protobuf/protoc-gen-go
REM go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

set OLDPATH=%PATH%
set PATH=%GOPATH%\bin;%PATH%
..\..\protoc\bin\protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative cache.proto
set PATH=%OLDPATH%

