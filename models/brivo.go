// Brivo Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	utils "github.com/christophertino/mindbody-brivo"
)

// Brivo stores Brivo API response data
type Brivo struct {
	Data     []BrivoUser `json:"data"`
	Offset   int         `json:"offset"`
	PageSize int         `json:"pageSize"`
	Count    int         `json:"count"`
}

// BrivoUser stores Brivo user data
type BrivoUser struct {
	ID           int           `json:"id,omitempty"`
	ExternalID   string        `json:"externalId"` // MINDBODY's ClientUniqueID (will not change)
	FirstName    string        `json:"firstName"`
	MiddleName   string        `json:"middleName"`
	LastName     string        `json:"lastName"`
	Suspended    bool          `json:"suspended"`
	CustomFields []CustomField `json:"customFields"`
	Emails       []email       `json:"emails"`
	PhoneNumbers []phoneNumber `json:"phoneNumbers"`
}

type email struct {
	Address   string `json:"address"`
	EmailType string `json:"type"`
}

type phoneNumber struct {
	Number     string `json:"number"`
	NumberType string `json:"type"`
}

var brivoIDSet = make(map[string]bool) // keep track of all existing IDs for quick lookup

// ListUsers builds the Brivo data model with user data
func (brivo *Brivo) ListUsers(brivoAPIKey string, brivoAccessToken string) error {
	var (
		count    = 0
		pageSize = 100 // Max 100
		results  []BrivoUser
	)

	utils.Logger("Fetching all Brivo users...")

	for {
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users?offset=%d&pageSize=%d", count, pageSize), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
		req.Header.Add("api-key", brivoAPIKey)

		if err = utils.DoRequest(req, brivo); err != nil {
			return err
		}

		// Stash external IDs in a set so we can check them later against MINDBODY users
		for _, element := range brivo.Data {
			brivoIDSet[element.ExternalID] = true
		}

		utils.Logger(fmt.Sprintf("Got Brivo users %d of %d", count, brivo.Count))

		results = append(results, brivo.Data...)
		count += brivo.PageSize
		if count >= brivo.Count {
			break
		}
	}
	brivo.Data = results

	utils.Logger(fmt.Sprintf("Completed fetching %d Brivo users.", brivo.Count))

	return nil
}

// ListUsersWithinGroup builds the Brivo data model with user data for a specific GroupID
func (brivo *Brivo) ListUsersWithinGroup(groupID int, brivoAPIKey string, brivoAccessToken string) error {
	var (
		count    = 0
		pageSize = 100 // Max 100
		results  []BrivoUser
	)

	utils.Logger(fmt.Sprintf("Fetching all Brivo users from group %d...", groupID))

	for {
		// Create HTTP request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/groups/%d/users?offset=%d&pageSize=%d", groupID, count, pageSize), nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %s", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
		req.Header.Add("api-key", brivoAPIKey)

		if err = utils.DoRequest(req, brivo); err != nil {
			return err
		}

		utils.Logger(fmt.Sprintf("Got Brivo user %d of %d", count, brivo.Count))

		results = append(results, brivo.Data...)
		count += brivo.PageSize
		if count >= brivo.Count {
			break
		}
	}
	brivo.Data = results

	utils.Logger(fmt.Sprintf("Completed fetching %d Brivo users.", brivo.Count))

	return nil
}

// BuildUser will build a Brivo user from MINDBODY user data
func (user *BrivoUser) BuildUser(mbUser MindBodyUser, config Config) {
	user.ExternalID = strconv.Itoa(mbUser.UniqueID)
	user.FirstName = mbUser.FirstName
	user.MiddleName = mbUser.MiddleName
	user.LastName = mbUser.LastName
	user.Suspended = (mbUser.Active == false || mbUser.Status != "Active")
	if mbUser.Email != "" {
		user.Emails = append(user.Emails, email{
			Address:   mbUser.Email,
			EmailType: "home",
		})
	}
	if mbUser.HomePhone != "" {
		user.PhoneNumbers = append(user.PhoneNumbers, phoneNumber{
			Number:     mbUser.HomePhone,
			NumberType: "home",
		})
	}
	if mbUser.MobilePhone != "" {
		user.PhoneNumbers = append(user.PhoneNumbers, phoneNumber{
			Number:     mbUser.MobilePhone,
			NumberType: "mobile",
		})
	}
	if mbUser.WorkPhone != "" {
		user.PhoneNumbers = append(user.PhoneNumbers, phoneNumber{
			Number:     mbUser.WorkPhone,
			NumberType: "work",
		})
	}

	// Create custom fields for MINDBODY barcodeID and user type
	barcodeID := GenerateCustomField(config.BrivoBarcodeFieldID, mbUser.ID)
	userType := GenerateCustomField(config.BrivoUserTypeFieldID, "Member")
	user.CustomFields = append(user.CustomFields, *barcodeID, *userType)
}

// CreateUser creates a new Brivo user
func (user *BrivoUser) CreateUser(brivoAPIKey string, brivoAccessToken string) error {
	// Check to see if user already exists
	if brivoIDSet[user.ExternalID] == true {
		return fmt.Errorf("User already exists")
	}

	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("Error building request body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.brivo.com/v1/api/users", bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	// Add new user ID to BrivoUser
	user.ID = int(r["id"].(float64))

	return nil
}

// UpdateCustomField updates the fieldValue for a particular Custom Field by fieldID
func (user *BrivoUser) UpdateCustomField(fieldID int, fieldValue string, brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(CustomField{Value: fieldValue})
	if err != nil {
		return fmt.Errorf("Error building request body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/custom-fields/%d", user.ID, fieldID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// AssignUserCredential assigns the credentialID to a user
func (user *BrivoUser) AssignUserCredential(credID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/credentials/%d", user.ID, credID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// AssignUserGroup assigns the user to groupID
func (user *BrivoUser) AssignUserGroup(groupID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/groups/%d/users/%d", groupID, user.ID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Retrieves a Brivo user by their UniqueID value
func (user *BrivoUser) getUserByID(uniqueID int, brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/external", uniqueID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	if err = utils.DoRequest(req, user); err != nil {
		return err
	}

	return nil
}

// Update an existing Brivo user
func (user *BrivoUser) updateUser(brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("Error building request body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d", user.ID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// Update the suspended status of the user in Brivo
func (user *BrivoUser) toggleSuspendedStatus(suspended bool, brivoAPIKey string, brivoAccessToken string) error {
	// Build request body JSON
	bytesMessage, err := json.Marshal(map[string]bool{"suspended": suspended})
	if err != nil {
		return fmt.Errorf("Error building request body json: %s", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d/suspended", user.ID), bytes.NewBuffer(bytesMessage))
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}

// DeleteUser will delete a Brivo user by ID
func (user *BrivoUser) DeleteUser(brivoAPIKey string, brivoAccessToken string) error {
	// Create HTTP request
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.brivo.com/v1/api/users/%d", user.ID), nil)
	if err != nil {
		return fmt.Errorf("Error creating HTTP request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+brivoAccessToken)
	req.Header.Add("api-key", brivoAPIKey)

	var r map[string]interface{}
	if err = utils.DoRequest(req, &r); err != nil {
		return err
	}

	return nil
}
