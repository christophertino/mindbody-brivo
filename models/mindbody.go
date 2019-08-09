/**
 * MindBody Data Model
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

import (
	"fmt"
	"log"
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
	Clients []mbUser `json:"Clients"`
}

type mbUser struct {
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
	Action      string `json:"Action"` // None,Added,Updated,Failed,Removed
}

// GetClients : Build MindBody data model with Client data
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
			log.Println("mindbody.GetClients: Error creating HTTP request", err)
			return err
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
