/**
 * Application Init
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	fiaoapi "github.com/christophertino/fiao_api"
	"github.com/christophertino/fiao_api/models"
)

func main() {
	settings, err := ioutil.ReadFile("conf/conf.json")
	if err != nil {
		log.Fatal("Failed reading from conf", err)
	}

	var config models.Config

	err = json.Unmarshal(settings, &config)
	if err != nil {
		log.Fatal("Unmarshal error:", err)
	}

	// fmt.Printf("%+v", config)

	fiaoapi.Authenticate(&config)
}
