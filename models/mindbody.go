// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// MINDBODY Data Model

package models

import (
	"fmt"
	"net/http"

	async "github.com/christophertino/mindbody-brivo/utils"
)

// MindBody Client Data
type MindBody struct {
	PaginationResponse struct {
		RequestedLimit  int `json:"RequestedLimit"`
		RequestedOffset int `json:"RequestedOffset"`
		PageSize        int `json:"PageSize"`
		TotalResults    int `json:"TotalResults"`
	} `json:"PaginationResponse"`
	Clients []MindBodyUser `json:"Clients"`
}

// MindBodyUser : MINDBODY user data
type MindBodyUser struct {
	ID          string `json:"Id"`       // Client’s barcode ID used for client-related API calls
	UniqueID    int    `json:"UniqueId"` // Client’s unique system-generated ID
	FirstName   string `json:"FirstName"`
	MiddleName  string `json:"MiddleName"`
	LastName    string `json:"LastName"`
	Email       string `json:"Email"`
	MobilePhone string `json:"MobilePhone"`
	HomePhone   string `json:"HomePhone"`
	WorkPhone   string `json:"WorkPhone"`
	Active      bool   `json:"Active"`
	Status      string `json:"Status"` // Declined,Non-Member,Active,Expired,Suspended,Terminated
}

// GetClients : Build MINDBODY data model with Client data
func (mb *MindBody) GetClients(config Config, mbAccessToken string) error {
	var (
		limit       = 5 // max 200
		count       = 0
		resultsLeft = 1 // so that we loop at least once
	)
	for {
		if resultsLeft <= 0 {
			break
		}
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.mindbodyonline.com/public/v6/client/clients?limit=%d&offset=%d", limit, count), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SiteId", config.MindbodySite)
		req.Header.Add("Api-Key", config.MindbodyAPIKey)
		req.Header.Add("Authorization", mbAccessToken)

		if err = async.DoRequest(req, mb); err != nil {
			return err
		}

		// For testing purposes just get a small set of users
		if config.Debug == true {
			break
		}

		count = count + mb.PaginationResponse.PageSize
		resultsLeft = mb.PaginationResponse.TotalResults - count
	}

	return nil
}

// Build MINDBODY user from webhook EventUserData
func (mbUser *MindBodyUser) buildUser(eventData EventUserData) {
	mbUser.ID = eventData.ClientID
	mbUser.UniqueID = eventData.ClientUniqueID
	mbUser.FirstName = eventData.FirstName
	mbUser.MiddleName = eventData.MiddleName
	mbUser.LastName = eventData.LastName
	mbUser.Email = eventData.Email
	mbUser.MobilePhone = eventData.MobilePhone
	mbUser.HomePhone = eventData.HomePhone
	mbUser.WorkPhone = eventData.WorkPhone
	mbUser.Active = (eventData.Status == "Active")
	mbUser.Status = eventData.Status
}
