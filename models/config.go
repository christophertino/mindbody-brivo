/**
 * Configuration Data Model
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

import (
	"encoding/base64"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config : Settings imported from .env
type Config struct {
	BrivoUsername          string
	BrivoPassword          string
	BrivoClientID          string
	BrivoClientSecret      string
	BrivoAPIKey            string
	BrivoMemberGroupID     int
	BrivoClientCredentials string

	MindbodyAPIKey   string
	MindbodyUsername string
	MindbodyPassword string
	MindbodySite     string
}

// GetConfig : Load environment variables into Config
func (config *Config) GetConfig() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("main: Error loading .env file")
	}

	config.BrivoUsername = getEnvStrings("brivo_username", "")
	config.BrivoPassword = getEnvStrings("brivo_password", "")
	config.BrivoClientID = getEnvStrings("brivo_client_id", "")
	config.BrivoClientSecret = getEnvStrings("brivo_client_secret", "")
	config.BrivoAPIKey = getEnvStrings("brivo_api_key", "")
	config.BrivoMemberGroupID = getEnvStrings("brivo_member_group_id", "0")

	config.MindbodyAPIKey = getEnvStrings("mindbody_api_key", "")
	config.MindbodyUsername = getEnvStrings("mindbody_username", "")
	config.MindbodyPassword = getEnvStrings("mindbody_password", "")
	config.MindbodySite = getEnvStrings("mindbody_site", "-99")
}

// BuildClientCredentials : Base64Encoded credentials for Authorization header
func (config *Config) BuildClientCredentials() {
	config.BrivoClientCredentials = base64.StdEncoding.EncodeToString([]byte(config.BrivoClientID + ":" + config.BrivoClientSecret))
}

// Helper function to check for environment variable strings
func getEnvStrings(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
