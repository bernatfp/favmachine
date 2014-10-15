package main

import (
	"github.com/mrjones/oauth"
	"log"
	"time"
	"strconv"
)


//Send a test fav
func testConn(tweet *Tweet, client *oauth.Consumer) (bool, int) {
	params := map[string]string{"id": strconv.FormatUint(tweet.Id, 10)}
	resp, err := client.Post("https://api.twitter.com/1.1/favorites/create.json", params, atoken)
	if err != nil {
		log.Println("Error sending test FAV ", err)
		//Parse err data into struct
		errData := new(ErrResponse)
		errData.loadErrData(err)
		//Check if the error is related to banned API access
		if errData.APIErr[0].Code > 0 {
			return false, errData.APIErr[0].Code
		} else {
			return false, errData.HTTPErr.Code
		}
	}
	defer resp.Body.Close()

	return true, 0
}

//Check whether Twitter has really banned us
func check(tweet *Tweet, client *oauth.Consumer, canFav *bool, retry chan<- bool) {
	log.Println("Looks like we've hit the limit. Trying one more time...")
	//Try to send a fav
	if _, code := testConn(tweet, client); code != 429 && code != 88 { //We just check if we are banned because of surpassing the limit
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
	if _, code := testConn(tweet, client); code != 429 && code != 88 { //We just check if we are banned because of surpassing the limit
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
