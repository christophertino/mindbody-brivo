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
	"log"
	"net/http"

	async "github.com/christophertino/fiao-sync/utils"
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

// Retrieve a MindBody Access Token
func (token *mbToken) GetMindBodyToken(config Config) error {
	// Build request body JSON
	body := map[string]string{
		"Username": config.MindbodyUsername,
		"Password": config.MindbodyPassword,
	}
	bytesMessage, err := json.Marshal(body)
	if err != nil {
		log.Println("auth.GetMindBodyToken: Error building POST body json", err)
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mindbodyonline.com/public/v6/usertoken/issue", bytes.NewBuffer(bytesMessage))
	if err != nil {
		log.Println("auth.GetMindBodyToken: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", config.MindbodySite)
	req.Header.Add("Api-Key", config.MindbodyAPIKey)

	if err = async.DoRequest(req, token); err != nil {
		return err
	}

	return nil
}

// Retrieve a Brivo Access Token using password grant type
func (token *brivoToken) GetBrivoToken(config *Config) error {
	// Create HTTP request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://auth.brivo.com/oauth/token?grant_type=password&username=%s&password=%s", config.BrivoUsername, config.BrivoPassword), nil)
	if err != nil {
		log.Println("auth.GetBrivoToken: Error creating HTTP request", err)
		return err
	}
	config.BuildClientCredentials()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+config.BrivoClientCredentials)
	req.Header.Add("api-key", config.BrivoAPIKey)

	if err = async.DoRequest(req, token); err != nil {
		return err
	}

	return nil
}
