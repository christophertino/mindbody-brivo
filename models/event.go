// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Webhook Event Data Model

package models

import (
	"fmt"
	"time"

	async "github.com/christophertino/mindbody-brivo/utils"
	"github.com/google/go-cmp/cmp"
)

// Event : MINDBODY webhook event data
type Event struct {
	MessageID                        string        `json:"messageId"`
	EventID                          string        `json:"eventId"`
	EventSchemaVersion               float64       `json:"eventSchemaVersion"`
	EventInstanceOriginationDateTime time.Time     `json:"eventInstanceOriginationDateTime"`
	EventData                        EventUserData `json:"eventData"`
}

// EventUserData : MINDBODY user data sent by webhook events
type EventUserData struct {
	SiteID           int       `json:"siteId"`
	ClientID         string    `json:"clientId"`       // The client’s public ID
	ClientUniqueID   int       `json:"clientUniqueId"` // The client’s guaranteed unique ID
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

// CreateOrUpdateUser : Webhook event handler for client.updated and client.created
func (event *Event) CreateOrUpdateUser(config Config, auth Auth) error {
	var (
		brivoUser BrivoUser
		mbUser    MindBodyUser
	)
	// Query the user on Brivo using the MINDBODY ExternalID
	var existingUser BrivoUser
	err := existingUser.getUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch err := err.(type) {
	// User already exists: Update user
	case nil:
		// Build event data into Brivo user
		mbUser.buildUser(event.EventData)
		brivoUser.buildUser(mbUser)
		// Update Brivo ID from existing user
		brivoUser.ID = existingUser.ID
		// Check diff to see if update is needed
		if !cmp.Equal(existingUser, brivoUser) {
			if err := brivoUser.updateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Event.CreateOrUpdateUser: Error updating user %s\n%s", brivoUser.ExternalID, err)
			}
			// Handle account re-activation
			if existingUser.Suspended && !brivoUser.Suspended {
				if err := brivoUser.toggleSuspendedStatus(false, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
					return fmt.Errorf("Event.CreateOrUpdateUser: Error re-activating user %s\n%s", brivoUser.ExternalID, err)
				}
			}
		} else {
			return fmt.Errorf("Event.CreateOrUpdateUser: UserID %s does not have any properties to update", brivoUser.ExternalID)
		}
		return nil
	// Handle specific error codes from the API server
	case *async.JSONError:
		// User does not exist: Create new user
		if err.Code == 404 {
			// Build event data into Brivo user
			mbUser.buildUser(event.EventData)
			brivoUser.buildUser(mbUser)

			// Create a new user
			if err := brivoUser.createUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Event.CreateOrUpdateUser: Error creating user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Create new Brivo credential for this user
			cred := generateCredential(brivoUser.ExternalID)
			credID, err := cred.createCredential(config.BrivoAPIKey, auth.BrivoToken.AccessToken)
			if err != nil {
				return fmt.Errorf("Event.CreateOrUpdateUser: Error creating credential for user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Assign credential to user
			if err := brivoUser.assignUserCredential(credID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Event.CreateOrUpdateUser: Error assigning credential to user %s with error: %s", brivoUser.ExternalID, err)
			}

			// Assign user to group
			if err := brivoUser.assignUserGroup(config.BrivoMemberGroupID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				return fmt.Errorf("Event.CreateOrUpdateUser: Error assigning user %s to group with error: %s", brivoUser.ExternalID, err)
			}

			fmt.Printf("Event.CreateOrUpdateUser: Successfully created Brivo user %s", brivoUser.ExternalID)
			return nil
		}
		return fmt.Errorf("Event.CreateOrUpdateUser: Error %s", err.Body)
	// General error
	default:
		return fmt.Errorf("Event.CreateOrUpdateUser: Error %s", err)
	}
}

// DeactivateUser : Webhook event handler for client.deactivated
func (event *Event) DeactivateUser(config Config, auth Auth) error {
	// Query the user data on Brivo using the MINDBODY ExternalID
	var brivoUser BrivoUser
	if err := brivoUser.getUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Event.DeactivateUser: Brivo user %s does not exist.\n%s", event.EventData.ClientID, err)
	}
	// Put Brivo user in suspended status
	if err := brivoUser.toggleSuspendedStatus(true, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		return fmt.Errorf("Event.DeactivateUser: Error deactivating user %s\n%s", brivoUser.ExternalID, err)
	}
	return nil
}
