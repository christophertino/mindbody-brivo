// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Brivo Account Cleanup
// Use this application to clear all existing users
// and credentials from your Brivo account.

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/christophertino/mindbody-brivo/clean"
	"github.com/christophertino/mindbody-brivo/models"
)

func main() {
	var config models.Config
	config.GetConfig()

	fmt.Println("This will delete all Brivo user data. Are you sure? (y/n)")
	fmt.Print("-> ")

	// Prompt user for confirmation
	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		log.Fatalln("Reader error:", err)
	}

	if char == 'y' || char == 'Y' {
		clean.Nuke(&config)
	}
}
