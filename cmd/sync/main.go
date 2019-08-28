// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Membership Sync Init
// Use this application to provision a new Brivo setup by
// bulk-migrating all active MINDBODY users.

package main

import (
	"github.com/christophertino/mindbody-brivo/models"
	"github.com/christophertino/mindbody-brivo/sync"
)

func main() {
	var config models.Config
	config.GetConfig()

	// Sync all MINDBODY clients to Brivo
	sync.GetAllUsers(&config)
}
