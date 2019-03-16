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
	"log"

	"github.com/christophertino/fiao-sync/models"
)

var (
	auth  models.Auth
	mb    models.MindBody
	brivo models.Brivo
)

// Authenticate : Fetch access tokens for MindBody and Brivo
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
			log.Fatalln("fiao.Authenticate: Token fetch failed:", err)
		case <-doneCh:
			log.Println("fiao.Authenticate: Token fetch success!")
		}
	}

	// fmt.Printf("%+v", auth)

	syncUsers(config)
}

func syncUsers(config *models.Config) {
	// TODO: Replace with MindBody client webhook
	// Get all MindBody clients
	mb.GetClients(config, auth.MindBodyToken.AccessToken)
	// Get existing Brivo clients
	err := brivo.ListUsers(config, auth.BrivoToken.AccessToken)
	if err != nil {
		log.Fatalln("fiao.syncUsers: Failed fetching Brivo users", err)
	}

	// Map existing user data from MindBody to Brivo
	brivo.BuildBrivoUsers(&mb)

	// POST new users to Brivo

	// Map user groups
}
