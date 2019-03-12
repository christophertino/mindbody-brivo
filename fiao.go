/**
 * Application Logic
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package fiao

import (
	"fmt"
	"log"

	"github.com/christophertino/fiao-sync/models"
)

var (
	auth  models.Auth
	mb    models.MindBody
	brivo models.Brivo
)

// Authenticate mindbody api
func Authenticate(config *models.Config) {
	doneCh := make(chan bool)
	errCh := make(chan error)

	go func() {
		if err := auth.MindBodyToken.GetMindBodyToken(config); err != nil {
			errCh <- err
		} else {
			doneCh <- true
		}
	}()
	go func() {
		if err := auth.BrivoToken.GetBrivoToken(config); err != nil {
			errCh <- err
		} else {
			doneCh <- true
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			log.Fatalln("Token fetch failed:", err)
		case <-doneCh:
			log.Println("Token fetch success!")
		}
	}

	fmt.Printf("%+v", auth)

	mb.GetClients(config, auth.MindBodyToken.AccessToken)
	brivo.ListUsers(config, auth.BrivoToken.AccessToken)

	fmt.Printf("%+v", brivo)

	// @TODO:
	// + Map existing user data from MindBody to Brivo
	// + Track matching user IDs using brivo.Data[i].CustomFields
	// + POST new users to Brivo from MindBody
	// + Map user groups
}
