/**
 * Brivo Credential Data Model
 *
 * Using CredentialFormat: 'Unknown Format'. Make a request to `v1/api/credentials/formats`
 * to list supported credential formats
 *
 * @link	https://apidocs.brivo.com/#api-Credential-ListCredentialFormats
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	async "github.com/christophertino/fiao-sync/utils"
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

// Create new Brivo access credential
func (cred *Credential) createCredential(brivoAPIKey string, brivoAccessToken string) (float64, error) {
	// Build request body JSON
	bytesMessage, err := json.Marshal(cred)
	if err != nil {
		log.Println("credential.createCredential: Error building POST body json", err)
		return 0, err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/credentials", bytes.NewBuffer(bytesMessage))
	if err != nil {
		log.Println("credential.createCredential: Error creating HTTP request", err)
		return 0, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err := async.DoRequest(req, &r); err != nil {
		return 0, err
	}

	// Return the new credential ID
	return r["id"].(float64), nil
}