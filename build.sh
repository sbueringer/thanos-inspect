#!/bin/bash

WORKDIR=`echo $0 | sed -e s/build.sh//`
cd ${WORKDIR}

TRAVIS_BUILD_DIR=${TRAVIS_BUILD_DIR:-"."}

echo "Building linux binary: thanos-inspect with env variables:"
env | grep GO
go build -ldflags='-s -w' -v -o $TRAVIS_BUILD_DIR/thanos-inspect

export GOOS=windows
export GOARCH=amd64

echo "Building windows binary: kubectx.exe with env variables:"
env | grep GO
go build -ldflags='-s -w' -v -o $TRAVIS_BUILD_DIR/thanos-inspect.exe

echo "Downloading upx"
cd $TRAVIS_BUILD_DIR
curl -L -O https://github.com/upx/upx/releases/download/v3.93/upx-3.93-amd64_linux.tar.xz
tar xvf upx-3.93-amd64_linux.tar.xz

echo "Using upx on kubectx"
upx-3.93-amd64_linux/upx $TRAVIS_BUILD_DIR/thanos-inspect

echo "Using upx on kubectx.exe"
upx-3.93-amd64_linux/upx $TRAVIS_BUILD_DIR/thanos-inspect.exe
