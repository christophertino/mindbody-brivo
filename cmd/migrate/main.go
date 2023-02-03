// Client Migration Init

// Use this application to provision a new Brivo setup by
// bulk-migrating all active MINDBODY users.

package main

import (
	"github.com/christophertino/mindbody-brivo/migrate"
	"github.com/christophertino/mindbody-brivo/models"
)

func main() {
	var config models.Config
	config.GetConfig()

	// Sync all MINDBODY clients to Brivo
	migrate.GetAllUsers(&config)
}
