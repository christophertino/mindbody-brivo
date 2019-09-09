// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// MINDBODY Data Model

package models

import (
	"fmt"
	"net/http"
	"regexp"

	utils "github.com/christophertino/mindbody-brivo"
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

// MindBodyUser stores MINDBODY user data
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

// GetClients builds the MINDBODY data model with client data
func (mb *MindBody) GetClients(config Config, mbAccessToken string) error {
	var (
		count   = 0
		limit   = 200 // Max 200
		results []MindBodyUser
	)

	utils.Logger("Fetching all MINDBODY clients...")

	for {
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.mindbodyonline.com/public/v6/client/clients?limit=%d&offset=%d", limit, count), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SiteId", config.MindbodySite)
		req.Header.Add("Api-Key", config.MindbodyAPIKey)
		req.Header.Add("Authorization", mbAccessToken)

		if err = utils.DoRequest(req, mb); err != nil {
			return err
		}

		utils.Logger(fmt.Sprintf("Got MINDBODY clients %d of %d", count, mb.PaginationResponse.TotalResults))

		results = append(results, mb.Clients...)
		count += mb.PaginationResponse.PageSize

		if count >= mb.PaginationResponse.TotalResults {
			break
		}
	}

	mb.Clients = results

	utils.Logger(fmt.Sprintf("Completed fetching %d MINDBODY clients.", mb.PaginationResponse.TotalResults))

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

// IsValidID checks to make sure MindBodyUser.ID and EventUserData.ClientID
// are valid 8 digit hex values. If the ID value does not match, that means the user has
// not been assigned a MINDBODY security bracelet and should not be added to Brivo.
func IsValidID(clientID string) bool {
	match, err := regexp.MatchString("^[0-9a-fA-F]{8}$", clientID)
	if err != nil {
		return false
	}
	return match
}
