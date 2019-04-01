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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	async "github.com/christophertino/fiao-sync/utils"
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

// ListUsers : Build Brivo data model with user data
func (brivo *Brivo) ListUsers(config *Config, token string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.brivo.com/v1/api/users", nil)
	if err != nil {
		log.Println("brivo.ListUsers: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("api-key", config.BrivoAPIKey)

	if _, err = async.DoRequest(req, brivo); err != nil {
		return err
	}

	return nil
}

// BuildBrivoUsers : Convert MB users to Brivo users
func (brivo *Brivo) BuildBrivoUsers(mb *MindBody, config *Config, auth *Auth) {
	// var brivoUsers = brivo.Data
	// var mbUsers []brivoUser

	// Map MindBody fields to Brivo
	// for i := 0; i < len(mb.Clients); i++ {
	for i := 0; i < 1; i++ {
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

		// mbUsers = append(mbUsers, user)

		// Create new Brivo credential for this user
		cred := Credential{
			CredentialFormat: CredentialFormat{
				ID: 110, // Unknown Format
			},
			ReferenceID:       user.ExternalID, // barcode ID
			EncodedCredential: hex.EncodeToString([]byte(user.ExternalID)),
		}
		credID, err := cred.createCredential(config, auth)
		if err != nil {
			log.Fatalln("brivo.BuildBrivoUsers: Error creating credential \n", err)
		}

		// Create a new user
		userID, err := createUser(&user, credID, config, auth.BrivoToken.AccessToken)
		if err != nil {
			log.Fatalln("brivo.BuildBrivoUsers: Error creating user \n", err)
		}

		// Assign credential to user
		assignUserCredential(userID, credID, config, auth.BrivoToken.AccessToken)

		// Assign user to group
	}
}

// Create a new Brivo user
func createUser(user *brivoUser, credentialID int32, config *Config, token string) (int32, error) {
	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		log.Println("brivo.createUser: Error building POST body json", err)
		return 0, err
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", "https://api.brivo.com/v1/api/users", bytes.NewBuffer(bytesMessage))
	if err != nil {
		log.Println("brivo.createUser: Error creating HTTP request", err)
		return 0, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("api-key", config.BrivoAPIKey)

	var r map[string]interface{}
	resp, err := async.DoRequest(req, &r)
	if err != nil {
		return 0, err
	}

	// Return the new user ID
	return resp.(map[string]interface{})["id"].(int32), nil
}

func assignUserCredential(userID int32, credID int32, config *Config, token string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/credentials/%d", userID, credID), nil)
	if err != nil {
		log.Println("brivo.ListUsers: Error creating HTTP request", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("api-key", config.BrivoAPIKey)

	var r map[string]interface{}
	if _, err = async.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}
