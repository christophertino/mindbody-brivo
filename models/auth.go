// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Handles AUTH tokens for MINDBODY and Brivo

package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	async "github.com/christophertino/mindbody-brivo/utils"
)

// Auth : Authentication tokens
type Auth struct {
	BrivoToken    BrivoToken
	MindBodyToken mbToken
}

// BrivoToken : Brivo API Tokens. Valid until `ExpiresIn` and then
// must be refreshed with `RefreshToken`
type BrivoToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	JTI          string `json:"jti"`
}

// MINDBODY access token. Valid for 7 days
type mbToken struct {
	TokenType   string `json:"TokenType"`
	AccessToken string `json:"AccessToken"`
}

// Authenticate : Fetch access tokens for MINDBODY and Brivo
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
			return fmt.Errorf("Auth.Authenticate: Token fetch failed: %g", err)
		case <-doneCh:
			fmt.Println("Auth.Authenticate: Token fetch success!")
		}
	}

	// fmt.Printf("AUTH Model: %+v\n", auth)
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
		fmt.Println("mbToken.getMindBodyToken: Error building POST body json", err)
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mindbodyonline.com/public/v6/usertoken/issue", bytes.NewBuffer(bytesMessage))
	if err != nil {
		fmt.Println("mbToken.getMindBodyToken: Error creating HTTP request", err)
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

// GetBrivoToken : Retrieve a Brivo Access Token using password grant type.
// It accepts `config` as a reference for updating via BuildClientCredentials().
func (token *BrivoToken) GetBrivoToken(config *Config) error {
	// Create HTTP request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://auth.brivo.com/oauth/token?grant_type=password&username=%s&password=%s", config.BrivoUsername, config.BrivoPassword), nil)
	if err != nil {
		fmt.Println("BrivoToken.GetBrivoToken: Error creating HTTP request", err)
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
