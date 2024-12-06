#!/bin/bash
# 提取当前的 commit ID
commit=$(git rev-parse HEAD)
# 提取当前的分支名称
branch=$(git rev-parse --abbrev-ref HEAD)

app=sshtunnel
go mod tidy
go env -w CGO_ENABLED=0 GOOS=windows GOARCH=amd64
go build -ldflags="-s -w -X main.commit=${commit} -X main.branch=${branch}" -o $app.exe  -buildvcs=false
upx $app.exe
go env -w CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go build -ldflags="-s -w -X main.commit=${commit} -X main.branch=${branch}" -o $app  -buildvcs=false
upx $app
