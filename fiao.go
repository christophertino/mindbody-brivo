/**
 * Application Logic
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package fiaoapi

import (
	"fmt"

	"github.com/christophertino/fiao_api/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/vacoj/Mindbody-API-Golang/siteservice"
	"github.com/vacoj/wsdl2go/soap"
)

// ConfigJSON : Settings imported from conf.json
type ConfigJSON struct {
	BrivoClientID     string `json:"brivo_client_id"`
	BrivoClientSecret string `json:"brivo_client_secret"`
	BrivoAPIKey       string `json:"brivo_api_key"`

	MindbodySourceName string `json:"mindbody_source_name"`
	MindbodySourcePass string `json:"mindbody_source_pass"`
	MindbodySite       int    `json:"mindbody_site"`
}

var (
	// MindBody : Data model for MindBody
	MindBody *models.MindBody
	Brivo    *models.Brivo
)

func (cj *ConfigJSON) buildModels() (mb *models.MindBody, b *models.Brivo, err error) {
	mb = &models.MindBody{
		SourceName: cj.MindbodySourceName,
		SourcePass: cj.MindbodySourcePass,
		Site:       cj.MindbodySite,
	}
	b = &models.Brivo{
		ClientID:     cj.BrivoClientID,
		ClientSecret: cj.BrivoClientSecret,
		APIKey:       cj.BrivoAPIKey,
	}
	return mb, b, nil
}

// Authenticate mindbody api
func Authenticate(cj *ConfigJSON) {
	var err error
	MindBody, Brivo, err = cj.buildModels()

	cli := soap.Client{
		URL:       "https://clients.mindbodyonline.com/api/0_5_1/SiteService.asmx",
		Namespace: siteservice.Namespace,
	}
	conn := siteservice.NewSite_x0020_ServiceSoap(&cli)
	sourceCreds := &siteservice.SourceCredentials{
		SourceName: MindBody.SourceName,
		Password:   MindBody.SourcePass,
		SiteIDs: &siteservice.ArrayOfInt{
			Int: []int{MindBody.Site},
		},
	}

	req := &siteservice.GetSitesRequest{
		SourceCredentials: sourceCreds,
	}

	reply, err := conn.GetSites(&siteservice.GetSites{Request: req})
	if err != nil {
		fmt.Println(err)
	}
	spew.Dump(reply.GetSitesResult.Sites.Site[0])
}
