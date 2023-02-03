// MINDBODY Webhook Init
//
// This application listens for webhook events from MINDBODY
// and makes corresponding changes in Brivo

package main

import (
	"github.com/christophertino/mindbody-brivo/models"
	"github.com/christophertino/mindbody-brivo/server"
)

func main() {
	var config models.Config
	config.GetConfig()

	// Initialize server API routes and listen for MINDBODY webhook events
	server.Launch(&config)
}
