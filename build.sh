#!/bin/bash
set -xve

REPO_DIR=/path/to/torii
RELEASE_DIR=/path/to/torii-release
DATE=`date +"%Y%m%d"`

# Create sub folder
mkdir -p ${RELEASE_DIR}/${DATE} || true

# Build client x86 linux binary
CGO_ENABLED=0 GOARCH="amd64" GOOS="linux" go build -o ${RELEASE_DIR}/${DATE}/client_linux_amd64_${DATE} ${REPO_DIR}/client/main.go
# Build client x86 windows binary
CGO_ENABLED=0 GOARCH="amd64" GOOS="windows" go build -o ${RELEASE_DIR}/${DATE}/client_windows_amd64_${DATE} ${REPO_DIR}/client/main.go
# Build client arm linux binary
CGO_ENABLED=0 GOARCH="arm" GOOS="linux" go build -o ${RELEASE_DIR}/${DATE}/client_linux_arm_${DATE} ${REPO_DIR}/client/main.go

# Build server x86 linux binary
CGO_ENABLED=0 GOARCH="amd64" GOOS="linux" go build -o ${RELEASE_DIR}/${DATE}/server_linux_amd64_${DATE} ${REPO_DIR}/server/main.go
# Build server x86 windows binary
CGO_ENABLED=0 GOARCH="amd64" GOOS="windows" go build -o ${RELEASE_DIR}/${DATE}/server_windows_amd64_${DATE} ${REPO_DIR}/server/main.go
# Build server arm linux binary
CGO_ENABLED=0 GOARCH="arm" GOOS="linux" go build -o ${RELEASE_DIR}/${DATE}/server_linux_arm_${DATE} ${REPO_DIR}/server/main.go
