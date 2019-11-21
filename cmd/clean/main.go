// Brivo Account Cleanup
//
// Use this application to clear all existing users
// and credentials from your Brivo account.
//
// Copyright 2019 Christopher Tino. All rights reserved.

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

	fmt.Println("WARNING: This will delete all Brivo user data.")
	fmt.Println("Enter [1] to delete Members only. Enter [2] to delete all users.")
	fmt.Print("-> ")

	// Prompt user for confirmation
	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		log.Fatalln("Reader error:", err)
	}

	if char == '1' || char == '2' {
		clean.Nuke(&config, char)
	} else {
		fmt.Println("Input not recognized")
	}
}
