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
// and POST to the Brivo API along with credential and group assignments
func createUsers(config *models.Config) {
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

	wg.Wait()

	o.printLog()
	fmt.Println("Sync completed. See sync_output.log")
}

// Make Brivo API calls
func processUser(user *models.BrivoUser, config *models.Config) {
	wg.Add(1)
	// Add four values to our rate limit buffer, since we make 4 Brivo API per user
	for i := 0; i < 4; i++ {
		semaphore <- true
	}
	go func(u models.BrivoUser) {
		defer func() {
			wg.Done()
			// Dequeue the semaphore
			for i := 0; i < 4; i++ {
				<-semaphore
			}
		}()

		// Create a new user
		err := u.CreateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch e := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if e.Code == 401 {
				errChan <- user
				doRefresh(config)
				return
			}
			fmt.Printf("Error creating user %s with error code %d and body: %s\n", u.ExternalID, e.Code, e.Body)
			o.failure(u.ExternalID, fmt.Sprintf("Create User: %s", e.Body))
			return
		default:
			fmt.Printf("Error creating user %s with error: %s\n", u.ExternalID, e.Error())
			o.failure(u.ExternalID, fmt.Sprintf("Create User: %s", e.Error()))
			return
		}

		// Create new Brivo credential for this user
		cred := models.GenerateCredential(u.ExternalID)
		credID, err := cred.CreateCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch e := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if e.Code == 401 {
				errChan <- user
				doRefresh(config)
				return
			}
			fmt.Printf("Error creating credential for user %s with error code %d and body: %s\n", u.ExternalID, e.Code, e.Body)
			o.failure(u.ExternalID, fmt.Sprintf("Create Credential: %s", e.Body))
			return
		default:
			fmt.Printf("Error creating credential for user %s with error: %s\n", u.ExternalID, e.Error())
			o.failure(u.ExternalID, fmt.Sprintf("Create Credential: %s", e.Error()))
			return
		}

		// Assign credential to user
		err = u.AssignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch e := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if e.Code == 401 {
				errChan <- user
				doRefresh(config)
				return
			}
			fmt.Printf("Error assigning credential to user %s with error code %d and body: %s\n", u.ExternalID, e.Code, e.Body)
			o.failure(u.ExternalID, fmt.Sprintf("Assign Credential: %s", e.Body))
			return
		default:
			fmt.Printf("Error assigning credential to user %s with error: %s\n", u.ExternalID, e.Error())
			o.failure(u.ExternalID, fmt.Sprintf("Assign Credential: %s", e.Error()))
			return
		}

		// Assign user to group
		err = u.AssignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
		switch e := err.(type) {
		case nil:
			break
		case *utils.JSONError:
			if e.Code == 401 {
				errChan <- user
				doRefresh(config)
				return
			}
			fmt.Printf("Error assigning user %s to group with error code %d and body: %s\n", u.ExternalID, e.Code, e.Body)
			o.failure(u.ExternalID, fmt.Sprintf("Assign Group: %s", e.Body))
			return
		default:
			fmt.Printf("Error assigning user %s to group with error: %s\n", u.ExternalID, e.Error())
			o.failure(u.ExternalID, fmt.Sprintf("Assign Group: %s", e.Error()))
			return
		}

		o.success++
		fmt.Printf("Successfully created Brivo user %s\n", u.ExternalID)
		time.Sleep(time.Second * 1)
	}(*user)
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
