/**
 * Webhook Event Data Model
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

import "time"

// Event : Webhook event data
type Event struct {
	MessageID                        string    `json:"messageId"`
	EventID                          string    `json:"eventId"`
	EventSchemaVersion               int       `json:"eventSchemaVersion"`
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
