package main

import (
	"github.com/mrjones/oauth"
	"log"
	"strconv"
)

//Create a FAV
func (tweet *Tweet) fav(client *oauth.Consumer, statsch chan<- int, errch chan<- int) {
	params := map[string]string{"id": strconv.FormatUint(tweet.Id, 10)}
	resp, err := client.Post("https://api.twitter.com/1.1/favorites/create.json", params, atoken)
	if err != nil {
		log.Println("Error sending FAV ", err)
		log.Println("Fav failed: ", tweet.Text)
		//Parse err data into struct
		errData := new(ErrResponse)
		errData.loadErrData(err)
		//Notify channels accordingly
		errData.notifyChans(statsch, errch)

	} else {
		//OK
		statsch <- 1
		log.Println("Fav sent: ", tweet.Text)
	}
	defer resp.Body.Close()
}

//This function checks stuff before creating a fav
func (tweet *Tweet) handler(client *oauth.Consumer, statsch chan<- int, errch chan<- int, canFav bool) {
	//Send fav if it's not blocked by API
	if canFav {
		//Filter if it is a RT because we might have already favourited it yet
		if len(tweet.Retweet.Id) > 0 {
			return
		}
		//Send fav
		tweet.fav(client, statsch, errch)
	}
}
