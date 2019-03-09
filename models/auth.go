/**
 * Auth Data Model
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

// Auth : Authentication tokens
type Auth struct {
	BrivoToken    brivoToken
	MindBodyToken mbToken
}

type brivoToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	JTI          string `json:"jti"`
}

type mbToken struct {
	TokenType   string `json:"TokenType"`
	AccessToken string `json:"AccessToken"`
}

type brivoError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type mbError struct {
	Error struct {
		Message string `json:"Message"`
		Code    string `json:"Code"`
	} `json:"Error"`
}

/**
 * Retrieve a MindBody Access Token
 */
func (token *mbToken) GetMindBodyToken(config *Config, ch chan string) {
	var client http.Client

	// Build request body JSON
	body := map[string]string{
		"Username": config.MindbodyUsername,
		"Password": config.MindbodyPassword,
	}
	bytesMessage, err := json.Marshal(body)
	if err != nil {
		log.Fatalln("Error building POST body json", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mindbodyonline.com/public/v6/usertoken/issue", bytes.NewBuffer(bytesMessage))
	if err != nil {
		log.Fatalln("Error creating HTTP request", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", config.MindbodySite)
	req.Header.Add("Api-Key", config.MindbodyAPIKey)

	// Make request
	res, err := client.Do(req)
	if err != nil || res.StatusCode >= 400 {
		log.Fatalln("Error fetching MindBody user token", err, res.StatusCode)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response into Model
	err = json.Unmarshal(data, &token)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}

	ch <- "Successfully fetched MindBody Token"
}

/**
 * Retrieve a Brivo Access Token using password grant type
 */
func (token *brivoToken) GetBrivoToken(config *Config, ch chan string) {
	var client http.Client

	// Create HTTP request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://auth.brivo.com/oauth/token?grant_type=password&username=%s&password=%s", config.BrivoUsername, config.BrivoPassword), nil)
	if err != nil {
		log.Fatalln("Error creating HTTP request", err)
	}
	config.BuildClientCredentials()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+config.BrivoClientCredentials)
	req.Header.Add("api-key", config.BrivoAPIKey)

	// Make request
	res, err := client.Do(req)
	if err != nil || res.StatusCode >= 400 {
		log.Fatalln("Error fetching Brivo access token", err, res.StatusCode)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response into Model
	err = json.Unmarshal(data, &token)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}

	ch <- "Successfully fetched Brivo Token"
}
