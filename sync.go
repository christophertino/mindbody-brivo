/**
 * Migrate all MINDBODY clients to Brivo as new users
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package sync

import (
	"fmt"
	"log"
	"sync"

	"github.com/christophertino/mindbody-brivo/models"
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
			log.Fatalln("sync.Authenticate: Token fetch failed:\n", err)
		case <-doneCh:
			fmt.Println("sync.Authenticate: Token fetch success!")
		}
	}

	// fmt.Printf("AUTH Model: %+v\n", auth)

	// Build Brivo client list from scratch (first-run)
	syncUsers(*config, auth)
}

// Provision Brivo client list from MindBody
func syncUsers(config models.Config, auth models.Auth) {
	var wg sync.WaitGroup
	var errCh = make(chan error)

	// Get all MindBody clients
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := mb.GetClients(config, auth.MindBodyToken.AccessToken); err != nil {
			errCh <- err
		}
	}()

	// Get existing Brivo clients
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := brivo.ListUsers(config.BrivoAPIKey, auth.BrivoToken.AccessToken); err != nil {
			errCh <- err
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			log.Fatalln("sync.syncUsers: User fetch failed:\n", err)
		default:
			fmt.Println("sync.syncUsers: User fetch success!")
		}
	}

	wg.Wait()

	// fmt.Printf("MindBody Model: %+v\n Brivo Model: %+v\n", mb, brivo)

	// Map existing user data from MindBody to Brivo
	brivo.BuildBrivoUsers(mb, config, auth)
}
