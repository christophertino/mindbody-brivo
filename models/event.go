// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Webhook Event Data Model

package models

import (
	"fmt"
	"time"

	async "github.com/christophertino/mindbody-brivo/utils"
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
	ClientID         string    `json:"clientId"`
	ClientUniqueID   int       `json:"clientUniqueId"`
	CreationDateTime time.Time `json:"creationDateTime"`
	Status           string    `json:"status"`
	FirstName        string    `json:"firstName"`
	LastName         string    `json:"lastName"`
	Email            string    `json:"email"`
	MobilePhone      string    `json:"mobilePhone"`
	HomePhone        string    `json:"homePhone"`
	WorkPhone        string    `json:"workPhone"`
}

var (
	mb    MindBody
	brivo Brivo
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
	err := brivo.GetUserByID(event.EventData.ClientID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	switch err := err.(type) {
	case *async.JSONError:
		if err.Code == 404 {
			// user does not exist
			fmt.Println("event.UpdateUser: Brivo user does not exist. Creating new user...")
			event.CreateUser(config, auth)
			return nil
		}
	default:
		return err
	}

	// Build event data into Brivo user
	// Check diff to see if update is needed?
	// Update https://apidocs.brivo.com/#api-User-UpdateUser
	return nil
}

// DeactivateUser : Webhook event handler for client.deactivated
func (event *Event) DeactivateUser(config Config, auth Auth) error {
	// Put brivo user in suspended status
	//https://apidocs.brivo.com/#api-User-RetrieveUserByExternal
	//https://apidocs.brivo.com/#api-User-ToggleSuspendedStatus
	return nil
}
