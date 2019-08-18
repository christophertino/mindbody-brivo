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

// Brivo client data
type Brivo struct {
	Data     []brivoUser `json:"data"`
	Offset   int         `json:"offset"`
	PageSize int         `json:"pageSize"`
	Count    int         `json:"count"`
}

type brivoUser struct {
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
		fmt.Println("brivo.ListUsers: Error creating HTTP request", err)
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

// CreateUsers : Convert MB users to Brivo users and create on Brivo
func (brivo *Brivo) CreateUsers(mb MindBody, config Config, auth Auth) {
	var (
		wg sync.WaitGroup
		o  outputLog
	)
	o.failed = make(map[string]string)

	// Map MINDBODY fields to Brivo
	for i := range mb.Clients {
		var (
			user      brivoUser
			userEmail email
			userPhone phoneNumber
		)
		user.ExternalID = mb.Clients[i].ID // barcode ID
		user.FirstName = mb.Clients[i].FirstName
		user.MiddleName = mb.Clients[i].MiddleName
		user.LastName = mb.Clients[i].LastName
		user.Suspended = (mb.Clients[i].Active == false || mb.Clients[i].Status != "Active")

		if mb.Clients[i].Email != "" {
			userEmail.Address = mb.Clients[i].Email
			userEmail.EmailType = "home"
			user.Emails = append(user.Emails, userEmail)
		}

		if mb.Clients[i].HomePhone != "" {
			userPhone.Number = mb.Clients[i].HomePhone
			userPhone.NumberType = "home"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if mb.Clients[i].MobilePhone != "" {
			userPhone.Number = mb.Clients[i].MobilePhone
			userPhone.NumberType = "mobile"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if mb.Clients[i].WorkPhone != "" {
			userPhone.Number = mb.Clients[i].WorkPhone
			userPhone.NumberType = "work"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}

		wg.Add(1)
		go func(u brivoUser) {
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
				fmt.Printf("brivo.CreateUsers: Error creating credential for user %s with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Create Credential"
				return
			}

			// Create a new user
			if err := u.createUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("brivo.CreateUsers: Error creating user %s with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Create User"
				return
			}

			// Assign credential to user
			if err := u.assignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("brivo.CreateUsers: Error assigning credential to user %s with error: %s. Skip to next user.\n", user.ExternalID, err)
				o.failed[user.ExternalID] = "Assign Credential"
				return
			}

			// Assign user to group
			if err := u.assignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("brivo.CreateUsers: Error assigning user %s to group with error: %s. Skip to next user.\n", user.ExternalID, err)
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

// Create a new Brivo user
func (user *brivoUser) createUser(brivoAPIKey string, brivoAccessToken string) error {
	// Check to see if user already exists
	if brivoIDSet[user.ExternalID] == true {
		return fmt.Errorf("brivo.createUser: User already exists")
	}

	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		fmt.Println("brivo.createUser: Error building POST body json", err)
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/users", bytes.NewBuffer(bytesMessage))
	if err != nil {
		fmt.Println("brivo.createUser: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err := async.DoRequest(req, &r); err != nil {
		return err
	}

	// Add new user ID to brivoUser
	user.ID = int(r["id"].(float64))

	return nil
}

// Assign credentialID to new user
func (user *brivoUser) assignUserCredential(credID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/credentials/%d", user.ID, credID), nil)
	if err != nil {
		fmt.Println("brivo.assignUserCredential: Error creating HTTP request", err)
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
func (user *brivoUser) assignUserGroup(groupID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/groups/%d/users/%d", groupID, user.ID), nil)
	if err != nil {
		fmt.Println("brivo.assignUserGroup: Error creating HTTP request", err)
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
func (brivo *Brivo) GetUserByID(externalID string, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users/%s/external", externalID), nil)
	if err != nil {
		fmt.Println("brivo.assignUserCredential: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	if err = async.DoRequest(req, &brivo); err != nil {
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
