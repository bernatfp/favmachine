package main

import (
	"github.com/mrjones/oauth"
	"log"
	"time"
)

//Send a test fav
func testConn(tweet *Tweet, client *oauth.Consumer) bool {
	params := map[string]string{"id": tweet.Id}
	resp, err := client.Post("https://api.twitter.com/1.1/favorites/create.json", params, atoken)
	if err != nil {
		//Parse err data into struct
		errData := new(ErrResponse)
		errData.loadErrData(err)
		//Check if the error is related to banned API access
		if errData.HTTPErr.Code == 429 || errData.APIErr[0].Code == 88 {
			return false
		}
	}
	defer resp.Body.Close()

	return true
}

//Check whether Twitter has really banned us
func check(tweet *Tweet, client *oauth.Consumer, canFav *bool, retry chan<- bool) {
	log.Println("Looks like we've hit the limit. Trying one more time...")
	//Try to send a fav
	if testConn(tweet, client) {
		*canFav = true
	} else {
		*canFav = false
		//Send a message to check again in 1 minute
		time.AfterFunc(1*time.Minute, func() {
			retry <- true
		})
		log.Println("Fav limit has been reached. Going to retry in 1 minute.")
	}

}

//Retry connection check (try to send a favorite again)
func retryCheck(tweet *Tweet, client *oauth.Consumer, canFav bool, retry chan<- bool, ns []time.Duration, periodIndex int) (bool, int) {
	log.Println("Checking again...")

	//Test connection again
	if testConn(tweet, client) {
		//Fav creation is allowed again, update parameters accordingly
		canFav = true
		periodIndex = 0

	} else {
		//Can't create favs yet
		canFav = false

		//Send retry message after period
		time.AfterFunc(ns[periodIndex] * time.Minute, func() {
			retry <- true
		})

		log.Println("Still banned from creating favorites. Going to retry in", ns[periodIndex].Nanoseconds(), "minutes.")

		//Increase period for next call
		if periodIndex < (len(ns) - 1) {
			periodIndex++
		}
	}

	return canFav, periodIndex
}
