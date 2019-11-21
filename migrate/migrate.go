// Migrate all MINDBODY clients to Brivo as new users
//
// Copyright 2019 Christopher Tino. All rights reserved.

package migrate

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/beefsack/go-rate"
	utils "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
)

// Creates a log of users created/failed during migration
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
	mu           sync.Mutex
	wg           sync.WaitGroup
	isRefreshing bool
	rateLimit    *rate.RateLimiter
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
	rateLimit = rate.New(config.BrivoRateLimit, time.Second)

	// Create buffer channel to handle errors from processUser. Set buffer to Brivo rate limit
	errChan = make(chan *models.BrivoUser, config.BrivoRateLimit)
	isRefreshing = false

	// Instantiate outputLog failed map
	o.failed = make(map[string]string)

	// Iterate over all MINDBODY users
	for i := range mb.Clients {
		var user models.BrivoUser
		mbUser := mb.Clients[i]

		// Validate that the ClientID has the correct facility access
		if !models.IsValidID(config.BrivoFacilityCode, mbUser.ID) {
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
	fmt.Println("Migration completed. See migrate_output.log")
}

// Make Brivo API calls
func processUser(user *models.BrivoUser) {
	wg.Add(1)
	rateLimit.Wait()
	go func(u models.BrivoUser) {
		defer wg.Done()

		// Create a new user
		if err := createUser(&u); err != nil {
			fmt.Println(err)
			return
		}

		// Set the Barcode ID custom field
		barcodeID, err := updateCustomField(&u, config.BrivoBarcodeFieldID)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Set the User Type custom field
		_, err = updateCustomField(&u, config.BrivoUserTypeFieldID)
		if err != nil {
			fmt.Println(err)
		}

		// Create a new credential
		credID, err := createCredential(barcodeID, &u)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Assign the credential to the new user
		if err := assignCredential(credID, &u); err != nil {
			fmt.Println(err)
		}

		// Assign the user to the Member's group
		if err := assignGroup(&u); err != nil {
			fmt.Println(err)
		}

		o.success++
		fmt.Printf("Successfully created Brivo user %s\n", u.ExternalID)
	}(*user)
}

// Create a new Brivo user
func createUser(user *models.BrivoUser) error {
	rateLimit.Wait()
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
	rateLimit.Wait()
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
	rateLimit.Wait()
	cred := models.GenerateCredential(barcodeID)
	rateLimit.Wait() // Add another count to the rate limit in case the credential exists and we need to make another call to fetch the ID
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
	rateLimit.Wait()
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
	rateLimit.Wait()
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

// PrintLog generates an output log file for Migration app
func (o *outputLog) printLog() {
	var b strings.Builder
	b.WriteString("---------- OUTPUT LOG ----------\n")
	fmt.Fprintln(&b, "Users Created Successfully:", o.success)
	fmt.Fprintln(&b, "Users Failed:", len(o.failed))
	for index, value := range o.failed {
		fmt.Fprintf(&b, "External ID: %s Reason: %s\n", index, value)
	}
	// Write to file
	if err := ioutil.WriteFile("migrate_output.log", []byte(b.String()), 0644); err != nil {
		log.Fatalln("Error writing output log", err)
	}
}
