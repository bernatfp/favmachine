package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

//Config file structure
type Config struct {
	APIKey            string
	APISecret         string
	AccessToken       string
	AccessTokenSecret string
	Words             string
}

//Load configuration
func loadConfig() *Config {
	data, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatal("Error while trying to read file: ", err)
	}

	config := new(Config)
	err = json.Unmarshal(data, config)
	if err != nil {
		log.Fatal("Error while trying to load JSON data: ", err)
	}

	return config
}
