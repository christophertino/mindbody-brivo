/**
 * Sync Init
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

	sync "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
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

	sync.Authenticate(&config)
}
