// Brivo Event Subscription Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"fmt"
	"strconv"
	"time"

	utils "github.com/christophertino/mindbody-brivo"
)

// Access stores Brivo user data when a site access event happens
type Access struct {
	Occurred  time.Time `json:"occurred"`
	EventData struct {
		ActionAllowed bool               `json:"actionAllowed"` // Was the action allowed? (May not be used)
		ActorName     string             `json:"actorName"`     // The user's name (for debugging)
		ObjectName    string             `json:"objectName"`    // Access point name
		Credentials   []AccessCredential `json:"credentials"`
	} `json:"eventData"`
}

// AccessCredential holds the user credential associate with the access event
type AccessCredential struct {
	ID       int  `json:"id"`
	Disabled bool `json:"disabled"`
}

// ProcessRequest takes a Brivo access requests and logs a client arrival in Mindbody
func (access *Access) ProcessRequest(config *Config, auth *Auth) {
	// Unwrap the AccessCredential from the event data
	accessCredential, err := access.getAccessCredential()
	if err != nil {
		utils.Logger(fmt.Sprintf("Error unwrapping AccessCredential\n%s", err))
		return
	}

	// Check if the Brivo token needs to be refreshed
	if time.Now().UTC().After(auth.BrivoToken.ExpireTime) {
		if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
			fmt.Println("Error refreshing Brivo AUTH token:\n", err)
			return
		}
		utils.Logger("Refreshed Brivo AUTH token")
	}

	// Fetch the user Credential by Brivo ID
	cred, err := GetCredentialByID(accessCredential.ID, config.BrivoAPIKey, auth.BrivoToken.AccessToken)
	if err != nil {
		utils.Logger(fmt.Sprintf("Error fetching user credential\n%s", err))
		return
	}

	// Validate that the credential has the correct facility access
	if !IsValidID(config.BrivoFacilityCode, cred.ReferenceID) {
		utils.Logger(fmt.Sprintf("Credential %s is not a valid ID", cred.ReferenceID))
		return
	}

	// TODO: Add the request timestamp to Redis

	// Check if the Mindbody token needs to be refreshed
	if time.Now().UTC().After(auth.MindBodyToken.ExpireTime) {
		if err := auth.MindBodyToken.getMindBodyToken(*config); err != nil {
			fmt.Println("Error refreshing Mindbody AUTH token:\n", err)
			return
		}
		utils.Logger("Refreshed Mindbody AUTH token")
	}

	// Log the user arrival in MINDBODY
	userID, _ := strconv.Atoi(cred.ReferenceID)
	err = AddArrival(userID, config, auth.MindBodyToken.AccessToken)
	if err != nil {
		utils.Logger(fmt.Sprintf("Error logging arrival to MINDBODY\n%s", err))
		return
	}
}

// Unwraps the AccessCredential from the Access event
func (access *Access) getAccessCredential() (*AccessCredential, error) {
	creds := access.EventData.Credentials
	if len(creds) > 0 {
		return &creds[0], nil
	}
	return &AccessCredential{}, fmt.Errorf("Access credential not found")
}
