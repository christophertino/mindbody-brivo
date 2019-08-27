// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Debug Logger

package utils

import (
	"fmt"
	"os"
)

// Logger wraps Println in a DEBUG env check
func Logger(message string) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Println(message)
	}
}
