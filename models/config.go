// Configuration Data Model
//
// Copyright 2019 Christopher Tino. All rights reserved.

package models

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config stores environment settings imported from .env
type Config struct {
	BrivoUsername          string
	BrivoPassword          string
	BrivoClientID          string
	BrivoClientSecret      string
	BrivoAPIKey            string
	BrivoFacilityCode      int
	BrivoSiteID            int
	BrivoMemberGroupID     int
	BrivoBarcodeFieldID    int
	BrivoUserTypeFieldID   int
	BrivoRateLimit         int
	BrivoClientCredentials string

	MindbodyAPIKey              string
	MindbodyMessageSignatureKey string
	MindbodyUsername            string
	MindbodyPassword            string
	MindbodySite                string
	MindbodyLocationID          int

	Debug bool
	Proxy bool
	Port  string
	Env   string
}

// GetConfig loads the environment variables into Config. Uses Config Vars on
// Heroku or .env file locally
func (config *Config) GetConfig() {
	// Check for "ENV" flag on Heroku
	if os.Getenv("ENV") != "staging" && os.Getenv("ENV") != "production" {
		// Load local env file
		if err := godotenv.Load(); err != nil {
			log.Fatalf("Error loading .env file: %s", err)
		}
	}

	config.BrivoUsername = getEnvStrings("brivo_username", "")
	config.BrivoPassword = getEnvStrings("brivo_password", "")
	config.BrivoClientID = getEnvStrings("brivo_client_id", "")
	config.BrivoClientSecret = getEnvStrings("brivo_client_secret", "")
	config.BrivoAPIKey = getEnvStrings("brivo_api_key", "")
	config.BrivoFacilityCode, _ = strconv.Atoi(getEnvStrings("brivo_facility_code", "0"))
	config.BrivoSiteID, _ = strconv.Atoi(getEnvStrings("brivo_site_id", "0"))
	config.BrivoMemberGroupID, _ = strconv.Atoi(getEnvStrings("brivo_member_group_id", "0"))
	config.BrivoBarcodeFieldID, _ = strconv.Atoi(getEnvStrings("brivo_barcode_field_id", "0"))
	config.BrivoUserTypeFieldID, _ = strconv.Atoi(getEnvStrings("brivo_user_type_field_id", "0"))
	config.BrivoRateLimit, _ = strconv.Atoi(getEnvStrings("brivo_rate_limit", "20"))

	config.MindbodyAPIKey = getEnvStrings("mindbody_api_key", "")
	config.MindbodyMessageSignatureKey = getEnvStrings("mindbody_message_signature_key", "")
	config.MindbodyUsername = getEnvStrings("mindbody_username", "")
	config.MindbodyPassword = getEnvStrings("mindbody_password", "")
	config.MindbodySite = getEnvStrings("mindbody_site", "-99")
	config.MindbodyLocationID, _ = strconv.Atoi(getEnvStrings("mindbody_location_id", "1"))

	config.Debug, _ = strconv.ParseBool(getEnvStrings("DEBUG", "true"))
	config.Proxy, _ = strconv.ParseBool(getEnvStrings("PROXY", "false"))
	config.Port = getEnvStrings("PORT", "")
	config.Env = getEnvStrings("ENV", "development")

}

// Base64Encoded credentials for Authorization header
func (config *Config) buildClientCredentials() {
	config.BrivoClientCredentials = base64.StdEncoding.EncodeToString([]byte(config.BrivoClientID + ":" + config.BrivoClientSecret))
}

// Helper function to check for environment variable strings
func getEnvStrings(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
