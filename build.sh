#!/bin/bash
dir=$(pwd)
if [ ! -e "$dir/bin" ];then
  mkdir bin
fi
cd "$dir/client"
go build  -ldflags "-s -w" -o "$dir/bin/sparsefile-client"
cd "$dir/server"
go build -ldflags "-s -w"  -o "$dir/bin/sparsefile-server"

