// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// MINDBODY Webhook Init
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

	// fmt.Printf("Config Model: %+v\n", config)

	// Initialize API routes and listen for MINDBODY webhook events
	server.Init(&config)
}
