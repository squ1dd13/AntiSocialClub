package social_club

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type UserAccount struct {
	RockstarId   string `xml:"RockstarId"`
	Age          string `xml:"Age"`
	CountryCode  string `xml:"CountryCode"`
	Email        string `xml:"Email"`
	LanguageCode string `xml:"LanguageCode"`
	Nickname     string `xml:"Nickname"`
}

type loginResponse struct {
	XMLName             xml.Name    `xml:"Response"`
	Xsd                 string      `xml:"xsd,attr"`
	Xsi                 string      `xml:"xsi,attr"`
	Ms                  string      `xml:"ms,attr"`
	Xmlns               string      `xml:"xmlns,attr"`
	Status              string      `xml:"Status"`
	Ticket              string      `xml:"Ticket"`
	PosixTime           string      `xml:"PosixTime"`
	SecsUntilExpiration string      `xml:"SecsUntilExpiration"`
	PlayerAccountId     string      `xml:"PlayerAccountId"`
	PublicIp            string      `xml:"PublicIp"`
	SessionId           string      `xml:"SessionId"`
	SessionKey          string      `xml:"SessionKey"`
	SessionTicket       string      `xml:"SessionTicket"`
	MFAEnabled          string      `xml:"MFAEnabled"`
	RockstarAccount     UserAccount `xml:"RockstarAccount"`
	Error               *struct {
		Code   string `xml:"Code,attr"`
		CodeEx string `xml:"CodeEx,attr"`
	} `xml:"Error"`
	Privileges string `xml:"Privileges"`
}

func (response loginResponse) getError() error {
	if response.Error != nil {
		return fmt.Errorf("%s: %s", response.Error.Code, response.Error.CodeEx)
	}

	return nil
}

// Details about a logged-in user.
type Session struct {
	initialLoginResponse loginResponse
	cachedExpirationTime int64
}

// Store the session in a file for loading later.
func (session *Session) Save() error {
	configDir, err := os.UserConfigDir()

	if err != nil {
		return err
	}

	socialClubDir := filepath.Join(configDir, "SocialClub")

	err = os.MkdirAll(socialClubDir, 0777)

	if err != nil {
		return err
	}

	sessionPath := filepath.Join(socialClubDir, "session")
	destinationFile, err := os.Create(sessionPath)

	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(destinationFile)
	err = encoder.Encode(session.initialLoginResponse)

	if err != nil {
		return err
	}

	return destinationFile.Close()
}

func LoadSession() (*Session, error) {
	configDir, err := os.UserConfigDir()

	if err != nil {
		return nil, err
	}

	sessionPath := filepath.Join(configDir, "SocialClub", "session")
	sessionFile, err := os.Open(sessionPath)

	if err != nil {
		return nil, err
	}

	session := &Session{}

	decoder := gob.NewDecoder(sessionFile)
	err = decoder.Decode(&session.initialLoginResponse)

	if err != nil {
		return nil, err
	}

	return session, nil
}

func (session *Session) User() UserAccount {
	return session.initialLoginResponse.RockstarAccount
}

func (session *Session) ticket() string {
	return session.initialLoginResponse.Ticket
}

func (session *Session) CreateUrl(differentiator string) string {
	const format = "http://prod.ros.rockstargames.com/cloud/11/cloudservices/members/sc/%s%s?%s"

	query := url.Values{
		"ticket": {session.ticket()},
	}

	return fmt.Sprintf(format, session.User().RockstarId, differentiator, query.Encode())
}

func (session *Session) Fetch(differentiator string) ([]byte, error) {
	response, err := http.Get(session.CreateUrl(differentiator))

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(response.Body)
}

func (session *Session) ExpirationTime() int64 {
	if session.cachedExpirationTime != 0 {
		return session.cachedExpirationTime
	}

	startTime, err := strconv.ParseInt(session.initialLoginResponse.PosixTime, 10, 64)

	// The values in the login response should not be invalid, so we panic if they are.
	if err != nil {
		panic(err)
	}

	sessionLength, err := strconv.ParseInt(session.initialLoginResponse.SecsUntilExpiration, 10, 64)

	if err != nil {
		panic(err)
	}

	session.cachedExpirationTime = startTime + sessionLength
	return session.cachedExpirationTime
}

func (session *Session) Expired() bool {
	return time.Now().Unix() >= session.ExpirationTime()
}

func LogIn(email string, password string) (*Session, error) {
	const salt = "CwJK/SThnLQ+4fz/w8BBT9s3Ambp9GuRzYZdXGVRNlf4zI5yrRTjt5rdq9QUybXT65Gz7lst+ha0sGPZMQDyCI8="
	key := newKeySalt(salt)

	// Build the login query and encrypt it.
	query := url.Values{
		// Always spoof an iOS device, since we know what data they send.
		"platformName": {"ios"},
		"email":        {email},
		"password":     {password},
	}

	encryptedBody := bytes.NewReader(encrypt(key, query.Encode()))

	const loginUrl = "http://prod.ros.rockstargames.com/gtasa/11/gameservices/auth.asmx/CreateTicketSc"
	request, err := http.NewRequest(http.MethodPost, loginUrl, encryptedBody)

	if err != nil {
		return nil, err
	}

	// Refuse redirects. The server tries to turn our POST request into a GET request for an error page,
	//  but everything works fine if we just ignore the redirect and continue with the POST.
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	// Add an encrypted user agent field. This is what tells the server that the request body is encrypted too.
	request.Header.Add("User-Agent", createUserAgent("gtasa", "ios", "11"))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	// Send the request.
	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	// The response will be encrypted, so we have to decrypt it.
	responseXml, err := decrypt(key, responseBytes)
	fmt.Println(responseXml)

	if err != nil {
		return nil, err
	}

	theLoginResponse := loginResponse{}
	err = xml.Unmarshal([]byte(responseXml), &theLoginResponse)

	if err != nil {
		return nil, err
	}

	// Check for login errors.
	err = theLoginResponse.getError()

	if err != nil {
		return nil, err
	}

	return &Session{initialLoginResponse: theLoginResponse}, nil
}
