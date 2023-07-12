#!/bin/sh

# sudo apt install protobuf-compiler
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

cd pkg/agent/tunnel/
protoc -I . tunnel.proto  --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative
