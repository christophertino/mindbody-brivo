// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Webhook Event Data Model

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
	ClientID         string    `json:"clientId"`       // The client’s public ID (MindBodyUser.ID)
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
		brivoUser BrivoUser
		mbUser    MindBodyUser
	)
	// Query the user on Brivo using the MINDBODY ExternalID
	var existingUser BrivoUser
	err := existingUser.getUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch e := err.(type) {
	// User already exists: Update user
	case nil:
		// Build event data into Brivo user
		mbUser.buildUser(event.EventData)
		brivoUser.BuildUser(mbUser)
		// Update Brivo ID from existing user
		brivoUser.ID = existingUser.ID

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
			brivoUser.BuildUser(mbUser)

			// Create a new user
			if err := brivoUser.CreateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Error creating user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Create new Brivo credential for this user
			cred := GenerateCredential(brivoUser.ExternalID)
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
	// Query the user data on Brivo using the MINDBODY ExternalID
	var brivoUser BrivoUser
	if err := brivoUser.getUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Brivo user %s does not exist. Error: %s", event.EventData.ClientID, err)
	}
	// Put Brivo user in suspended status
	if err := brivoUser.toggleSuspendedStatus(true, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Error deactivating user %s: %s", brivoUser.ExternalID, err)
	}

	fmt.Printf("Brivo user %s suspended status set to true\n", brivoUser.ExternalID)

	return nil
}
