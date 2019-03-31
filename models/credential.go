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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

type credentialError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

/**
 * Create new Brivo access credential
 * @param	config
 * @param	auth
 * @return	credentialID, error
 */
func (cred *Credential) createCredential(config *Config, auth *Auth) (string, error) {
	// var client http.Client
	var PTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	client := http.Client{Transport: PTransport}

	bytesMessage, err := json.Marshal(cred)
	if err != nil {
		log.Println("credential.createCredential: Error building POST body json", err)
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/credentials", bytes.NewBuffer(bytesMessage))
	if err != nil {
		log.Println("credential.createCredential: Error creating HTTP request", err)
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+auth.BrivoToken.AccessToken)
	req.Header.Add("api-key", config.BrivoAPIKey)

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("credential.createCredential: Error reading response", err)
		return "", err
	}

	// Check for error response
	if res.StatusCode >= 400 {
		var ce credentialError
		_ = json.Unmarshal(data, &ce)
		return "", fmt.Errorf("credential.createCredential: Error creating credential \n %+v", ce)
	}

	// Build response into Model
	var output map[string]interface{}
	err = json.Unmarshal(data, &output)
	if err != nil {
		log.Println("credential.createCredential: Error unmarshalling json", err)
		return "", err
	}

	fmt.Printf("%+v", output)

	// Return the new credential ID
	return output["id"].(string), nil
}
