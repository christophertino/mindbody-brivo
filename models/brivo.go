/**
 * Brivo Data Model
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
	"time"
)

// Brivo client data
type Brivo struct {
	Data []struct {
		ID                 int           `json:"id"`
		FirstName          string        `json:"firstName"`
		MiddleName         string        `json:"middleName"`
		LastName           string        `json:"lastName"`
		Created            time.Time     `json:"created"`
		Updated            time.Time     `json:"updated"`
		Suspended          bool          `json:"suspended"`
		BleTwoFactorExempt bool          `json:"bleTwoFactorExempt"`
		CustomFields       []interface{} `json:"customFields"`
		Emails             []interface{} `json:"emails"`
		PhoneNumbers       []interface{} `json:"phoneNumbers"`
	} `json:"data"`
	Offset   int `json:"offset"`
	PageSize int `json:"pageSize"`
	Count    int `json:"count"`
}

// ListUsers : Build Brivo data model with user data
func (brivo *Brivo) ListUsers(config *Config, token string) {
	var client http.Client

	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.brivo.com/v1/api/users", nil)
	if err != nil {
		log.Fatalln("Error creating HTTP request", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-key", config.BrivoAPIKey)
	req.Header.Add("Authorization", "Bearer "+token)

	// Make request
	res, err := client.Do(req)
	if err != nil || res.StatusCode >= 400 {
		log.Fatalln("Error fetching Brivo users", err, res.StatusCode)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response into Model
	err = json.Unmarshal(data, &brivo)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}
}
