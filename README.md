# apkparser

[![GoDoc](https://godoc.org/github.com/v4n0v/apkparser?status.svg)](https://godoc.org/github.com/v4n0v/apkparser)
[![Build Status](https://travis-ci.org/appflight/apkparser.svg?branch=master)](https://travis-ci.org/appflight/apkparser)

APK AndroidManifest.xml and resources.arsc parsing.

**Works with Go 1.9 or higher.**

Documentation on [GoDoc](https://godoc.org/github.com/v4n0v/apkparser)

    go get github.com/v4n0v/apkparser

## ZipReader
Because Android can handle even broken ZIP archives, this packages has it's own zip reader,
based on archive/zip.

## axml2xml
A tool to extract AndroidManifest.xml and verify APK signature is also part of this repo.

    go get github.com/v4n0v/apkparser
    go install github.com/v4n0v/apkparser/axml2xml
    ./axml2xml -v application.apk

## Example

```go
package main

import (
	"fmt"
	"github.com/v4n0v/apkparser"
	"os"
)

func main() {
	apkInfo, err := apkparser.ParseApk(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open the APK: %s", err)
		os.Exit(1)
		return
	}
	fmt.Println(apkInfo)
}
```
