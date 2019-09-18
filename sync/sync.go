// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Migrate all MINDBODY clients to Brivo as new users

package sync

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	utils "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
)

// Creates a log of users created/failed during sync
type outputLog struct {
	access  sync.Mutex
	success int
	failed  map[string]string
}

var (
	auth         models.Auth
	config       *models.Config
	brivo        models.Brivo
	mb           models.MindBody
	wg           sync.WaitGroup
	isRefreshing bool
	semaphore    chan bool
	errChan      chan *models.BrivoUser
	o            outputLog
)

// GetAllUsers will fetch all existing users from MINDBODY and Brivo
func GetAllUsers(c *models.Config) {
	config = c

	if err := auth.Authenticate(config); err != nil {
		fmt.Println("Error generating AUTH tokens:", err)
		return
	}

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
	createUsers()
}

// Iterate over all MINDBODY users, convert them to Brivo users
// and POST to the Brivo API along with credential and group assignments
func createUsers() {
	// Handle rate limiting
	semaphore = make(chan bool, config.BrivoRateLimit)

	// Create buffer channel to handle errors from processUser. Set buffer to Brivo rate limit
	errChan = make(chan *models.BrivoUser, config.BrivoRateLimit)
	isRefreshing = false

	// Instantiate outputLog failed map
	o.failed = make(map[string]string)

	// Iterate over all MINDBODY users
	for i := range mb.Clients {
		var user models.BrivoUser
		mbUser := mb.Clients[i]

		// Validate that the ClientID is a valid hex ID
		if !models.IsValidID(mbUser.ID) {
			// o.failed[mbUser.ID] = "Invalid Hex ID"
			continue
		}

		// Convert MINDBODY user to Brivo user
		user.BuildUser(mbUser, *config)

		// Check current refresh status
		if !isRefreshing {
			// Process the event normally
			processUser(&user)
		} else {
			// A refresh is currently taking place. Push the event into the error channel
			errChan <- &user
		}
	}

	wg.Wait()

	o.printLog()
	fmt.Println("Sync completed. See sync_output.log")
}

// Make Brivo API calls
func processUser(user *models.BrivoUser) {
	wg.Add(1)
	go func(u models.BrivoUser) {
		defer wg.Done()

		if err := createUser(&u); err != nil {
			fmt.Println(err)
			return
		}

		// Set barcode ID
		barcodeID, err := updateCustomField(&u, config.BrivoBarcodeFieldID)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Set status field
		_, err = updateCustomField(&u, config.BrivoStatusFieldID)
		if err != nil {
			fmt.Println(err)
			return
		}

		credID, err := createCredential(barcodeID, &u)
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := assignCredential(credID, &u); err != nil {
			fmt.Println(err)
			return
		}

		if err := assignGroup(&u); err != nil {
			fmt.Println(err)
			return
		}

		o.success++
		fmt.Printf("Successfully created Brivo user %s\n", u.ExternalID)
	}(*user)

	// Respect Brivo rate limit
	time.Sleep(time.Second * 1)
}

// Create a new Brivo user
func createUser(user *models.BrivoUser) error {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	err := user.CreateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return nil
	case *utils.JSONError:
		if e.Code == 401 {
			errChan <- user
			doRefresh()
			return fmt.Errorf("Access token expired")
		}
	}
	o.failure(user.ExternalID, fmt.Sprintf("Create User: %s", err.Error()))
	return fmt.Errorf("Error creating user %s with error: %s", user.ExternalID, err.Error())
}

// Add custom fields for the user
func updateCustomField(user *models.BrivoUser, customFieldID int) (string, error) {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	customFieldValue, err := models.GetFieldValue(customFieldID, user.CustomFields)
	if err == nil {
		err = user.UpdateCustomField(customFieldID, customFieldValue, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch e := err.(type) {
		case nil:
			return customFieldValue, nil
		case *utils.JSONError:
			if e.Code == 401 {
				errChan <- user
				doRefresh()
				return "", fmt.Errorf("Access token expired")
			}
		}
	}
	o.failure(user.ExternalID, fmt.Sprintf("Update Custom Field ID %d: %s", customFieldID, err.Error()))
	return "", fmt.Errorf("Error updating custom field ID %d for user %s with error: %s", customFieldID, user.ExternalID, err.Error())
}

// Create new Brivo credential for this user
func createCredential(barcodeID string, user *models.BrivoUser) (int, error) {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	cred := models.GenerateCredential(barcodeID)
	credID, err := cred.CreateCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return credID, nil
	case *utils.JSONError:
		if e.Code == 401 {
			errChan <- user
			doRefresh()
			return 0, fmt.Errorf("Access token expired")
		}
	}
	o.failure(user.ExternalID, fmt.Sprintf("Create Credential: %s", err.Error()))
	return 0, fmt.Errorf("Error creating credential for user %s with error: %s", user.ExternalID, err.Error())
}

// Assign credential to user
func assignCredential(credID int, user *models.BrivoUser) error {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	err := user.AssignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return nil
	case *utils.JSONError:
		if e.Code == 401 {
			errChan <- user
			doRefresh()
			return fmt.Errorf("Access token expired")
		}
	}
	o.failure(user.ExternalID, fmt.Sprintf("Assign Credential: %s", err.Error()))
	return fmt.Errorf("Error assigning credential to user %s with error: %s", user.ExternalID, err.Error())
}

// Assign user to group
func assignGroup(user *models.BrivoUser) error {
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	err := user.AssignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	case nil:
		return nil
	case *utils.JSONError:
		if e.Code == 401 {
			errChan <- user
			doRefresh()
			return fmt.Errorf("Access token expired")
		}
	}
	o.failure(user.ExternalID, fmt.Sprintf("Assign Group: %s", err.Error()))
	return fmt.Errorf("Error assigning user %s to group with error: %s", user.ExternalID, err.Error())
}

// Call Brivo and fetch a refreshed token
func refreshToken() error {
	semaphore <- true
	defer func() {
		<-semaphore
	}()
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

	isRefreshing = true
	if err := refreshToken(); err != nil {
		fmt.Println(err)
		// @TODO: potential unending loop scenario if Token API continuously fails
		return
	}
	fmt.Println("Refreshed Brivo AUTH token")
	isRefreshing = false

	// Listen for new events in the error channel
loop:
	for {
		select {
		case user := <-errChan:
			processUser(user)
		default:
			break loop
		}
	}
}

// Uses mutual exclusion for thread-safe update to failed map[]
func (o *outputLog) failure(userID string, reason string) {
	o.access.Lock()
	o.failed[userID] = reason
	o.access.Unlock()
}

// PrintLog generates an output log file for Sync app
func (o *outputLog) printLog() {
	var b strings.Builder
	b.WriteString("---------- OUTPUT LOG ----------\n")
	fmt.Fprintln(&b, "Users Created Successfully:", o.success)
	fmt.Fprintln(&b, "Users Failed:", len(o.failed))
	for index, value := range o.failed {
		fmt.Fprintf(&b, "External ID: %s Reason: %s\n", index, value)
	}
	// Write to file
	if err := ioutil.WriteFile("sync_output.log", []byte(b.String()), 0644); err != nil {
		log.Fatalln("Error writing output log", err)
	}
}
