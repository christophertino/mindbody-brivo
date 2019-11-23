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
	ObjectType     string `json:"objectType"`
	UUID           string `json:"uuid"`
	AccountID      int    `json:"accountId"`
	SecurityAction struct {
		SecurityActionID int    `json:"securityActionId"`
		Action           string `json:"action"`
		Exception        bool   `json:"exception"`
	} `json:"securityAction"`
	Actor struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"actor"`
	EventObject struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		DeviceType struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"deviceType"`
	} `json:"eventObject"`
	Site struct {
		ID       int    `json:"id"`
		SiteName string `json:"siteName"`
	} `json:"site"`
	Occurred  time.Time `json:"occurred"`
	EventData struct {
		ActorName          string             `json:"actorName"`
		ObjectName         string             `json:"objectName"`
		ObjectGroupName    string             `json:"objectGroupName"`
		ActionAllowed      bool               `json:"actionAllowed"`
		ObjectTypeID       int                `json:"objectTypeId"`
		Credentials        []AccessCredential `json:"credentials"`
		CredentialObjectID int                `json:"credentialObjectId"`
		DeviceTypeID       int                `json:"deviceTypeId"`
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
