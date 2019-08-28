// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Brivo Credential Data Model
// Using CredentialFormat: 'Unknown Format'. Make a request to `v1/api/credentials/formats`
// to list supported credential formats.
// See https://apidocs.brivo.com/#api-Credential-ListCredentialFormats

package models

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	async "github.com/christophertino/mindbody-brivo/utils"
)

// CredentialList is the data format returned when querying credentials from Brivo
type CredentialList struct {
	Offset   int          `json:"offset"`
	PageSize int          `json:"pageSize"`
	Data     []Credential `json:"data"`
	Count    int          `json:"count"`
}

// Credential stores the Brivo access credential
type Credential struct {
	ID                int              `json:"id,omitempty"`
	CredentialFormat  CredentialFormat `json:"credentialFormat"`
	ReferenceID       string           `json:"referenceId"`
	EncodedCredential string           `json:"encodedCredential"`
}

// CredentialFormat stores the Brivo credential format
type CredentialFormat struct {
	ID int `json:"id"`
}

// Create new Brivo access credential. If the credential already exists, return the ID
func (cred *Credential) createCredential(brivoAPIKey string, brivoAccessToken string) (int, error) {
	// Build request body JSON
	bytesMessage, err := json.Marshal(cred)
	if err != nil {
		return 0, fmt.Errorf("Error building POST body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/credentials", bytes.NewBuffer(bytesMessage))
	if err != nil {
		return 0, fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	err = async.DoRequest(req, &r)
	switch err := err.(type) {
	case nil:
		// Return the new credential ID
		return int(r["id"].(float64)), nil
	case *async.JSONError:
		// If the credential already exists we need to fetch it's ID from Brivo
		if err.Code == 400 && strings.Contains(err.Body["message"].(string), "Duplicate Credential Found") {
			fmt.Printf("Credential ID %s already exists.\n", cred.ReferenceID)
			return getCredentialByID(cred.ReferenceID, brivoAPIKey, brivoAccessToken)
		}
		// Server error
		return 0, err
	default:
		// General error
		return 0, err
	}
}

// Generate a credential that uses MINDBODY ExternalID
// in an exceptable format for Brivo
func generateCredential(externalID string) *Credential {
	cred := Credential{
		CredentialFormat: CredentialFormat{
			ID: 110, // Unknown Format
		},
		ReferenceID:       externalID, // barcode ID
		EncodedCredential: hex.EncodeToString([]byte(externalID)),
	}
	return &cred
}

// Get a Brivo credential by reference_id (external id) and return the credentail ID
func getCredentialByID(externalID string, brivoAPIKey string, brivoAccessToken string) (int, error) {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/credentials?filter=reference_id__eq:%s", externalID), nil)
	if err != nil {
		return 0, fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var creds CredentialList
	if err = async.DoRequest(req, &creds); err != nil {
		return 0, err
	}

	// The count should always be 1 or 0
	if creds.Count > 0 {
		fmt.Printf("Successfully fetched Credential ID %d for Reference ID %s\n", creds.Data[0].ID, externalID)
		return creds.Data[0].ID, nil
	}

	return 0, fmt.Errorf("Credential with ReferenceID %s not found", externalID)
}

// GetCredentials fetches all existing credentials from Brivo
func (creds *CredentialList) GetCredentials(brivoAPIKey string, brivoAccessToken string) error {
	var (
		count    = 0
		pageSize = 100
		results  []Credential
	)

	for {
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/credentials?offset=%d&pageSize:%d", count, pageSize), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
		req.Header.Add("api-key", brivoAPIKey)

		if err = async.DoRequest(req, &creds); err != nil {
			return err
		}

		results = append(results, creds.Data...)
		count += creds.PageSize
		if count >= creds.Count {
			break
		}
	}

	creds.Data = results

	return nil
}

// DeleteCredential will delete a Brivo user by ID
func (cred *Credential) DeleteCredential(brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.brivo.com/v1/api/credentials/%d", cred.ID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
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
