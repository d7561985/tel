// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.

package main

import (
	"hotrod/cmd"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	cmd.Execute()
}
