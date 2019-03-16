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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Brivo client data
type Brivo struct {
	Data     []brivoUser `json:"data"`
	Offset   int         `json:"offset"`
	PageSize int         `json:"pageSize"`
	Count    int         `json:"count"`
}

type brivoUser struct {
	ID           int           `json:"id"`
	ExternalID   string        `json:"externalId"` // Barcode ID from MindBody to link accounts
	FirstName    string        `json:"firstName"`
	MiddleName   string        `json:"middleName"`
	LastName     string        `json:"lastName"`
	Suspended    bool          `json:"suspended"`
	CustomFields []customField `json:"customFields"`
	Emails       []email       `json:"emails"`
	PhoneNumbers []phoneNumber `json:"phoneNumbers"`
}

type customField struct {
	fieldName string `json:"fieldName"`
	fieldType string `json:"fieldType"`
}

type email struct {
	address   string `json:"address"`
	emailType string `json:"type"`
}

type phoneNumber struct {
	number     string `json:"number"`
	numberType string `json:"type"`
}

// ListUsers : Build Brivo data model with user data
func (brivo *Brivo) ListUsers(config *Config, token string) error {
	var client http.Client

	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.brivo.com/v1/api/users", nil)
	if err != nil {
		log.Println("brivo.ListUsers: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-key", config.BrivoAPIKey)
	req.Header.Add("Authorization", "Bearer "+token)

	// Make request
	res, err := client.Do(req)
	if err != nil {
		log.Println("brivo.ListUsers: Error making request", err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		log.Println("brivo.ListUsers: Error fetching Brivo users", err)
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		bodyString := string(bodyBytes)
		return fmt.Errorf("brivo.ListUsers: Error fetching Brivo users with status code: %d, and body: %s", res.StatusCode, bodyString)
	}

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("brivo.ListUsers: Error reading response", err)
		return err
	}

	// Build response into Model
	err = json.Unmarshal(data, &brivo)
	if err != nil {
		log.Println("brivo.ListUsers: Error unmarshalling json", err)
		return err
	}

	return nil
}

// BuildBrivoUsers : Convert MB users to Brivo users
func (brivo *Brivo) BuildBrivoUsers(mb *MindBody) {
	// var brivoUsers = brivo.Data
	var mbUsers []brivoUser

	// Map MindBody fields to Brivo
	for i := 0; i < len(mb.Clients); i++ {
		var (
			user      brivoUser
			userEmail email
			userPhone phoneNumber
		)
		user.ExternalID = mb.Clients[i].ID // barcode ID
		user.FirstName = mb.Clients[i].FirstName
		user.MiddleName = mb.Clients[i].MiddleName
		user.LastName = mb.Clients[i].LastName
		user.Suspended = (mb.Clients[i].Active == false || mb.Clients[i].Status != "Active")

		userEmail.address = mb.Clients[i].Email
		userEmail.emailType = "home"
		user.Emails = append(user.Emails, userEmail)

		if mb.Clients[i].HomePhone != "" {
			userPhone.number = mb.Clients[i].HomePhone
			userPhone.numberType = "home"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if mb.Clients[i].MobilePhone != "" {
			userPhone.number = mb.Clients[i].MobilePhone
			userPhone.numberType = "mobile"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if mb.Clients[i].WorkPhone != "" {
			userPhone.number = mb.Clients[i].WorkPhone
			userPhone.numberType = "work"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}

		mbUsers = append(mbUsers, user)
	}

	// If user doesn't exist, create new Brivo user

}
