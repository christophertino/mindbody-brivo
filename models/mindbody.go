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
	Clients []user `json:"Clients"`
}

type user struct {
	FirstName string `json:"FirstName"`
	ID        string `json:"Id"`
	LastName  string `json:"LastName"`
}

// GetClients : Build MindBody data model with Client data
func (mb *MindBody) GetClients(config *Config, token string) {
	var client http.Client

	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.mindbodyonline.com/public/v6/client/clients", nil)
	if err != nil {
		log.Fatalln("Error creating HTTP request", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", config.MindbodySite)
	req.Header.Add("Api-Key", config.MindbodyAPIKey)
	req.Header.Add("Authorization", token)

	// Make request
	res, err := client.Do(req)
	if err != nil || res.StatusCode >= 400 {
		log.Fatalln("Error fetching MindBody clients", err, res.StatusCode)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response into Model
	err = json.Unmarshal(data, &mb)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}
}
