// MINDBODY Data Model

package models

import (
	"bytes"
	"encoding/json"
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
	ID          string `json:"Id"`       // Client’s public barcode ID used for client-related API calls (this is changeable)
	UniqueID    int    `json:"UniqueId"` // Client’s unique system-generated ID (does not change)
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

// Client arrival information
type clientArrival struct {
	ClientID   string `json:"ClientId"`
	LocationID int    `json:"LocationId"`
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

// AddArrival logs a client arrival to a location in MINDBODY. This is used
// by Brivo event subscriptions when a user enters the facility through an access point
func AddArrival(barcodeID string, config *Config, mbAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(clientArrival{
		ClientID:   barcodeID,
		LocationID: config.MindbodyLocationID,
	})
	if err != nil {
		return fmt.Errorf("Error building request body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mindbodyonline.com/public/v6/client/addarrival", bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", config.MindbodySite)
	req.Header.Add("Api-Key", config.MindbodyAPIKey)
	req.Header.Add("Authorization", mbAccessToken)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	utils.Logger(fmt.Sprintf("Added arrival for user %s", barcodeID))

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
// follow the correct ID format of XX-XXXXX, where the first two digits are the
// Brivo facility code. If the ID value does not validate, that means the user has
// not been assigned a MINDBODY security bracelet and should not be added to Brivo.
func IsValidID(facilityCode int, barcodeID string) bool {
	pattern := fmt.Sprintf("^%d-[0-9]{5}$", facilityCode)
	match, err := regexp.MatchString(pattern, barcodeID)
	if err != nil {
		return false
	}
	return match
}

// IsValidHexID checks for a valid 8 digit hex value
// @deprecated
func IsValidHexID(barcodeID string) bool {
	match, err := regexp.MatchString("^[0-9a-fA-F]{8}$", barcodeID)
	if err != nil {
		return false
	}
	return match
}
