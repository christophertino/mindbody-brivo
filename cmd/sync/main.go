/**
 * Membership Sync Init
 *
 * Use this application to provision a new Brivo setup by
 * bulk-migrating all active MINDBODY users.
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package main

import (
	sync "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
)

func main() {
	var config models.Config
	config.GetConfig()

	// fmt.Printf("Config Model: %+v\n", config)

	sync.Authenticate(&config)
}
