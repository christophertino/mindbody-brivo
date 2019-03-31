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
	"os"

	"github.com/christophertino/fiao-sync"
	"github.com/christophertino/fiao-sync/models"
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

	// fmt.Printf("%+v", config)

	// Check for command-line arguments
	if len(os.Args) > 1 {
		config.ProgramArgs = os.Args[1]
	}

	fiao.Authenticate(&config)
}
