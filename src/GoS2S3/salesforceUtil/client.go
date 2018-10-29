// TODO:
//	- All the function call returning a boolean can be refactored to return also an error instead of exiting?
//	- It is probably a good idea to include a "contructor" method  on the SF_connection type
//		that accepts a map of fields and sets the corresponding fields so that we don't have to expose them
package salesforceUtil

import (
	"GoS2S3/SalesforceWSDL"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func (connection *SF_connection) GetAuthenticationToken() string {
	var authenticationRequest string
	authenticationRequest += connection.TargetURI + "/services/oauth2/token?grant_type=password"
	authenticationRequest += "&client_id=" + connection.ClientId
	authenticationRequest += "&client_secret=" + connection.ClientSecret
	authenticationRequest += "&username=" + connection.Username
	authenticationRequest += "&password=" + connection.Password + connection.SecurityToken

	response, requestError := http.PostForm(authenticationRequest, nil)
	if requestError != nil {
		response.Body.Close()
		log.Println("ERROR: Authentication failure!")
		if connection.Debug {
			log.Println(requestError)
		} else {
			log.Println("Error: an error occurred during the execution, please run with -debug for more information")
		}

		os.Exit(1)
	}
	defer response.Body.Close()

	log.Println("Authentication SUCCESSFUL")

	body, _ := ioutil.ReadAll(response.Body)
	//fmt.Printf("%s\n", string(body))
	//fmt.Println(string(body))
	var jsonBody map[string]interface{}
	if err := json.Unmarshal(body, &jsonBody); err != nil {
		panic(err)
	}

	connection.AuthenticationToken.Access_token = jsonBody["access_token"].(string)
	connection.AuthenticationToken.Instance_url = jsonBody["instance_url"].(string)
	connection.AuthenticationToken.Id = jsonBody["id"].(string)
	connection.AuthenticationToken.Token_type = jsonBody["token_type"].(string)
	connection.AuthenticationToken.Issued_at = jsonBody["issued_at"].(string)
	connection.AuthenticationToken.Signature = jsonBody["signature"].(string)

	if connection.Debug {
		log.Printf("Access token: %s", connection.AuthenticationToken.Access_token)
	}

	return connection.AuthenticationToken.Access_token
}

// TODO handle request error
func (connection *SF_connection) RequestPageOAuth(targetUrl string) string {
	client := &http.Client{}
	request, requestError := http.NewRequest("GET", targetUrl, nil)
	if requestError != nil {
		return ""
	}
	cookieOrg := http.Cookie{Name: "oid", Value: connection.OrganizationId}
	cookieSid := http.Cookie{Name: "sid", Value: connection.SoapLogin.SessionId}

	request.AddCookie(&cookieOrg)
	request.AddCookie(&cookieSid)

	response, responseError := client.Do(request)
	if responseError != nil {
		log.Println(responseError)
	}
	defer response.Body.Close()

	body, readError := ioutil.ReadAll(response.Body)
	if readError != nil {
		if connection.Debug {
			log.Println(readError)
		} else {
			log.Println("Error: an error occurred during the execution, please run with -debug for more information")
		}
	}

	return string(body)
}

type SF_connection struct {
	TargetURI           string
	Username            string
	Password            string
	SecurityToken       string
	ClientSecret        string
	ClientId            string
	AuthenticationToken AccessToken
	SoapEndpoint        string
	SoapLogin           SalesforceWSDL.LoginResult
	SessionId           string
	OrganizationId      string
	ConnectionCookies   map[string]interface{}
	Debug               bool
}

type AccessToken struct {
	Access_token string
	Instance_url string
	Id           string
	Token_type   string
	Issued_at    string
	Signature    string
}
