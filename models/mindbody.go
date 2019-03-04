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
	Clients []user
}

type user struct {
	FirstName string `json:"FirstName"`
	ID        string `json:"Id"`
	LastName  string `json:"LastName"`
}

func GetClients(cj *Config, token string) {
	var client http.Client

	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.mindbodyonline.com/public/v6/client/clients", nil)
	if err != nil {
		log.Fatalln("Error creating http request", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", cj.MindbodySite)
	req.Header.Add("Api-Key", cj.MindbodyAPIKey)
	req.Header.Add("Authorization", token)

	// Make request
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln("Error fetching MindBody clients", err)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response json
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}

	// Look for response errors
	if res.StatusCode >= 400 {
		log.Fatalln("API returned an error", res.StatusCode, result["Error"].(map[string]interface{})["Message"])
	}

	fmt.Printf("%+v", result)

	// Build MB Model and return
}
