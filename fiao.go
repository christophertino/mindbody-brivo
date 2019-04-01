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
			log.Fatalln("fiao.Authenticate: Token fetch failed:\n", err)
		case <-doneCh:
			log.Println("fiao.Authenticate: Token fetch success!")
		}
	}

	// fmt.Printf("%+v", auth)

	if config.ProgramArgs == "provision" {
		// TODO: Use this to build Brivo client list from scratch (first-run)
		syncUsers(config, &auth)
	} else {
		// TODO: Implement MindBody client webhook and poll for changes
	}
}

func syncUsers(config *models.Config, auth *models.Auth) {
	// Get all MindBody clients
	mb.GetClients(config, auth.MindBodyToken.AccessToken)
	// Get existing Brivo clients
	if err := brivo.ListUsers(config, auth.BrivoToken.AccessToken); err != nil {
		log.Fatalln("fiao.syncUsers: Failed fetching Brivo users \n", err)
	}

	// Map existing user data from MindBody to Brivo
	brivo.BuildBrivoUsers(&mb, config, auth)

	// POST new users to Brivo

	// Map user groups
}
