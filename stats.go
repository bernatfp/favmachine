package main

import (
	"log"
	"math"
	"time"
)

//Prints stats collected
func printStats(favNum, failFav int, start time.Time) {
	log.Println("Favs sent until error: ", favNum)
	log.Println("Favs failed: ", failFav)
	log.Println("Total: ", favNum+failFav)
	log.Println("Started at: ", start.Hour(), "h ", start.Minute(), "m ", start.Second(), "s")
	finish := time.Now()
	log.Println("Finished at: ", finish.Hour(), "h ", finish.Minute(), "m ", finish.Second(), "s")
	duration := time.Since(start)
	log.Println("Elapsed time: ", int(math.Floor(duration.Hours()))%24, "h ", int(math.Floor(duration.Minutes()))%60, "m ", int(math.Floor(duration.Seconds()))%60, "s")
}

//Collects stats about favs sent
func countfavs(favch <-chan int) {

	//Initialize counters and timer
	favNum := 0
	failFav := 0
	start := time.Now()

	//Process info about favs sent
	for fav := range favch {
		switch fav {
		//Fine
		case 1:
			favNum++

		//Failed fav (RT, deleted tweet or such)
		case 2, 3:
			failFav++

		//(Daily) Limit exceeded
		case -1:
			log.Println("Error, limit exceeded.")
			printStats(favNum, failFav, start)
			return

		//Unknown code error
		default:
			log.Println("Unknown error. Code: ", fav)
			printStats(favNum, failFav, start)
			return

		}

	}
}
