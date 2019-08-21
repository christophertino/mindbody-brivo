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
		mbClient := mb.Clients[i]
		user.BuildUser(mbClient, 0)

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
				fmt.Printf("Brivo.CreateUsers: Error creating credential for user %s with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Create Credential"
				return
			}

			// Create a new user
			if err := u.createUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Brivo.CreateUsers: Error creating user %s with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Create User"
				return
			}

			// Assign credential to user
			if err := u.assignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Brivo.CreateUsers: Error assigning credential to user %s with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Assign Credential"
				return
			}

			// Assign user to group
			if err := u.assignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Brivo.CreateUsers: Error assigning user %s to group with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Assign Group"
				return
			}
			o.success++
			fmt.Printf("Successfully created Brivo user %s\n", user.ExternalID)
		}(user)
		wg.Wait()
	}
	o.printLog()
}

// BuildUser : Build a Brivo user from MINDBODY user data. The property brivoID
// is only used with Event data if the user already exists on Brivo
func (user *BrivoUser) BuildUser(userType interface{}, brivoID int) {
	var (
		userEmail email
		userPhone phoneNumber
	)
	// handle both Event.userData and Mindbody.mbUser types
	switch u := userType.(type) {
	case mbUser:
		user.ExternalID = u.ID // barcode ID
		user.FirstName = u.FirstName
		user.MiddleName = u.MiddleName
		user.LastName = u.LastName
		user.Suspended = (u.Active == false || u.Status != "Active")
		if u.Email != "" {
			userEmail.Address = u.Email
			userEmail.EmailType = "home"
			user.Emails = append(user.Emails, userEmail)
		}
		if u.HomePhone != "" {
			userPhone.Number = u.HomePhone
			userPhone.NumberType = "home"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if u.MobilePhone != "" {
			userPhone.Number = u.MobilePhone
			userPhone.NumberType = "mobile"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if u.WorkPhone != "" {
			userPhone.Number = u.WorkPhone
			userPhone.NumberType = "work"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
	case userData:
		user.ID = brivoID            // Brivo ID property retrieved from GetUserByID()
		user.ExternalID = u.ClientID // The client’s public ID
		user.FirstName = u.FirstName
		user.LastName = u.LastName
		user.Suspended = u.Status != "Active"
		if u.Email != "" {
			userEmail.Address = u.Email
			userEmail.EmailType = "home"
			user.Emails = append(user.Emails, userEmail)
		}
		if u.HomePhone != "" {
			userPhone.Number = u.HomePhone
			userPhone.NumberType = "home"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if u.MobilePhone != "" {
			userPhone.Number = u.MobilePhone
			userPhone.NumberType = "mobile"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if u.WorkPhone != "" {
			userPhone.Number = u.WorkPhone
			userPhone.NumberType = "work"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
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

	return nil
}

// DeactivateUser : Mark the Brivo user as 'suspended'
func (user *BrivoUser) DeactivateUser(brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(map[string]bool{"suspended": true})
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
