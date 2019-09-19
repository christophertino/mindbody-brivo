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
		auth         models.Auth
		brivo        models.Brivo
		creds        models.CredentialList
		customFields models.CustomFields
		wg           sync.WaitGroup
	)

	// Generate Brivo access token
	if err := auth.BrivoToken.GetBrivoToken(config); err != nil {
		log.Fatalf("Error generating Brivo access token: %s", err)
	}

	// Get all Brivo users from Member Group
	if err := brivo.ListUsersWithinGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
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

	fmt.Println("Deleteing all Brivo users...")

	// Keep track of which users we deleted
	var brivoIDSet = make(map[string]bool)

	// Loop over all users and delete
	for _, user := range brivo.Data {
		// Check for valid Brivo AccessToken
		if time.Now().UTC().After(auth.BrivoToken.ExpireTime) {
			if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
				log.Fatalln("Error refreshing Brivo AUTH token", err)
			}
			fmt.Println("Refreshed Brivo AUTH token")
		}

		wg.Add(1)
		semaphore <- true
		// Get custom fields for user
		go func(u models.BrivoUser) {
			defer func() {
				<-semaphore
				wg.Done()
			}()
			if err := customFields.GetCustomFieldsForUser(u.ID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Error fetching custom fields for user %d: %s\n", u.ID, err)
				return
			}
		}(user)

		wg.Wait()

		// Stash Barcode ID in a set so we can check it later against Credential.ReferenceID
		barcodeID, err := models.GetFieldValue(config.BrivoBarcodeFieldID, customFields.Data)
		if err != nil {
			<-semaphore
			wg.Done()
			fmt.Printf("Skipping user %d with error: %s\n", user.ID, err)
			continue
		}
		brivoIDSet[barcodeID] = true

		wg.Add(1)
		semaphore <- true
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

	fmt.Println("Deleteing all Brivo credentials...")

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

		// Make sure this credential belongs to a user we are deleteing (in the Member group only)
		if brivoIDSet[cred.ReferenceID] != true {
			<-semaphore
			wg.Done()
			continue
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
