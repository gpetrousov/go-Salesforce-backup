// TODO:
//		- refactor the configuration file loader to return a map of strings so that we can add
//			an initializer method to the Salesforce client
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	// "encoding/json"
	"GoS2S3/SalesforceWSDL"
	"GoS2S3/salesforceUtil"

	"github.com/aws/aws-sdk-go/aws/session"
)

var debug bool
var timestampEpoch time.Time
var todayEpoch int64

func main() {

	arguments := os.Args

	for _, element := range arguments {
		if element == "-help" {
			// Put here some help explanation
			fmt.Println("This is a help page!")
			os.Exit(3)
		}
	}

	timestampEpoch = time.Now()
	todayEpoch = timestampEpoch.Unix() - (timestampEpoch.Unix() % 86400)

	var SF_Soap SalesforceWSDL.Soap
	var SF_BasicAuth SalesforceWSDL.BasicAuth

	var activeSalesforceConnection salesforceUtil.SF_connection

	// --------------------- INITIALIZATION ---------------------

	activeSalesforceConnection.ConnectionCookies = make(map[string]interface{}, 0)

	// Command line parsing
	flag.BoolVar(&debug, "debug", false, "Activate debug mode")
	flag.Parse()

	if debug {
		log.Println("")
		log.Println("DEBUG MODE ON!")
		log.Println("")
		activeSalesforceConnection.Debug = debug
	} else {
		log.Println("we are in NORMAL mode")
	}

	for _, element := range arguments {
		if element == "-config" {
			var configuration Configuration

			configuration.LoadConfigFrom("application-config.json")
		}
	}

	// refactor methods to use pointer to struct
	loadSalesforceConfigurationFromFile(&configuration.Salesforce, &activeSalesforceConnection)
	loadAWSConfigurationFromFile(&configuration.Amazon)
	// --------------------- END INITIALIZATION ---------------------

	activeSalesforceConnection.GetAuthenticationToken()

	SF_Soap, SF_BasicAuth = activeSalesforceConnection.AuthenticateThroughSOAP()

	downloadPage := activeSalesforceConnection.RequestPageOAuth(activeSalesforceConnection.AuthenticationToken.Instance_url + "/ui/setup/export/DataExportPage/d?setupid=DataManagementExport&retURL=%2Fui%2Fsetup%2FSetup%3Fsetupid%3DDataManagementq")

	log.Println("Extracting download links... ")
	var downloadLinks = extractDownloadLinks(downloadPage)
	downloadLinks = prependBasicUrl(downloadLinks, activeSalesforceConnection.AuthenticationToken.Instance_url)
	if activeSalesforceConnection.Debug {
		log.Println("URls found in page:")
		for _, element := range downloadLinks {
			log.Printf("\t- %s\n", element)
		}
	} else {
		log.Printf("%d links found in the page", len(downloadLinks))
	}

	amazonSession := session.Must(session.NewSession())
	log.Println("Amazon session created")

	if debug {
		downloadLinks = make([]string, 0)
		downloadLinks = append(downloadLinks, "https://raw.githubusercontent.com/nikotrone/GoS2S3/master/README.md?fileName=test.foo")
	}

	if len(downloadLinks) == 0 {
		log.Println("")
		log.Println("Nothing to do")
		log.Println("")
		return
	}

	creationError := os.Mkdir("tmp", 0777)
	if (creationError != nil) && (!os.IsExist(creationError)) {
		log.Printf("Error creating destination folder: %v", creationError)
		panic(1)
	}

	for _, value := range downloadLinks {
		fileName, transferError := transferFile(value, activeSalesforceConnection.ConnectionCookies, amazonSession, configuration)
		if transferError != nil {
			log.Printf("Error while transfering file %s: %v", fileName, transferError)
		}
	}

	// to avoid the imported and not used error
	if false {
		log.Println(SF_Soap)
		log.Println(SF_BasicAuth)
	}

}

func prependBasicUrl(links []string, basicUrl string) []string {
	if len(links) <= 0 {
		return make([]string, 0)
	}

	for key, _ := range links {
		links[key] = basicUrl + links[key]
	}
	return links
}

func transferFile(downloadLink string, salesforceConnectionCookies map[string]interface{}, amazonSession *session.Session, applicationConfiguration Configuration) (fileName string, transferError error) {
	log.Printf("Downloading file: %s", downloadLink)
	fileName, downloadError := downloadFileFromUrl(downloadLink, salesforceConnectionCookies)
	if downloadError != nil {
		log.Println("Error downloading the file from target location")
		return fileName, downloadError
	} else {
		log.Println("Download Successful!!")
	}

	log.Println("Uploading file to S3 bucket...")
	_, uploadError := uploadFileToS3(amazonSession, applicationConfiguration.Amazon, fileName)
	if uploadError != nil {
		log.Println("Error uploading the file to target location")
		return fileName, uploadError
	} else {
		log.Println("Upload Successful!!")
	}

	return fileName, nil
}
