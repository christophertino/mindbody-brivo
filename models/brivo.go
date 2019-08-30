// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Brivo Data Model

package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	utils "github.com/christophertino/mindbody-brivo"
)

// Brivo stores Brivo API response data
type Brivo struct {
	Data     []BrivoUser `json:"data"`
	Offset   int         `json:"offset"`
	PageSize int         `json:"pageSize"`
	Count    int         `json:"count"`
}

// BrivoUser stores Brivo user data
type BrivoUser struct {
	ID           int           `json:"id,omitempty"`
	ExternalID   string        `json:"externalId"` // Barcode ID from MINDBODY to link accounts
	FirstName    string        `json:"firstName"`
	MiddleName   string        `json:"middleName"`
	LastName     string        `json:"lastName"`
	Suspended    bool          `json:"suspended"`
	CustomFields []customField `json:"customFields"`
	Emails       []email       `json:"emails"`
	PhoneNumbers []phoneNumber `json:"phoneNumbers"`
}

type customField struct {
	FieldName string `json:"fieldName"`
	FieldType string `json:"fieldType"`
}

type email struct {
	Address   string `json:"address"`
	EmailType string `json:"type"`
}

type phoneNumber struct {
	Number     string `json:"number"`
	NumberType string `json:"type"`
}

type outputLog struct {
	success int
	failed  map[string]string
}

var brivoIDSet map[string]bool // keep track of all existing IDs for quick lookup

// ListUsers builds the Brivo data model with user data
func (brivo *Brivo) ListUsers(brivoAPIKey string, brivoAccessToken string) error {
	var (
		count      = 0
		pageSize   = 100 // Max 100
		brivoIDSet = make(map[string]bool)
		results    []BrivoUser
	)
	for {
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users?offset=%d&pageSize:%d", count, pageSize), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
		req.Header.Add("api-key", brivoAPIKey)

		if err = utils.DoRequest(req, brivo); err != nil {
			return err
		}

		// Stash external IDs in a set so we can check them later against MB users
		for _, element := range brivo.Data {
			brivoIDSet[element.ExternalID] = true
		}

		results = append(results, brivo.Data...)
		count += brivo.PageSize
		if count >= brivo.Count {
			break
		}
	}
	brivo.Data = results

	return nil
}

// CreateUsers will iterate over all MINDBODY users, convert them to Brivo users
// and POST to them Brivo API along with credential and group assignments
func (brivo *Brivo) CreateUsers(mb MindBody, config Config, auth Auth) {
	var (
		wg sync.WaitGroup
		o  outputLog
	)
	o.failed = make(map[string]string)

	// Iterate over all MINDBODY users
	for i := range mb.Clients {
		var user BrivoUser
		mbUser := mb.Clients[i]

		// Validate that the ClientID is a valid hex ID
		if !IsValidID(mbUser.ID) {
			fmt.Printf("User %s is not a valid hex ID", mbUser.ID)
			return
		}

		// Convert MINDBODY user to Brivo user
		user.buildUser(mbUser)

		// Make Brivo API calls
		wg.Add(1)
		go func(u BrivoUser) {
			defer wg.Done()
			fmt.Println("Creating user...")

			// Create a new user
			if err := u.createUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Error creating user %s with error: %s\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Create User"
				return
			}

			// Create new Brivo credential for this user
			cred := generateCredential(u.ExternalID)
			credID, err := cred.createCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
			if err != nil {
				fmt.Printf("Error creating credential for user %s with error: %s\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Create Credential"
				return
			}

			// Assign credential to user
			if err := u.assignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Error assigning credential to user %s with error: %s\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Assign Credential"
				return
			}

			// Assign user to group
			if err := u.assignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Error assigning user %s to group with error: %s\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Assign Group"
				return
			}
			o.success++
			fmt.Printf("Successfully created Brivo user %s\n", u.ExternalID)
		}(user)
		wg.Wait()
	}
	o.printLog()
}

// Build a Brivo user from MINDBODY user data
func (user *BrivoUser) buildUser(mbUser MindBodyUser) {
	var (
		userEmail email
		userPhone phoneNumber
	)
	user.ExternalID = mbUser.ID // barcode ID
	user.FirstName = mbUser.FirstName
	user.MiddleName = mbUser.MiddleName
	user.LastName = mbUser.LastName
	user.Suspended = (mbUser.Active == false || mbUser.Status != "Active")
	user.CustomFields = []customField{} // prevents nil comparator issues with cmp.Equal()
	if mbUser.Email != "" {
		userEmail.Address = mbUser.Email
		userEmail.EmailType = "home"
		user.Emails = append(user.Emails, userEmail)
	}
	if mbUser.HomePhone != "" {
		userPhone.Number = mbUser.HomePhone
		userPhone.NumberType = "home"
		user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
	}
	if mbUser.MobilePhone != "" {
		userPhone.Number = mbUser.MobilePhone
		userPhone.NumberType = "mobile"
		user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
	}
	if mbUser.WorkPhone != "" {
		userPhone.Number = mbUser.WorkPhone
		userPhone.NumberType = "work"
		user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
	}
}

// Create a new Brivo user
func (user *BrivoUser) createUser(brivoAPIKey string, brivoAccessToken string) error {
	// Check to see if user already exists
	if brivoIDSet[user.ExternalID] == true {
		return fmt.Errorf("User already exists")
	}

	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("Error building POST body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/users", bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	// Add new user ID to BrivoUser
	user.ID = int(r["id"].(float64))

	return nil
}

// Assign credentialID to new user
func (user *BrivoUser) assignUserCredential(credID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/credentials/%d", user.ID, credID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Assign user to group
func (user *BrivoUser) assignUserGroup(groupID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/groups/%d/users/%d", groupID, user.ID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Retrieves a Brivo user by their ExternalID value
func (user *BrivoUser) getUserByID(externalID string, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users/%s/external", externalID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	if err = utils.DoRequest(req, user); err != nil {
		return err
	}

	return nil
}

// Update an existing Brivo user
func (user *BrivoUser) updateUser(brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("Error building POST body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d", user.ID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Update the suspended status of the user in Brivo
func (user *BrivoUser) toggleSuspendedStatus(suspended bool, brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(map[string]bool{"suspended": suspended})
	if err != nil {
		return fmt.Errorf("Error building POST body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/suspended", user.ID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// DeleteUser will delete a Brivo user by ID
func (user *BrivoUser) DeleteUser(brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d", user.ID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Print output log for Sync app
func (o *outputLog) printLog() {
	fmt.Println("---------- OUTPUT LOG ----------")
	fmt.Println("Users Created Successfully:", o.success)
	fmt.Println("Users Failed:", len(o.failed))
	for index, value := range o.failed {
		fmt.Printf("External ID: %s Reason: %s\n", index, value)
	}
}
