// Brivo Event Subscription Data Model

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
func (access *Access) ProcessRequest(config *Config, auth *Auth, pool *redis.Pool) {
	// Get a connection from the Redis pool and close it when the handler is done
	conn := pool.Get()
	defer conn.Close()

	// Unwrap the AccessCredential from the event data
	accessCredential, err := access.getAccessCredential()
	if err != nil {
		fmt.Printf("Error unwrapping AccessCredential: %s\n", err)
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
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	timestamp, err := db.Get(cred.ReferenceID, conn)
	utils.Logger(fmt.Sprintf("Redis: Fetch for key %s returned %s", cred.ReferenceID, timestamp))
	if err == redis.ErrNil {
		// Timestamp not found in Redis. Add current timestamp for the user
		db.Set(cred.ReferenceID, now, conn)
		timestamp = now
		utils.Logger(fmt.Sprintf("Redis: Creating new key %s with timestamp %s", cred.ReferenceID, timestamp))
	} else if err != nil {
		utils.Logger(fmt.Sprintf("Redis: Fetch for key %s returned error %s", cred.ReferenceID, err))
		return
	} else {
		// Don't log a Mindbody arrival for the user if we have seen them within the last 30min
		if isActiveTimestamp(timestamp) {
			utils.Logger(fmt.Sprintf("Redis: User %s already has an active Mindbody arrival timestamp", cred.ReferenceID))
			return
		}
		// The user has an older arrival timestamp from a previous time. Update to current timestamp
		db.Set(cred.ReferenceID, now, conn)
		utils.Logger(fmt.Sprintf("Redis: Setting timestamp for existing key %s to %s", cred.ReferenceID, timestamp))
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
		fmt.Printf("Error logging arrival to MINDBODY for user %s\n%s", cred.ReferenceID, err)
		return
	}
}

// Unwraps the AccessCredential from the Access event
func (access *Access) getAccessCredential() (*AccessCredential, error) {
	creds := access.EventData.Credentials
	// Filter out empty AccessCredential objects and instances where the Credential ID is 0
	if len(creds) > 0 && creds[0].ID != 0 {
		return &creds[0], nil
	}
	return &AccessCredential{}, fmt.Errorf("Access credential not found")
}

// Checks to see if the timestamp is active within the past 30min
// @TODO: Make the timeout value an ENV property
func isActiveTimestamp(timestamp string) bool {
	now := time.Now().UTC()
	lastVisit, err := time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		fmt.Printf("Error parsing timestamp \n%s\n", err)
		return false
	}
	// Has 30min passed since the last visit?
	if now.Before(lastVisit.Add(time.Minute * 30)) {
		return true
	}
	return false
}

// Checks to see if the timestamp matches the day, month and year of the current date
// @deprecated
func isToday(timestamp string) bool {
	today := time.Now().UTC()
	date, err := time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		fmt.Printf("Error parsing timestamp \n%s\n", err)
		return false
	}
	return date.Day() == today.Day() && date.Month() == today.Month() && date.Year() == today.Year()
}
