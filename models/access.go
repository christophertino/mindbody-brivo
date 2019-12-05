// Brivo Event Subscription Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"fmt"
	"time"

	db "github.com/christophertino/mindbody-brivo"
	utils "github.com/christophertino/mindbody-brivo"
	"github.com/gomodule/redigo/redis"
)

// Access stores Brivo user data when a site access event happens
type Access struct {
	Occurred time.Time `json:"occurred"`
	Actor    struct {
		ID   int    `json:"id"`   // The user's Brivo ID
		Name string `json:"name"` // The user's name (for debugging)
	} `json:"actor"`
	EventData struct {
		ActionAllowed bool               `json:"actionAllowed"` // Was the action allowed? (May not be used)
		ObjectName    string             `json:"objectName"`    // Access point name
		Credentials   []AccessCredential `json:"credentials"`
	} `json:"eventData"`
}

// AccessCredential holds the user credential associate with the access event
type AccessCredential struct {
	ID       int  `json:"id"` // Brivo credential ID
	Disabled bool `json:"disabled"`
}

// ProcessRequest takes a Brivo access requests and logs a client arrival in Mindbody
func (access *Access) ProcessRequest(config *Config, auth *Auth, conn redis.Conn) {
	// Unwrap the AccessCredential from the event data
	accessCredential, err := access.getAccessCredential()
	if err != nil {
		fmt.Printf("Error unwrapping AccessCredential\n%s\n", err)
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
		fmt.Printf("Error fetching user credential\n%s\n", err)
		return
	}

	// Validate that the credential has the correct facility access
	if !IsValidID(config.BrivoFacilityCode, cred.ReferenceID) {
		utils.Logger(fmt.Sprintf("Credential %s is not a valid ID", cred.ReferenceID))
		return
	}

	// Add the request timestamp to Redis
	today := time.Now().UTC().Format("2006-01-02 15:04:05")
	timestamp, err := db.Get(cred.ReferenceID, conn)
	if err == redis.ErrNil {
		// Timestamp not found in Redis. Add today's timestamp for the user
		db.Set(cred.ReferenceID, today, conn)
	} else {
		// Don't log a Mindbody arrival for the user if we have already seen them today
		if isToday(timestamp) {
			utils.Logger("User already has an active Mindbody arrival for today")
			return
		}
		// The user has an older arrival timestamp from a previous day. Update to today's date
		db.Set(cred.ReferenceID, today, conn)
	}

	// Check if the Mindbody token needs to be refreshed
	if time.Now().UTC().After(auth.MindBodyToken.ExpireTime) {
		if err := auth.MindBodyToken.getMindBodyToken(*config); err != nil {
			fmt.Println("Error refreshing Mindbody AUTH token:\n", err)
			return
		}
		utils.Logger("Refreshed Mindbody AUTH token")
	}

	// Log the user arrival in MINDBODY
	err = AddArrival(cred.ReferenceID, config, auth.MindBodyToken.AccessToken)
	if err != nil {
		fmt.Printf("Error logging arrival to MINDBODY\n%s\n", err)
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

// Checks to see if the timestamp matches the day, month and year of the current date
func isToday(timestamp string) bool {
	today := time.Now().UTC()
	date, err := time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		fmt.Printf("Error parsing timestamp \n%s\n", err)
		return false
	}
	return date.Day() == today.Day() && date.Month() == today.Month() && date.Year() == today.Year()
}
