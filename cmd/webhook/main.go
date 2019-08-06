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
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/christophertino/mindbody-brivo/models"
	"github.com/christophertino/mindbody-brivo/server"
)

func main() {
	settings, err := ioutil.ReadFile("conf/conf.json")
	if err != nil {
		log.Fatal("main: Failed reading from conf", err)
	}

	var config models.Config

	err = json.Unmarshal(settings, &config)
	if err != nil {
		log.Fatal("main: Unmarshal error:", err)
	}

	// fmt.Printf("Config Model: %+v\n", config)

	// Initialize API routes and listen for MindBody webhook events
	server.Init(&config)
}
