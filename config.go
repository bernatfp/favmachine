package main

import (
	"io/ioutil"
	"log"
	"encoding/json"
)

type Config struct {
	APIKey string
	APISecret string
	AccessToken string
	AccessTokenSecret string
	Words string
}

func loadConfig() (*Config){
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