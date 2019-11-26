// Brivo Event Subscription Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"fmt"
	"time"
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

// GetAccessCredential unwraps the AccessCredential from the Access event
func GetAccessCredential(creds []AccessCredential) (*AccessCredential, error) {
	if len(creds) > 0 {
		return &creds[0], nil
	}
	return &AccessCredential{}, fmt.Errorf("Access credential not found")
}
