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
	success int
	failed  map[string]string
}

var (
	auth         models.Auth
	mb           models.MindBody
	brivo        models.Brivo
	wg           sync.WaitGroup
	isRefreshing bool
	semaphore    chan bool
	errChan      chan *models.BrivoUser
	o            outputLog
)

// GetAllUsers will fetch all existing users from MINDBODY and Brivo
func GetAllUsers(config *models.Config) {
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
	createUsers(config)
}

// Iterate over all MINDBODY users, convert them to Brivo users
// and POST to them Brivo API along with credential and group assignments
func createUsers(config *models.Config) {
	o.failed = make(map[string]string)

	// Handle rate limiting. Brivo rate limit is 20 calls/second
	const rateLimit = 20
	semaphore = make(chan bool, rateLimit)

	// Iterate over all MINDBODY users
	for i := range mb.Clients {
		var user models.BrivoUser
		mbUser := mb.Clients[i]

		// Validate that the ClientID is a valid hex ID
		if !models.IsValidID(mbUser.ID) {
			fmt.Printf("User %s is not a valid hex ID\n", mbUser.ID)
			o.failed[mbUser.ID] = "Invalid Hex ID"
			continue
		}

		// Convert MINDBODY user to Brivo user
		user.BuildUser(mbUser)

		// Check current refresh status
		if !isRefreshing {
			// Process the event normally
			processUser(&user, config)
		} else {
			// A refresh is currently taking place. Push the event into the error channel
			errChan <- &user
		}
	}
}

// Make Brivo API calls
func processUser(user *models.BrivoUser, config *models.Config) {
	wg.Add(1)
	semaphore <- true
	go func(u models.BrivoUser) {
		defer func() {
			<-semaphore
			wg.Done()
		}()

		// Create a new user
		err := u.CreateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch err := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if err.Error() == "401" {
				errChan <- user
				doRefresh(config)
				return
			}
		default:
			fmt.Printf("Error creating user %s with error: %s\n", u.ExternalID, err)
			o.failed[u.ExternalID] = "Create User"
			return
		}

		// Create new Brivo credential for this user
		cred := models.GenerateCredential(u.ExternalID)
		credID, err := cred.CreateCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch err := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if err.Error() == "401" {
				errChan <- user
				doRefresh(config)
				return
			}
		default:
			fmt.Printf("Error creating credential for user %s with error: %s\n", u.ExternalID, err)
			o.failed[u.ExternalID] = "Create Credential"
			return
		}

		// Assign credential to user
		err = u.AssignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch err := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if err.Error() == "401" {
				errChan <- user
				doRefresh(config)
				return
			}
		default:
			fmt.Printf("Error assigning credential to user %s with error: %s\n", u.ExternalID, err)
			o.failed[u.ExternalID] = "Assign Credential"
			return
		}

		// Assign user to group
		err = u.AssignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch err := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if err.Error() == "401" {
				errChan <- user
				doRefresh(config)
				return
			}
		default:
			fmt.Printf("Error assigning user %s to group with error: %s\n", u.ExternalID, err)
			o.failed[u.ExternalID] = "Assign Group"
			return
		}

		o.success++
		fmt.Printf("Successfully created Brivo user %s\n", u.ExternalID)
		time.Sleep(time.Second * 1)
	}(*user)

	wg.Wait()

	o.printLog()
}

// Check current refreshing status and process new refresh token
func doRefresh(config *models.Config) {
	if isRefreshing {
		return
	}
	isRefreshing = true
	if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
		fmt.Println("Error refreshing Brivo AUTH token:\n", err)
		return
	}
	fmt.Println("Refreshed Brivo AUTH token")

loop:
	// Listen for new events in the error channel
	for {
		select {
		case user := <-errChan:
			processUser(user, config)
		default:
			break loop
		}
	}

	isRefreshing = false
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
