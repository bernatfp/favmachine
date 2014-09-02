package main

import (
	"encoding/json"
	"fmt"
	"github.com/mrjones/oauth"
	"log"
	"strings"
)

type ErrJsonArr struct {
	Errors []ErrJson
}

type ErrJson struct {
	Code    int
	Message string
}

func parseErrCode(err error) int {
	//Convert from type error to string
	errStr := fmt.Sprintf("%v", err)
	//Find the part of the JSON that mentions errors
	start := strings.Index(errStr, "{\"errors\":")
	//No error JSON returned by Twitter
	if start == -1 {
		return -1
	}
	end := strings.Index(errStr[start:], "\n")

	errJsonArr := new(ErrJsonArr)
	err = json.Unmarshal([]byte(errStr[start:start+end]), errJsonArr)
	if err != nil {
		fmt.Println("Error unmarshal: ", err)
	}

	return errJsonArr.Errors[0].Code
}

func favTweet(tweet *Tweet, client *oauth.Consumer, favch chan<- int, errch chan<- int) {
	params := map[string]string{"id": tweet.Id}
	resp, err := client.Post("https://api.twitter.com/1.1/favorites/create.json", params, atoken)
	if err != nil {
		log.Println("Error sending FAV ", err)
		switch code := parseErrCode(err); code {
		//Already FAV
		case 139:
			favch <- 2
		//Not found
		case 34:
			favch <- 3
		//No JSON error sent by Twitter
		//It's expected to be an HTTP 429 response code, which means we've exceeded the limit
		case -1:
			favch <- -1
			errch <- -1
		//Unknown error
		default:
			favch <- code
			errch <- -1
		}

		log.Println("Fav failed: ", tweet.Text)

	} else {
		//OK
		favch <- 1
		log.Println("Fav sent: ", tweet.Text)
	}
	defer resp.Body.Close()

}

//This function checks stuff before creating a fav
func tweetHandler(tweet *Tweet, client *oauth.Consumer, favch chan<- int, errch chan<- int, canFav bool) {

	//Send fav if it's not blocked by API
	if canFav {
		//Filter if it is a RT because we might have already favourited it yet
		if len(tweet.RT.Id) > 0 {
			return
		}

		//Send fav
		favTweet(tweet, client, favch, errch)
	}

}
