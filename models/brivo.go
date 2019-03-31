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
	"encoding/hex"
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
	ID           int           `json:"id,omitempty"`
	ExternalID   string        `json:"externalId"` // Barcode ID from MindBody to link accounts
	FirstName    string        `json:"firstName"`
	MiddleName   string        `json:"middleName"`
	LastName     string        `json:"lastName"`
	Suspended    bool          `json:"suspended"`
	CustomFields []customField `json:"customFields,omitempty"`
	Emails       []email       `json:"emails"`
	PhoneNumbers []phoneNumber `json:"phoneNumbers"`
}

type customField struct {
	FieldName string `json:"fieldName"`
	FieldType string `json:"fieldType"`
}

type email struct {
	Address   string `json:"address"`
	EmailType string `json:"type"`
}

type phoneNumber struct {
	Number     string `json:"number"`
	NumberType string `json:"type"`
}

// BrivoError : Standard Brivo error struct
type BrivoError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
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
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("api-key", config.BrivoAPIKey)

	// Make request
	res, err := client.Do(req)
	if err != nil {
		log.Println("brivo.ListUsers: Error making request", err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
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
func (brivo *Brivo) BuildBrivoUsers(mb *MindBody, config *Config, auth *Auth) {
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

		userEmail.Address = mb.Clients[i].Email
		userEmail.EmailType = "home"
		user.Emails = append(user.Emails, userEmail)

		if mb.Clients[i].HomePhone != "" {
			userPhone.Number = mb.Clients[i].HomePhone
			userPhone.NumberType = "home"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if mb.Clients[i].MobilePhone != "" {
			userPhone.Number = mb.Clients[i].MobilePhone
			userPhone.NumberType = "mobile"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}
		if mb.Clients[i].WorkPhone != "" {
			userPhone.Number = mb.Clients[i].WorkPhone
			userPhone.NumberType = "work"
			user.PhoneNumbers = append(user.PhoneNumbers, userPhone)
		}

		mbUsers = append(mbUsers, user)

		cred := Credential{
			CredentialFormat: CredentialFormat{
				ID: 110, // Unknown Format
			},
			ReferenceID:       user.ExternalID, // barcode ID
			EncodedCredential: hex.EncodeToString([]byte(user.ExternalID)),
		}

		id, err := cred.createCredential(config, auth)
		if err != nil {
			log.Fatalln("brivo.BuildBrivoUsers: Error creating credential \n", err)
		}

		fmt.Println(id)

	}

	// If user doesn't exist

	// create new credential

	// create new Brivo user

	// assign credential to user

	// Assign user to group

}
