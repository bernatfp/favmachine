package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"github.com/mrjones/oauth"
	"log"
	"sync"
	"time"
)

//Tweet structure
type Tweet struct {
	Id   string `json:"id_str"`
	Text string
	User UserTwitter `json:"user"`
	Retweet   RetweetData `json:"retweeted_status"`
}

type UserTwitter struct {
	Id string `json:"id_str"`
}

type RetweetData struct {
	Id string `json:"id_str"`
}

//OAuth global data
var atoken *oauth.AccessToken

//Config file flag
var configPath = flag.String("c", "./config.json", "Indicates the path where the config file is located. Otherwise, an attempt to load a config.json file in the current directory will be made.")

//Main function
func main() {

	//Parse flag and load configuration
	flag.Parse()
	config := loadConfig()

	//Register OAuth client and access token
	client := oauth.NewConsumer(config.APIKey, config.APISecret, oauth.ServiceProvider{})
	atoken = &oauth.AccessToken{config.AccessToken, config.AccessTokenSecret, map[string]string{}}

	//Call Streaming API
	params := map[string]string{"track": config.Words}
	resp, err := client.Post("https://stream.twitter.com/1.1/statuses/filter.json", params, atoken)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	//Open a channel to count favs sent and print them
	statsch := make(chan int, 100)
	stopHours := make(chan int)
	go countfavs(statsch, stopHours)

	//Create error reporting channel
	errch := make(chan int)

	//Flag that indicates if favorites can be sent at a certain moment
	var canFav bool = true

	//Multiple goroutines can trigger an error but we just need to check once
	once := new(sync.Once)

	//Create retry channel, used to trigger periodic connection checks
	retry := make(chan bool)

	//Retry connection check periods
	minutes := []time.Duration{5, 10, 15, 30, 60}
	minIndex := 0

	//Read from tweets stream
	r := bufio.NewReader(resp.Body)
	var line []byte

	//Process tweets forever
	for {
		//Read one tweet
		line, err = r.ReadBytes('\n')
		if err != nil {
			log.Println("Error reading buffer: ", err)
			return //Replaced continue bc if stream is lost it keeps attempting to read bytes FOREVER
		}

		//Empty line
		if bytes.Equal(line, []byte{13, 10}) {
			continue
		}

		//Load data from tweet
		tweet := &Tweet{}
		err = json.Unmarshal(line, tweet)
		if err != nil {
			log.Println("Error decoding JSON: ", err)
			continue
		}

		//Watch channels
		select {
			//Check if we've received an error from a goroutine
			//Triggered by goroutines that are blocked access by Twitter when creating a fav or when an unknown error happens
			case code := <-errch:
				//This must be called only once even if multiple goroutines trigger an error
				once.Do(func() {
					//Account suspended
					if code == 64 {
						//Wait enough to let other goroutines print their stuff about this issue
						//The goroutine reporting the error is responsible for terminating execution
						time.Sleep(5 * time.Second)
					}
					
					//Check connection
					check(tweet, client, &canFav, retry)		
					
				})

			//Retry connection check
			case <-retry:
				canFav, minIndex = retryCheck(tweet, client, canFav, retry, minutes, minIndex)
				//When access to the API is restored errch can be triggered again, hence the need of a new Once instance
				if canFav {
					once = new(sync.Once)
					//Stats are restarted after recovering access
					go countfavs(statsch, stopHours)
				}

			//Stop execution for 24h to prevent being suspended
			case hours := <-stopHours:
				canFav = false
				log.Println("Can't continue. Going to wait for", hours, "hours.")
				time.AfterFunc(time.Duration(hours) * time.Hour, func(){
					canFav = true
					go countfavs(statsch, stopHours)
				})

			//Nothing to check, keep going
			default:
				break
		}

		//Process tweet
		go tweet.handler(client, statsch, errch, canFav)

	}
}
