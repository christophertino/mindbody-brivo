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

	"github.com/beefsack/go-rate"
	utils "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
)

type brivoIDSet struct {
	access sync.Mutex
	ids    map[string]bool
}

var (
	auth         models.Auth
	brivo        models.Brivo
	creds        models.CredentialList
	config       *models.Config
	rateLimit    *rate.RateLimiter
	mu           sync.Mutex
	wg           sync.WaitGroup
	brivoIDs     brivoIDSet
	isRefreshing bool
	errUser      chan *models.BrivoUser
	errCred      chan *models.Credential
)

// Nuke is meant for cleaning up your Brivo developer environment. It will
// remove all existing users and credentials so that you can start fresh.
func Nuke(cfg *models.Config, scope rune) {
	config = cfg
	isRefreshing = false
	errUser = make(chan *models.BrivoUser, config.BrivoRateLimit)
	errCred = make(chan *models.Credential, config.BrivoRateLimit)

	// Generate Brivo access token
	if err := auth.BrivoToken.GetBrivoToken(config); err != nil {
		log.Fatalf("Error generating Brivo access token: %s", err)
	}

	// Get Brivo users
	if scope == '1' {
		// Fetch from Member Group only
		if err := brivo.ListUsersWithinGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
			log.Fatalln("Error fetching Brivo users", err)
		}
	} else if scope == '2' {
		// Fetch all users
		if err := brivo.ListUsers(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
			log.Fatalln("Error fetching Brivo users", err)
		}
	} else {
		log.Fatalln("Error fetching Brivo users")
	}

	// Get all Brivo credentials
	if err := creds.GetCredentials(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		log.Fatalf("Error fetching Brivo credentials: %s", err)
	}

	// Allow Brivo rate limit to reset
	time.Sleep(time.Second * 1)

	// Handle rate limiting
	rateLimit = rate.New(config.BrivoRateLimit, time.Second)

	// Keep track of which users we deleted
	brivoIDs.ids = make(map[string]bool)

	fmt.Println("Deleteing all Brivo users...")

	// Loop over all users and delete
	for _, user := range brivo.Data {
		if isRefreshing {
			errUser <- &user
			continue
		}
		processUser(user)
	}

	wg.Wait()

	fmt.Println("Deleteing all Brivo credentials...")

	// Loop over all credentials and delete
	for _, cred := range creds.Data {
		if isRefreshing {
			errCred <- &cred
			continue
		}
		processCredential(cred)
	}

	// Wait for all routines to finish and close
	wg.Wait()

	fmt.Println("Nuke completed. Check error logs for output.")
}

// Fetch the user's Barcode ID and delete the user from Brivo
func processUser(user models.BrivoUser) {
	wg.Add(1)
	rateLimit.Wait()
	go func(u models.BrivoUser) {
		defer wg.Done()
		// Get custom fields for user
		customFields, err := getCustomFields(&u)
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

		// Delete the user
		if err := deleteUser(&u); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Deleted user %d\n", u.ID)
	}(user)
}

// Remove the credential from Brivo concurrently
func processCredential(cred models.Credential) {
	wg.Add(1)
	rateLimit.Wait()
	go func(c models.Credential) {
		defer wg.Done()

		// Make sure this credential belongs to a user we are deleteing (in the Member group only)
		if brivoIDs.ids[c.ReferenceID] == true {
			// Delete the credential
			if err := deleteCredential(&c); err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Printf("Deleted credential %d\n", c.ID)
	}(cred)
}

// Get user's custom fields
func getCustomFields(user *models.BrivoUser) (models.CustomFields, error) {
	rateLimit.Wait()
	var customFields models.CustomFields
	err := customFields.GetCustomFieldsForUser(user.ID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return customFields, nil
	case *utils.JSONError:
		if e.Code == 401 {
			errUser <- user
			doRefresh()
			return customFields, fmt.Errorf("Access token expired")
		}
	}
	return customFields, fmt.Errorf("Error fetching custom fields for user %d: %s", user.ID, err)
}

// Delete a user from Brivo
func deleteUser(user *models.BrivoUser) error {
	rateLimit.Wait()
	err := user.DeleteUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return nil
	case *utils.JSONError:
		if e.Code == 401 {
			errUser <- user
			doRefresh()
			return fmt.Errorf("Access token expired")
		}
	}
	return fmt.Errorf("Error deleting user %d: %s", user.ID, err)
}

// Delete a credential from Brivo
func deleteCredential(cred *models.Credential) error {
	rateLimit.Wait()
	err := cred.DeleteCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return nil
	case *utils.JSONError:
		if e.Code == 401 {
			errCred <- cred
			doRefresh()
			return fmt.Errorf("Access token expired")
		}
	}
	return fmt.Errorf("Error deleting credential %d: %s", cred.ID, err)
}

// Call Brivo and fetch a refreshed token
func refreshToken() error {
	rateLimit.Wait()
	if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
		return fmt.Errorf("Error refreshing Brivo token: %s", err)
	}
	return nil
}

// Check current refreshing status and process new refresh token
func doRefresh() {
	if isRefreshing {
		return
	}

	// Lock the refresh sequence as there may be multiple routines attempting to refresh at once
	mu.Lock()
	isRefreshing = true

	// Check that token hasn't already been refreshed
	if time.Now().UTC().After(auth.BrivoToken.ExpireTime) {
		if err := refreshToken(); err != nil {
			// Potential infinite loop if Token API continuously fails
			log.Fatalf("Failed refreshing Brivo AUTH token with err %s\n", err)
		}
		fmt.Println("Refreshed Brivo AUTH token")
	}

	isRefreshing = false
	mu.Unlock()

	// Listen for new events in the error channels
loop:
	for {
		select {
		case user := <-errUser:
			processUser(*user)
		case cred := <-errCred:
			processCredential(*cred)
		default:
			break loop
		}
	}
}

// Uses mutual exclusion for thread-safe update to brivoIDSet map[]
func (set *brivoIDSet) update(userID string) {
	set.access.Lock()
	set.ids[userID] = true
	set.access.Unlock()
}
