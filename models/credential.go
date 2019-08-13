/**
 * Brivo Credential Data Model
 *
 * Using CredentialFormat: 'Unknown Format'. Make a request to `v1/api/credentials/formats`
 * to list supported credential formats
 *
 * @link	https://apidocs.brivo.com/#api-Credential-ListCredentialFormats
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	async "github.com/christophertino/mindbody-brivo/utils"
)

// Credential : Brivo access credential
type Credential struct {
	CredentialFormat  CredentialFormat `json:"credentialFormat"`
	ReferenceID       string           `json:"referenceId"`
	EncodedCredential string           `json:"encodedCredential"`
}

// CredentialFormat : Credential format
type CredentialFormat struct {
	ID int `json:"id"`
}

// Create new Brivo access credential. If the credential already exists, return the ID
func (cred *Credential) createCredential(brivoAPIKey string, brivoAccessToken string) (int, error) {
	// Build request body JSON
	bytesMessage, err := json.Marshal(cred)
	if err != nil {
		fmt.Println("credential.createCredential: Error building POST body json", err)
		return 0, err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/credentials", bytes.NewBuffer(bytesMessage))
	if err != nil {
		fmt.Println("credential.createCredential: Error creating HTTP request", err)
		return 0, err
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
		// If the credential already exists, return the credential ID
		// so that we can still create a new user
		if err.Code == 400 && strings.Contains(err.Body["message"].(string), "Duplicate Credential Found") {
			fmt.Println("credential.createCredential: Credential ID already exists, continue.")
			return cred.CredentialFormat.ID, nil
		}
	}

	return 0, err
}
