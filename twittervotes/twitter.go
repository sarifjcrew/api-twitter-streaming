package main

import(
	"net"
	"time"
	"io"
	"github.com/garyburd/go-oauth/oauth"
	"gopkg.in/mgo.v2"
	"log"
	"sync"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"encoding/json"
)

var (
	conn net.Conn
	reader io.ReadCloser
	authClient *oauth.Client
	creds *oauth.Credentials
	authSetupOnce sync.Once
	httpClient *http.Client
	db *mgo.Session
)

type tweet struct {
	Text string
}

func dial(netw, addr string)(net.Conn, error) {
	if conn != nil {
		conn.Close()
		conn = nil
	}

	netc, err := net.DialTimeout(netw, addr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	conn = netc
	return netc, nil
}

func closeConn() {
	if conn != nil {
		conn.Close()
	}

	if reader != nil {
		reader.Close()
	}
}

func setupTwitterAuth() {
	type ts struct {
		ConsumerKey string
		ConsumerSecret string
		AccessToken string
		AccessSecret string
	}
	
	var twitterCred = ts{
		ConsumerKey : "9dfpB2dyG8upC3eT0swL3XRwj",
		ConsumerSecret : "pXTzHR5gPnblLha2qgUG4iXxHYrbWyS6Z1a0S4eucunYp4Jhye",
		AccessToken : "3116441227-X1qAK01xg6hbevi4YENN7MGxCtwBKf0gnw5oaYr",
		AccessSecret : "ZKqBXeKZyG84a2nHuYYYRnmFf5rleEA7MKio12VfRvRZA",
	}

	creds = &oauth.Credentials{
		Token: twitterCred.AccessToken,
		Secret: twitterCred.AccessSecret,
	}

	authClient = &oauth.Client{
		Credentials: oauth.Credentials{
			Token: twitterCred.ConsumerKey,
			Secret: twitterCred.ConsumerSecret,
		},
	}
}


func makeRequest(req *http.Request, params url.Values) (*http.Response, error) {
	authSetupOnce.Do(func(){
		setupTwitterAuth()
		httpClient = &http.Client {
			Transport: &http.Transport{
				Dial: dial,
			},
		}
	})
	
	formEnc := params.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))
	req.Header.Set("Authorization", authClient.AuthorizationHeader(creds, "POST", req.URL, params))
	return httpClient.Do(req)
}

func dialdb() error {
	var err error
	log.Println("dialing mongodb: localhost")
	db, err = mgo.Dial("localhost")
	return err
}

func closedb() {
	db.Close()
	log.Println("closed database connection")
}

func readFromTwitter(votes chan<- string) {
	options, err := loadOptions()
	if err != nil {
		log.Println("failed to load options:", err)
		return
	}

	u, err := url.Parse("https://stream.twitter.com/1.1/statuses/filter.json")
	_ = u
	if err != nil {
		log.Println("creating filter request failed:", err)
		return
	}

	query := make(url.Values)
	query.Set("track", strings.Join(options, ","))
	req, err := http.NewRequest("POST", u.String(), strings.NewReader(query.Encode()))
	if err != nil {
		log.Println("creating filter request failed:", err)
		return
	}

	resp, err := makeRequest(req, query)
	if err != nil {
		log.Println("making request failed:", err)
		return
	}

	reader := resp.Body
	decoder := json.NewDecoder(reader)
	var t tweet
	if err := decoder.Decode(&t); err != nil {
		log.Fatalln("error decode:", err)
	}

	for _,option := range options {
		if strings.Contains(strings.ToLower(t.Text), strings.ToLower(option)) {
			votes <- option
		}

	}
}

func startTwitterStream(stopchan <-chan struct{}, votes chan<- string) <-chan struct{} {
	stoppedchan := make(chan struct{}, 1)
	go func() {
		defer func() {
			stoppedchan <- struct{}{}
		}()
	

		for {
			select {
			case <-stopchan:
				log.Println("stopping twitter...")
				return
			default:
				log.Println("Querying Twitter...")
				readFromTwitter(votes)
				log.Println(" (waiting)")
				time.Sleep(10 * time.Second) //wait before reconnecting
			}
		}
	}()

	return stoppedchan
}