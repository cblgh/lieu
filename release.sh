#!/bin/bash

for os in linux darwin openbsd
do
    executable="lieu"
    if [ $os = "windows" ]; then
        executable="lieu.exe"
    fi
    env GOOS="$os" go build -tags fts5 -ldflags "-s -w"
    tar czf "lieu-$os.tar.gz" README.md html/ data/ lieu.toml "$executable"
    echo "lieu-$os.tar.gz"
    rm -f "$executable"
done
    
