// Webhook Event Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"fmt"
	"net/http"
	"time"

	utils "github.com/christophertino/mindbody-brivo"
	"github.com/google/go-cmp/cmp"
)

// Event stores MINDBODY webhook event data
type Event struct {
	MessageID                        string        `json:"messageId"`
	EventID                          string        `json:"eventId"`
	EventSchemaVersion               float64       `json:"eventSchemaVersion"`
	EventInstanceOriginationDateTime time.Time     `json:"eventInstanceOriginationDateTime"`
	EventData                        EventUserData `json:"eventData"`
}

// EventUserData stores MINDBODY user data sent by webhook events
type EventUserData struct {
	SiteID           int       `json:"siteId"`
	ClientID         string    `json:"clientId"`       // The client’s public ID (MindBodyUser.ID) used for barcode credential
	ClientUniqueID   int       `json:"clientUniqueId"` // The client’s guaranteed unique ID (MindBodyUser.UniqueID)
	CreationDateTime time.Time `json:"creationDateTime"`
	FirstName        string    `json:"firstName"`
	MiddleName       string    `json:"middleName"` // Currently not supported
	LastName         string    `json:"lastName"`
	Email            string    `json:"email"`
	MobilePhone      string    `json:"mobilePhone"`
	HomePhone        string    `json:"homePhone"`
	WorkPhone        string    `json:"workPhone"`
	Status           string    `json:"status"` // Declined,Non-Member,Active,Expired,Suspended,Terminated
}

// CreateOrUpdateUser is a webhook event handler for client.updated and client.created
func (event *Event) CreateOrUpdateUser(config Config, auth Auth) error {
	var (
		brivoUser    BrivoUser
		mbUser       MindBodyUser
		customFields CustomFields
	)
	// Query the user on Brivo using the MINDBODY ClientUniqueID
	var existingUser BrivoUser
	err := existingUser.getUserByID(event.EventData.ClientUniqueID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	// User already exists: Update user
	case nil:
		// Build event data into Brivo user
		mbUser.buildUser(event.EventData)
		brivoUser.BuildUser(mbUser, config)

		// Update Brivo ID from existing user
		brivoUser.ID = existingUser.ID

		// Fetch custom fields for the existing user on Brivo as the barcode ID may have changed on MINDBODY
		if err := customFields.GetCustomFieldsForUser(brivoUser.ID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
			return fmt.Errorf("Error fetching custom fields for user %s: %s", brivoUser.ExternalID, err)
		}
		existingUser.CustomFields = customFields.Data

		// Check diff to see if update is needed
		if !cmp.Equal(existingUser, brivoUser) {
			if err := brivoUser.updateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error updating user %s: %s", brivoUser.ExternalID, err)
			}

			// Handle account re-activation
			if existingUser.Suspended != brivoUser.Suspended {
				if err := brivoUser.toggleSuspendedStatus(brivoUser.Suspended, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
					return fmt.Errorf("Error changing suspended status for user %s: %s", brivoUser.ExternalID, err)
				}
				fmt.Printf("Brivo user %s suspended status set to %t\n", brivoUser.ExternalID, brivoUser.Suspended)
			}

			// Check if the barcode ID has changed
			existingBarcode, _ := GetFieldValue(config.BrivoBarcodeFieldID, existingUser.CustomFields)
			newBarcode, _ := GetFieldValue(config.BrivoBarcodeFieldID, brivoUser.CustomFields)
			if existingBarcode != newBarcode {
				// Check to see if the credential exists for this user
				oldCred, err := getCredentialByID(existingBarcode, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
				if err == nil {
					// Delete the old credential
					if err := oldCred.DeleteCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
						fmt.Printf("Error deleting Credential ID %s with message: %s\n", existingBarcode, err)
					}
				} else {
					fmt.Printf("Credential ID %s not found: %s\n", existingBarcode, err)
				}

				// Update barcode ID in custom fields
				if err := brivoUser.UpdateCustomField(config.BrivoBarcodeFieldID, newBarcode, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
					return fmt.Errorf("Error updating custom field for user %s with error: %s", brivoUser.ExternalID, err)
				}

				// Create new Brivo credential for this user based on new Barcode ID
				cred := GenerateCredential(newBarcode)
				credID, err := cred.CreateCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
				if err != nil {
					return fmt.Errorf("Error creating credential for user %s with error: %s", brivoUser.ExternalID, err)
				}

				// Assign new credential to user
				if err := brivoUser.AssignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
					return fmt.Errorf("Error assigning credential to user %s with error: %s", brivoUser.ExternalID, err)
				}
			}
			fmt.Printf("Brivo user %s updated successfully\n", brivoUser.ExternalID)
		} else {
			fmt.Printf("UserID %s does not have any properties to update\n", brivoUser.ExternalID)
		}
		return nil
	// Handle specific error codes from the API server
	case *utils.JSONError:
		// Unauthorized: Invalid token
		if e.Code == 401 {
			return fmt.Errorf("%d", http.StatusUnauthorized)
		}
		// User does not exist: Create new user
		if e.Code == 404 {
			// Build event data into Brivo user
			mbUser.buildUser(event.EventData)
			brivoUser.BuildUser(mbUser, config)

			// Create a new user
			if err := brivoUser.CreateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error creating user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Fetch the barcode ID from CustomFields
			barcodeID, err := GetFieldValue(config.BrivoBarcodeFieldID, brivoUser.CustomFields)
			if err != nil {
				return fmt.Errorf("Error fetching barcode ID for user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Add barcode ID to Brivo custom fields
			if err := brivoUser.UpdateCustomField(config.BrivoBarcodeFieldID, barcodeID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error updating custom field for user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Add "Member" type to Brivo custom fields
			if err := brivoUser.UpdateCustomField(config.BrivoUserTypeFieldID, "Member", config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error updating custom field for user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Create new Brivo credential for this user
			cred := GenerateCredential(barcodeID)
			credID, err := cred.CreateCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
			if err != nil {
				return fmt.Errorf("Error creating credential for user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Assign credential to user
			if err := brivoUser.AssignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error assigning credential to user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Assign user to group
			if err := brivoUser.AssignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error assigning user %s to group with error: %s", brivoUser.ExternalID, err)
			}

			fmt.Printf("Successfully created Brivo user %s\n", brivoUser.ExternalID)
			return nil
		}
		return fmt.Errorf("%s", e.Body)
	// General error
	default:
		return err
	}
}

// DeactivateUser is a webhook event handler for client.deactivated
func (event *Event) DeactivateUser(config Config, auth Auth) error {
	// Query the user data on Brivo using the MINDBODY ClientUniqueID
	var brivoUser BrivoUser
	if err := brivoUser.getUserByID(event.EventData.ClientUniqueID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Brivo user %d does not exist. Error: %s", event.EventData.ClientUniqueID, err)
	}
	// Put Brivo user in suspended status
	if err := brivoUser.toggleSuspendedStatus(true, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Error deactivating user %s: %s", brivoUser.ExternalID, err)
	}

	fmt.Printf("Brivo user %s suspended status set to true\n", brivoUser.ExternalID)

	return nil
}
