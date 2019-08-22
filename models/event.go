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

// Event : Webhook event data
type Event struct {
	MessageID                        string    `json:"messageId"`
	EventID                          string    `json:"eventId"`
	EventSchemaVersion               float64   `json:"eventSchemaVersion"`
	EventInstanceOriginationDateTime time.Time `json:"eventInstanceOriginationDateTime"`
	EventData                        userData  `json:"eventData"`
}

type userData struct {
	SiteID           int       `json:"siteId"`
	ClientID         string    `json:"clientId"`       // The client’s public ID
	ClientUniqueID   int       `json:"clientUniqueId"` // The client’s guaranteed unique ID
	CreationDateTime time.Time `json:"creationDateTime"`
	FirstName        string    `json:"firstName"`
	LastName         string    `json:"lastName"`
	Email            string    `json:"email"`
	MobilePhone      string    `json:"mobilePhone"`
	HomePhone        string    `json:"homePhone"`
	WorkPhone        string    `json:"workPhone"`
	Status           string    `json:"status"` // Declined,Non-Member,Active,Expired,Suspended,Terminated
}

var (
	mb        MindBody
	brivo     Brivo
	brivoUser BrivoUser
)

// CreateUser : Webhook event handler for client.created
func (event *Event) CreateUser(config Config, auth Auth) error {
	// check if user already exists on Brivo
	//if exists, send to our update function (below)
	//if new, call series of create functions in Brivo
	return nil
}

// UpdateUser : Webhook event handler for client.updated
func (event *Event) UpdateUser(config Config, auth Auth) error {
	// Query the user data on Brivo using the MINDBODY ExternalID
	var existingUser BrivoUser
	err := existingUser.GetUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch err := err.(type) {
	// Handle specific error codes from the API server
	case *async.JSONError:
		// user does not exist
		if err.Code == 404 {
			fmt.Println("Event.UpdateUser: Brivo user does not exist. Creating new user...")
			event.CreateUser(config, auth)
			return nil
		}
		fmt.Printf("Event.UpdateUser error %s\n", err.Body)
	case nil:
		// Build event data into Brivo user
		brivoUser.BuildUser(event.EventData, existingUser.ID)
		// Check diff to see if update is needed
		if !cmp.Equal(existingUser, brivoUser) {
			if err := brivoUser.UpdateUser(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
				fmt.Printf("Event.UpdateUser: Error updating user %s\n", brivoUser.ExternalID)
				return err
			}
			// Handle account re-activation
			if existingUser.Suspended && !brivoUser.Suspended {
				if err := brivoUser.ToggleSuspendedStatus(false, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
					fmt.Printf("Event.UpdateUser: Error re-activating user %s\n", brivoUser.ExternalID)
					return err
				}
			}
		} else {
			return fmt.Errorf("Event.UpdateUser: UserID %s does not have any properties to update", brivoUser.ExternalID)
		}

		return nil
	}

	// General error
	return fmt.Errorf("Event.UpdateUser: Error %s", err)
}

// DeactivateUser : Webhook event handler for client.deactivated
func (event *Event) DeactivateUser(config Config, auth Auth) error {
	// Query the user data on Brivo using the MINDBODY ExternalID
	var brivoUser BrivoUser
	if err := brivoUser.GetUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		fmt.Printf("Event.DeactivateUser: Brivo user %s does not exist.\n", event.EventData.ClientID)
		return err
	}
	// Put Brivo user in suspended status
	if err := brivoUser.ToggleSuspendedStatus(true, config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
		fmt.Printf("Event.DeactivateUser: Error deactivating user %s\n", brivoUser.ExternalID)
		return err
	}
	return nil
}
