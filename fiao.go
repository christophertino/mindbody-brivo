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

	models "github.com/christophertino/fiao_api/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/vacoj/Mindbody-API-Golang/siteservice"
	"github.com/vacoj/wsdl2go/soap"
)

var mindBody = models.MindBody{
	SourceName: "",
	SourcePass: "",
	Site:       -99,
}

type ConfigJSON struct {
	BrivoClientID     string `json:"brivo_client_id"`
	BrivoClientSecret string `json:"brivo_client_secret"`
	BrivoAPIKey       string `json:"brivo_api_key"`

	MindbodySourceName string `json:"mindbody_source_name"`
	MindbodySourcePass string `json:"mindbody_source_pass"`
	MindbodySite       string `json:"mindbody_site"`
}

// Authenticate mindbody api
func Authenticate() {
	cli := soap.Client{
		URL:       "https://api.mindbodyonline.com/0_5/SiteService.asmx",
		Namespace: siteservice.Namespace,
	}
	conn := siteservice.NewSite_x0020_ServiceSoap(&cli)
	sourceCreds := &siteservice.SourceCredentials{
		SourceName: mindBody.SourceName,
		Password:   mindBody.SourcePass,
		SiteIDs: &siteservice.ArrayOfInt{
			Int: []int{mindBody.Site},
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
