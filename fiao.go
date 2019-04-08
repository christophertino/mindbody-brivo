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

	// Fetch MindBody token
	go func() {
		if err := auth.MindBodyToken.GetMindBodyToken(*config); err != nil {
			errCh <- err
		} else {
			doneCh <- true
		}
	}()

	//Fetch Brivo token
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

	// fmt.Printf("AUTH Model: %+v\n", auth)

	if config.ProgramArgs == "provision" {
		// Build Brivo client list from scratch (first-run)
		syncUsers(*config, auth)
	} else {
		// @TODO: Implement MindBody client webhook and poll for changes
	}
}

// Provision Brivo client list from MindBody
func syncUsers(config models.Config, auth models.Auth) {
	doneCh := make(chan bool)
	errCh := make(chan error)

	// Get all MindBody clients
	go func() {
		if err := mb.GetClients(config, auth.MindBodyToken.AccessToken); err != nil {
			errCh <- err
		} else {
			doneCh <- true
		}
	}()

	// Get existing Brivo clients
	go func() {
		if err := brivo.ListUsers(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
			errCh <- err
		} else {
			doneCh <- true
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			log.Fatalln("fiao.syncUsers: User fetch failed:\n", err)
		case <-doneCh:
			log.Println("fiao.syncUsers: User fetch success!")
		}
	}

	// fmt.Printf("MindBody Model: %+v\n Brivo Model: %+v\n", mb, brivo)

	// Map existing user data from MindBody to Brivo
	brivo.BuildBrivoUsers(mb, config, auth)
}
