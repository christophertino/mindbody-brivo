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

type brivoIDSet struct {
	access sync.Mutex
	ids    map[string]bool
}

var (
	auth      models.Auth
	brivo     models.Brivo
	creds     models.CredentialList
	config    *models.Config
	semaphore chan bool
	wg        sync.WaitGroup
	brivoIDs  brivoIDSet
)

// Nuke is meant for cleaning up your Brivo developer environment. It will
// remove all existing users and credentials so that you can start fresh.
func Nuke(cfg *models.Config) {
	config = cfg

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
	semaphore = make(chan bool, config.BrivoRateLimit)

	// Keep track of which users we deleted
	brivoIDs.ids = make(map[string]bool)

	fmt.Println("Deleteing all Brivo users...")

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
		go func(u models.BrivoUser) {
			defer wg.Done()

			customFields, err := getCustomFields(u.ID)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Stash Barcode ID in a set so we can check it later against Credential.ReferenceID
			barcodeID, err := models.GetFieldValue(config.BrivoBarcodeFieldID, customFields.Data)
			if err != nil {
				fmt.Printf("Skipping user %d with error: %s\n", u.ID, err)
				return
			}
			brivoIDs.update(barcodeID)

			if err := deleteUser(u); err != nil {
				fmt.Println(err)
			}

			// Reset rate limit
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

		go func(c models.Credential) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			// Make sure this credential belongs to a user we are deleteing (in the Member group only)
			if brivoIDs.ids[c.ReferenceID] == true {
				if err := c.DeleteCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
					fmt.Printf("Error deleting credential %d: %s\n", c.ID, err)
				}
				time.Sleep(time.Second * 1)
			}
		}(cred)
	}

	// Wait for all routines to finish and close
	wg.Wait()

	fmt.Println("Nuke completed. Check error logs for output.")
}

// Get user's custom fields
func getCustomFields(userID int) (models.CustomFields, error) {
	semaphore <- true
	defer func() {
		<-semaphore
	}()
	var customFields models.CustomFields
	if err := customFields.GetCustomFieldsForUser(userID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return customFields, fmt.Errorf("Error fetching custom fields for user %d: %s", userID, err)
	}
	return customFields, nil
}

// Delete a user from Brivo
func deleteUser(user models.BrivoUser) error {
	semaphore <- true
	defer func() {
		<-semaphore
	}()
	if err := user.DeleteUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Error deleting user %d: %s", user.ID, err)
	}
	return nil
}

// Uses mutual exclusion for thread-safe update to brivoIDSet map[]
func (set *brivoIDSet) update(userID string) {
	set.access.Lock()
	set.ids[userID] = true
	set.access.Unlock()
}
