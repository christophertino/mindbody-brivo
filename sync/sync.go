// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Migrate all MINDBODY clients to Brivo as new users

package sync

import (
	"fmt"
	"log"
	"sync"

	"github.com/christophertino/mindbody-brivo/models"
)

var (
	auth  models.Auth
	mb    models.MindBody
	brivo models.Brivo
)

// GetAllUsers will fetch all existing users from MINDBODY and Brivo
func GetAllUsers(config *models.Config) {
	if err := auth.Authenticate(config); err != nil {
		fmt.Println("Error generating AUTH tokens:", err)
		return
	}

	var wg sync.WaitGroup

	// Get all MINDBODY clients
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := mb.GetClients(*config, auth.MindBodyToken.AccessToken); err != nil {
			log.Fatalln("Error fetching MINDBODY clients", err)
		}
	}()

	// Get existing Brivo users
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := brivo.ListUsers(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
			log.Fatalln("Error fetching Brivo users", err)
		}
	}()

	wg.Wait()

	// fmt.Printf("MindBody Model: %+v\n Brivo Model: %+v\n", mb, brivo)

	// Map existing user data from MINDBODY to Brivo
	brivo.CreateUsers(mb, *config, auth)
}
