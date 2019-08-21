// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Configuration Data Model

package models

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"

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

	MindbodyAPIKey              string
	MindbodyMessageSignatureKey string
	MindbodyUsername            string
	MindbodyPassword            string
	MindbodySite                string

	Debug bool
	Port  string
}

// GetConfig : Load environment variables into Config. Uses Config Vars on
// Heroku or .env file locally
func (config *Config) GetConfig() {
	// Check for "ENV" flag on Heroku
	if os.Getenv("ENV") != "staging" && os.Getenv("ENV") != "production" {
		// Load local env file
		if err := godotenv.Load(); err != nil {
			log.Fatalln("Config.GetConfig: Error loading .env file")
		}
	}

	config.BrivoUsername = getEnvStrings("brivo_username", "")
	config.BrivoPassword = getEnvStrings("brivo_password", "")
	config.BrivoClientID = getEnvStrings("brivo_client_id", "")
	config.BrivoClientSecret = getEnvStrings("brivo_client_secret", "")
	config.BrivoAPIKey = getEnvStrings("brivo_api_key", "")
	config.BrivoMemberGroupID, _ = strconv.Atoi(getEnvStrings("brivo_member_group_id", "0")) // parse to int

	config.MindbodyAPIKey = getEnvStrings("mindbody_api_key", "")
	config.MindbodyMessageSignatureKey = getEnvStrings("mindbody_message_signature_key", "")
	config.MindbodyUsername = getEnvStrings("mindbody_username", "")
	config.MindbodyPassword = getEnvStrings("mindbody_password", "")
	config.MindbodySite = getEnvStrings("mindbody_site", "-99")

	config.Debug, _ = strconv.ParseBool(getEnvStrings("DEBUG", "true"))
	config.Port = getEnvStrings("PORT", "")
}

// BuildClientCredentials : Base64Encoded credentials for Authorization header
func (config *Config) BuildClientCredentials() {
	config.BrivoClientCredentials = base64.StdEncoding.EncodeToString([]byte(config.BrivoClientID + ":" + config.BrivoClientSecret))
}

// Helper function to check for environment variable strings
func getEnvStrings(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
