// Brivo Credential Data Model
//
// Make a request to `v1/api/credentials/formats` to list supported credential formats.
// See https://apidocs.brivo.com/#api-Credential-ListCredentialFormats
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	utils "github.com/christophertino/mindbody-brivo"
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
	ReferenceID       string           `json:"referenceId"` // Barcode ID (MindBodyUser.ID | BrivoUser.CustomField BarcodeID)
	EncodedCredential string           `json:"encodedCredential"`
	FieldValues       []FieldValue     `json:"fieldValues"`
}

// CredentialFormat stores the Brivo credential format
type CredentialFormat struct {
	ID int `json:"id"`
}

// FieldValue contains relevant Credential fields for `card_number` and `facility_code`
type FieldValue struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// CreateCredential will create new Brivo access credential. If the credential already exists, return the ID
func (cred *Credential) CreateCredential(brivoAPIKey string, brivoAccessToken string) (int, error) {
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
	err = utils.DoRequest(req, &r)
	switch e := err.(type) {
	case nil:
		// Return the new credential ID
		return int(r["id"].(float64)), nil
	case *utils.JSONError:
		// If the credential already exists we need to fetch it's ID from Brivo
		if e.Code == 400 && strings.Contains(e.Body["message"].(string), "Duplicate Credential Found") {
			fmt.Printf("Credential ID %s already exists.\n", cred.ReferenceID)
			cred, err := getCredentialByID(cred.ReferenceID, brivoAPIKey, brivoAccessToken)
			if err != nil {
				return 0, err
			}
			return cred.ID, nil
		}
		// Server error
		return 0, err
	default:
		// General error
		return 0, err
	}
}

// GenerateStandardCredential creates a Standard 26 Bit credential that uses the MINDBODY
// barcode ID and Brivo facility code as Field Values
func GenerateStandardCredential(barcodeID string, facilityCode int) *Credential {
	cardNumber := strings.Replace(barcodeID, fmt.Sprintf("%d-", facilityCode), "", 1) // remove facilityCode from barcodeID
	cred := Credential{
		CredentialFormat: CredentialFormat{
			ID: 100, // Standard 26 Bit Format
		},
		ReferenceID: cardNumber,
		FieldValues: []FieldValue{
			FieldValue{
				ID:    1, // card number
				Value: cardNumber,
			},
			FieldValue{
				ID:    2, // facility_code
				Value: string(facilityCode),
			},
		},
	}
	return &cred
}

// GenerateUnknownCredential creates an encoded credential using format 110 Unknown
// @deprecated
func GenerateUnknownCredential(barcodeID string) *Credential {
	cred := Credential{
		CredentialFormat: CredentialFormat{
			ID: 110, // Unknown Format
		},
		ReferenceID:       barcodeID, // barcode ID
		EncodedCredential: hex.EncodeToString([]byte(barcodeID)),
	}
	return &cred
}

// Get a Brivo credential by reference_id (Credential.ReferenceID) and return the Credential
func getCredentialByID(barcodeID string, brivoAPIKey string, brivoAccessToken string) (Credential, error) {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/credentials?filter=reference_id__eq:%s", barcodeID), nil)
	if err != nil {
		return Credential{}, fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var creds CredentialList
	if err = utils.DoRequest(req, &creds); err != nil {
		return Credential{}, err
	}

	// The count should always be 1 or 0
	if creds.Count > 0 {
		fmt.Printf("Successfully fetched Credential ID %d for Reference ID %s\n", creds.Data[0].ID, barcodeID)
		return creds.Data[0], nil
	}

	return Credential{}, fmt.Errorf("Credential with ReferenceID %s not found", barcodeID)
}

// GetCredentials fetches all existing credentials from Brivo
func (creds *CredentialList) GetCredentials(brivoAPIKey string, brivoAccessToken string) error {
	var (
		count    = 0
		pageSize = 100 // Max 100
		results  []Credential
	)

	utils.Logger("Fetching all Brivo credentials...")

	for {
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/credentials?offset=%d&pageSize=%d", count, pageSize), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
		req.Header.Add("api-key", brivoAPIKey)

		if err = utils.DoRequest(req, creds); err != nil {
			return err
		}

		utils.Logger(fmt.Sprintf("Got credentials %d of %d", count, creds.Count))

		results = append(results, creds.Data...)
		count += creds.PageSize
		if count >= creds.Count {
			break
		}
	}

	creds.Data = results

	utils.Logger(fmt.Sprintf("Completed fetching %d Brivo credentials.", creds.Count))

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
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}
