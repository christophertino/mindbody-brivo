// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Brivo Data Model

package models

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	async "github.com/christophertino/mindbody-brivo/utils"
)

// Brivo API response data
type Brivo struct {
	Data     []BrivoUser `json:"data"`
	Offset   int         `json:"offset"`
	PageSize int         `json:"pageSize"`
	Count    int         `json:"count"`
}

// BrivoUser : Brivo user data
type BrivoUser struct {
	ID           int           `json:"id,omitempty"`
	ExternalID   string        `json:"externalId"` // Barcode ID from MINDBODY to link accounts
	FirstName    string        `json:"firstName"`
	MiddleName   string        `json:"middleName"`
	LastName     string        `json:"lastName"`
	Suspended    bool          `json:"suspended"`
	CustomFields []customField `json:"customFields,omitempty"`
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

// ListUsers : Build Brivo data model with user data
func (brivo *Brivo) ListUsers(brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.brivo.com/v1/api/users", nil)
	if err != nil {
		fmt.Println("Brivo.ListUsers: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	if err = async.DoRequest(req, brivo); err != nil {
		return err
	}

	// Stash external IDs in a set so we can check them later against MB users
	brivoIDSet = make(map[string]bool)
	for _, element := range brivo.Data {
		brivoIDSet[element.ExternalID] = true
	}

	return nil
}

// CreateUsers : Iterate over all MINDBODY users, convert to Brivo users
// and POST to Brivo API along with credential and group assignments
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
		user.BuildUser(mbUser)

		// Make Brivo API calls
		wg.Add(1)
		go func(u BrivoUser) {
			defer wg.Done()
			// Create new Brivo credential for this user
			cred := Credential{
				CredentialFormat: CredentialFormat{
					ID: 110, // Unknown Format
				},
				ReferenceID:       u.ExternalID, // barcode ID
				EncodedCredential: hex.EncodeToString([]byte(u.ExternalID)),
			}
			credID, err := cred.createCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
			if err != nil {
				fmt.Printf("Brivo.CreateUsers: Error creating credential for user %s with error: %s. Skip to next user.\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Create Credential"
				return
			}

			// Create a new user
			if err := u.createUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Brivo.CreateUsers: Error creating user %s with error: %s. Skip to next user.\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Create User"
				return
			}

			// Assign credential to user
			if err := u.assignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Brivo.CreateUsers: Error assigning credential to user %s with error: %s. Skip to next user.\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Assign Credential"
				return
			}

			// Assign user to group
			if err := u.assignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Brivo.CreateUsers: Error assigning user %s to group with error: %s. Skip to next user.\n", u.ExternalID, err)
				o.failed[u.ExternalID] = "Assign Group"
				return
			}
			o.success++
			fmt.Printf("Brivo.CreateUsers: Successfully created Brivo user %s\n", u.ExternalID)
		}(user)
		wg.Wait()
	}
	o.printLog()
}

// BuildUser : Build a Brivo user from MINDBODY user data
func (user *BrivoUser) BuildUser(mbUser MindBodyUser) {
	var (
		userEmail email
		userPhone phoneNumber
	)
	user.ExternalID = mbUser.ID // barcode ID
	user.FirstName = mbUser.FirstName
	user.MiddleName = mbUser.MiddleName
	user.LastName = mbUser.LastName
	user.Suspended = (mbUser.Active == false || mbUser.Status != "Active")
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
		return fmt.Errorf("BrivoUser.createUser: User already exists")
	}

	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		fmt.Println("BrivoUser.createUser: Error building POST body json", err)
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/users", bytes.NewBuffer(bytesMessage))
	if err != nil {
		fmt.Println("BrivoUser.createUser: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err := async.DoRequest(req, &r); err != nil {
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
		fmt.Println("BrivoUser.assignUserCredential: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = async.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Assign user to group
func (user *BrivoUser) assignUserGroup(groupID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/groups/%d/users/%d", groupID, user.ID), nil)
	if err != nil {
		fmt.Println("BrivoUser.assignUserGroup: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = async.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// GetUserByID : Retrieves a Brivo user by their ExternalID value
func (user *BrivoUser) GetUserByID(externalID string, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users/%s/external", externalID), nil)
	if err != nil {
		fmt.Println("BrivoUser.GetUserByID: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	if err = async.DoRequest(req, &user); err != nil {
		return err
	}

	return nil
}

// UpdateUser : Update an existing Brivo user
func (user *BrivoUser) UpdateUser(brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		fmt.Println("BrivoUser.UpdateUser: Error building POST body json", err)
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d", user.ID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		fmt.Println("BrivoUser.UpdateUser: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = async.DoRequest(req, &r); err != nil {
		return err
	}

	fmt.Printf("BrivoUser.UpdateUser: Brivo user %d updated successfully.\n", user.ID)

	return nil
}

// ToggleSuspendedStatus : Update the suspended status of the user in Brivo
func (user *BrivoUser) ToggleSuspendedStatus(suspended bool, brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(map[string]bool{"suspended": suspended})
	if err != nil {
		fmt.Println("BrivoUser.DeactivateUser: Error building POST body json", err)
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/suspended", user.ID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		fmt.Println("BrivoUser.DeactivateUser: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = async.DoRequest(req, &r); err != nil {
		return err
	}

	fmt.Printf("BrivoUser.ToggleSuspendedStatus: Brivo user %d suspended status set to %b\n", user.ID, suspended)

	return nil
}

func (o *outputLog) printLog() {
	fmt.Println("---------- OUTPUT LOG ----------")
	fmt.Println("Users Created Successfully:", o.success)
	fmt.Println("Users Failed:", len(o.failed))
	for index, value := range o.failed {
		fmt.Printf("External ID: %s Reason: %s\n", index, value)
	}
}
