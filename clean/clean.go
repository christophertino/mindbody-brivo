// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Nuke all Brivo users and credentials

package clean

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/christophertino/mindbody-brivo/models"
)

// Nuke is meant for cleaning up your Brivo developer environment. It will
// remove all existing users and credentials so that you can start fresh.
func Nuke(config *models.Config) {
	var (
		auth  models.Auth
		brivo models.Brivo
		creds models.CredentialList
		wg    sync.WaitGroup
	)

	// Generate Brivo access token
	if err := auth.BrivoToken.GetBrivoToken(config); err != nil {
		log.Fatalf("Error generating Brivo access token: %s", err)
	}

	// Get all Brivo users
	if err := brivo.ListUsers(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		log.Fatalln("Error fetching Brivo users", err)
	}

	// Get all Brivo credentials
	if err := creds.GetCredentials(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		log.Fatalf("Error fetching Brivo credentials: %s", err)
	}

	// Allow Brivo rate limit to reset
	time.Sleep(time.Second * 1)

	// Handle rate limiting
	semaphore := make(chan bool, config.BrivoRateLimit)

	// Loop over all users and delete
	for _, user := range brivo.Data {
		wg.Add(1)
		semaphore <- true

		// Check for valid Brivo AccessToken
		if time.Now().UTC().After(auth.BrivoToken.ExpireTime) {
			if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
				log.Fatalln("Error refreshing Brivo AUTH token", err)
			}
			fmt.Println("Refreshed Brivo AUTH token")
		}

		go func(u models.BrivoUser) {
			defer func() {
				<-semaphore
				wg.Done()
			}()
			if err := u.DeleteUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Error deleting user %d: %s\n", u.ID, err)
			}
			time.Sleep(time.Second * 1)
		}(user)
	}

	// Loop over all credentials and delete
	for _, cred := range creds.Data {
		wg.Add(1)
		semaphore <- true

		// Check for valid Brivo AccessToken
		if time.Now().UTC().After(auth.BrivoToken.ExpireTime) {
			if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
				log.Fatalln("Error refreshing Brivo AUTH token", err)
			}
			fmt.Println("Refreshed Brivo AUTH token")
		}

		go func(c models.Credential) {
			defer func() {
				<-semaphore
				wg.Done()
			}()
			if err := c.DeleteCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Error deleting credential %d: %s\n", c.ID, err)
			}
			time.Sleep(time.Second * 1)
		}(cred)
	}

	// Wait for all routines to finish and close
	wg.Wait()

	fmt.Println("Nuke completed. Check error logs for output.")
}
