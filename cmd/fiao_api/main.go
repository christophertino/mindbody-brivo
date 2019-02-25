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
	"fmt"
	"io/ioutil"
	"log"

	fiaoapi "github.com/christophertino/fiao_api"
)

func main() {
	settings, err := ioutil.ReadFile("conf/conf.json")
	if err != nil {
		log.Fatal("Failed reading from conf", err)
	}

	var conf fiaoapi.ConfigJSON

	err = json.Unmarshal(settings, &conf)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
	}

	fmt.Printf("%+v", conf)

	fiaoapi.Authenticate(&conf)
}
