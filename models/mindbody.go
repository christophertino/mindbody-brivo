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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

type mbError struct {
	Error struct {
		Message string `json:"Message"`
		Code    string `json:"Code"`
	} `json:"Error"`
}

// GetClients : Build MindBody data model with Client data
func (mb *MindBody) GetClients(config *Config, token string) error {
	var client http.Client

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

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		bodyString := string(bodyBytes)
		return fmt.Errorf("mindbody.GetClients: Error fetching MB clients with status code: %d, and body: %s", res.StatusCode, bodyString)
	}

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("mindbody.GetClients: Error reading response", err)
		return err
	}

	// Build response into Model
	err = json.Unmarshal(data, &mb)
	if err != nil {
		log.Println("mindbody.GetClients: Error unmarshalling json", err)
		return err
	}

	return nil
}
