// Brivo Event Subscription Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import "time"

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
		ActorName       string `json:"actorName"`
		ObjectName      string `json:"objectName"`
		ObjectGroupName string `json:"objectGroupName"`
		ActionAllowed   bool   `json:"actionAllowed"`
		ObjectTypeID    int    `json:"objectTypeId"`
		Credentials     []struct {
			ID       int  `json:"id"`
			Disabled bool `json:"disabled"`
		} `json:"credentials"`
		CredentialObjectID int `json:"credentialObjectId"`
		DeviceTypeID       int `json:"deviceTypeId"`
	} `json:"eventData"`
}
