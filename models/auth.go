// Handles AUTH tokens for MINDBODY and Brivo
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	utils "github.com/christophertino/mindbody-brivo"
)

// Auth stores authentication tokens for Brivo and MINDBODY
type Auth struct {
	BrivoToken    BrivoToken
	MindBodyToken mbToken
}

// BrivoToken stores Brivo API Tokens. Valid until `ExpiresIn` and then
// must be refreshed with `RefreshToken`
type BrivoToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	JTI          string `json:"jti"`
	ExpireTime   time.Time
}

// MINDBODY access token. Valid for 7 days
type mbToken struct {
	TokenType   string `json:"TokenType"`
	AccessToken string `json:"AccessToken"`
}

// Authenticate fetches access tokens for MINDBODY and Brivo
func (auth *Auth) Authenticate(config *Config) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	// Fetch MINDBODY token
	go func() {
		if err := auth.MindBodyToken.getMindBodyToken(*config); err != nil {
			errCh <- err
		} else {
			doneCh <- true
		}
	}()

	//Fetch Brivo token
	go func() {
		if err := auth.BrivoToken.GetBrivoToken(config); err != nil { // Pass `config` by reference
			errCh <- err
		} else {
			doneCh <- true
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			fmt.Println("Token fetch success!")
		}
	}

	return nil
}

// Retrieve a MINDBODY Access Token
func (token *mbToken) getMindBodyToken(config Config) error {
	// Build request body JSON
	body := map[string]string{
		"Username": config.MindbodyUsername,
		"Password": config.MindbodyPassword,
	}
	bytesMessage, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("Error building request body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mindbodyonline.com/public/v6/usertoken/issue", bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", config.MindbodySite)
	req.Header.Add("Api-Key", config.MindbodyAPIKey)

	if err = utils.DoRequest(req, token); err != nil {
		return err
	}

	return nil
}

// GetBrivoToken retrieves a Brivo Access Token using password grant type.
// It accepts `config` as a reference for updating via BuildClientCredentials().
func (token *BrivoToken) GetBrivoToken(config *Config) error {
	// Create HTTP request
	req, err := http.NewRequest("POST", "https://auth.brivo.com/oauth/token", nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	// Encode credentials
	query := req.URL.Query()
	query.Add("grant_type", "password")
	query.Add("username", config.BrivoUsername)
	query.Add("password", config.BrivoPassword)
	req.URL.RawQuery = query.Encode()

	config.buildClientCredentials()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+config.BrivoClientCredentials)
	req.Header.Add("api-key", config.BrivoAPIKey)

	if err = utils.DoRequest(req, token); err != nil {
		return err
	}

	// Set AccessToken expiration time
	token.ExpireTime = time.Now().UTC().Add(time.Second * time.Duration(token.ExpiresIn))

	return nil
}

// RefreshBrivoToken fetches a Brivo refresh token after the original access token expires
func (token *BrivoToken) RefreshBrivoToken(config Config) error {
	// Create HTTP request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://auth.brivo.com/oauth/token?grant_type=refresh_token&refresh_token=%s", token.RefreshToken), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+config.BrivoClientCredentials)
	req.Header.Add("api-key", config.BrivoAPIKey)

	if err = utils.DoRequest(req, token); err != nil {
		return err
	}

	// Update AccessToken expiration time
	token.ExpireTime = time.Now().UTC().Add(time.Second * time.Duration(token.ExpiresIn))

	return nil
}
