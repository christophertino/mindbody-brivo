/**
 * MindBody Data Model
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

import (
	"log"
	"net/http"

	async "github.com/christophertino/fiao-sync/utils"
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
	UniqueID    int64  `json:"UniqueId"` // Client’s unique system-generated ID
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
func (mb *MindBody) GetClients(config *Config, token string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.mindbodyonline.com/public/v6/client/clients", nil)
	if err != nil {
		log.Println("mindbody.GetClients: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", config.MindbodySite)
	req.Header.Add("Api-Key", config.MindbodyAPIKey)
	req.Header.Add("Authorization", token)

	if _, err = async.DoRequest(req, mb); err != nil {
		return err
	}

	return nil
}
