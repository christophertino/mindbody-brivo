/**
 * MINDBODY Webhook Init
 *
 * This application listens for webhook events from MINDBODY
 * and makes corresponding changes in Brivo
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package main

import (
	"github.com/christophertino/mindbody-brivo/models"
	"github.com/christophertino/mindbody-brivo/server"
)

func main() {
	var config models.Config
	config.GetConfig()

	// fmt.Printf("Config Model: %+v\n", config)

	// Initialize API routes and listen for MindBody webhook events
	server.Init(&config)
}
