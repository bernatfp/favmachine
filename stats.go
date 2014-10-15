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
func countfavs(statsch <-chan int, stopHours chan<- int) {

	//Initialize counters and timer
	favNum := 0
	failFav := 0
	start := time.Now()

	//Process info about favs sent
	for fav := range statsch {
		switch fav {
		//Fine
		case 1:
			favNum++

		//Failed fav (RT, deleted tweet or such), not critical
		case 139, 34, 136:
			failFav++

		//Suspended
		case 64:
			log.Println("Error, account suspended.")
			printStats(favNum, failFav, start)
			return

		//Rate limit exceeded
		case 88:
			log.Println("Error, rate limit exceeded.")
			printStats(favNum, failFav, start)
			return

		//HTTP 5XX are associated to these API errors
		case 130, 131:
			log.Println("Error, Twitter is experiencing an internal error or is over capacity.")
			printStats(favNum, failFav, start)
			return

		//Too many requests
		case 429:
			log.Println("Error, too many requests.")
			printStats(favNum, failFav, start)
			return

		//Unknown code error
		default:
			log.Println("Unknown error. Code: ", fav)
			printStats(favNum, failFav, start)
			return

		}

		log.Println("Total favs: ", favNum + failFav)

		//To prevent being suspended, we stop execution for certain time when we've reached a reasonable limit 
		//If enough time has passed (24h), we continue
		if favNum + failFav == 1000 {
			log.Println("1000 favs limit reached, checking if we have to wait...")
			hours := time.Since(start).Hours()
			if hours < 24.0 {
				stopHours <- 24 - int(math.Ceil(hours))
				return
			}
			log.Println("Can continue.")
		}

	}
}
