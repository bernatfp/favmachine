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

	log.Println("Welcome to favmachine \\o/")

	//Parse flag and load configuration
	flag.Parse()
	config := loadConfig()

	log.Println("Configuration loaded")

	var waitTime time.Duration

	for {
		log.Println("Opening a new tweet stream")
		waitTime = master(config)
		if waitTime == -1 {
			log.Fatal("Error reading stream, can't continue...")
		}
		log.Println("Closing stream. We're going to wait for", waitTime)
		time.Sleep(waitTime)
	}
}

func getTweet(r *bufio.Reader) (*Tweet, error) {
	//Read one tweet
	line, err := r.ReadBytes('\n')
	if err != nil {
		log.Println("Error reading buffer: ", err)
		return nil, err //Replaced continue bc if stream is lost it keeps attempting to read bytes FOREVER
	}

	//Empty line
	if bytes.Equal(line, []byte{13, 10}) {
		return nil, nil //just discard it, sometimes the stream sends rubbish
	}

	//Load data from tweet
	tweet := &Tweet{}
	err = json.Unmarshal(line, tweet)
	if err != nil {
		log.Println("Error decoding JSON: ", err)
		return nil, nil //just discard it, sometimes the stream sends rubbish
		//Idea: if it happens many times in a row, we can stop execution for a minute and open a new stream (by returning to main)
	}

	return tweet, nil

}

//Main function
func master(config *Config) time.Duration {

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
	ns := []time.Duration{5, 10, 15, 30, 60} //By default the order of magnitude for Duration is nanoseconds
	periodIndex := 0

	//Read from tweets stream
	r := bufio.NewReader(resp.Body)

	//One test fav to check it works
	tweet, err := getTweet(r)
	if err != nil {
		return -1
	}
	if tweet == nil {
		return time.Duration(1) * time.Minute
	}
	if allowed, _ := testConn(tweet, client); allowed == false {
		return time.Duration(1) * time.Hour
	}

	//Process tweets forever
	for {
		//Read one tweet
		/*line, err = r.ReadBytes('\n')
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
		}*/

		tweet, err = getTweet(r)
		if err != nil {
			return -1
		}

		//Watch channels
		select {
			//Check if we've received an error from a goroutine
			//Triggered by goroutines that are blocked access by Twitter when creating a fav or when an unknown error happens
			case code := <-errch:
				//This must be called only once even if multiple goroutines trigger an error
				once.Do(func() {
					//Account suspended, we'll terminate execution in a few seconds
					if code == 64 {
						//Wait enough to let other goroutines print their stuff about this issue
						//The goroutine reporting the error is responsible for terminating execution
						time.Sleep(5 * time.Second)
					}
					
					//Check connection
					check(tweet, client, &canFav, retry)		
					//If connection is working again, execution will continue normally
					//If we are still not allowed to do favs, <-retry will be triggered in one minute
				})

			//Retry connection check
			case <-retry:
				canFav, periodIndex = retryCheck(tweet, client, canFav, retry, ns, periodIndex)
				//When access to the API is restored errch can be triggered again, hence the need of a new Once instance
				if canFav {
					once = new(sync.Once)
					//Stats are restarted after recovering access
					go countfavs(statsch, stopHours)
				} else {
					//Once the period is 60 we consider it's not worth keeping the stream open (we've already been trying for an hour)
					if ns[periodIndex] == time.Duration(60){
						return 	ns[periodIndex] * time.Minute //1 hour
					}		
				}
				

			//Stop execution for 24h to prevent being suspended
			case hours := <-stopHours:
				canFav = false
				log.Println("Can't continue. Going to wait for", hours, "hours.")
				
				return time.Duration(hours) * time.Hour

				/*time.AfterFunc(time.Duration(hours) * time.Hour, func(){
					canFav = true
					go countfavs(statsch, stopHours)
				})*/

			//Nothing to check, keep going
			default:
				break
		}

		//Process tweet
		go tweet.handler(client, statsch, errch, canFav)

	}
}
