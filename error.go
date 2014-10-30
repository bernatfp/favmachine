package main

import (
	"strings"
	"strconv"
	"encoding/json"
	"log"
	"time"
)


//Contains structured data about the error received
type ErrResponse struct {
	HTTPErr ErrJson
	APIErr []ErrJson `json:"errors"`
	Err error 
}

//If code is 0 => empty
//If code is -1 => error
//Otherwise code is the same as received in the response
type ErrJson struct {
	Code    int
	Message string
}

//Assigns a custom API error object to APIErr instead of the received
func (errStruct *ErrResponse) CustomAPIErr(code int, message string) {
	errStruct.APIErr = make([]ErrJson, 1)
	errStruct.APIErr[0].Code = code
	errStruct.APIErr[0].Message = message
}

//Loads error data related to a response from Twitter's API
func (errStruct *ErrResponse) LoadAPIErr(errStr string){	
	//Find the part of the JSON that mentions errors
	start := strings.Index(errStr, "{\"errors\":")
	//No error JSON returned by Twitter
	if start == -1 {
		errStruct.CustomAPIErr(0, "No error JSON in the response")
		return
	}
	end := strings.Index(errStr[start:], "\n")

	err := json.Unmarshal([]byte(errStr[start:start+end]), errStruct)
	if err != nil {
		log.Println("Error unmarshal: ", err)
		errStruct.CustomAPIErr(-1, "Error Unmarshal")
	}
}

func (errStruct *ErrResponse) LoadHTTPErr(errStr string){
	//Parse Code and Message
	sep := "Response Status: '"
	start := strings.Index(errStr, sep) + len(sep)
	if start - len(sep) == -1 {
		errStruct.HTTPErr.Code = -1
		errStruct.HTTPErr.Message = "No HTTP status attached to response"
		return
	}
	end := strings.Index(errStr[start:], "'")
	errParts := strings.SplitN(errStr[start:start+end], " ", 2)

	//Assign to struct
	var err error
	errStruct.HTTPErr.Code, err = strconv.Atoi(errParts[0])
	if err != nil {
		log.Println("Error parsing HTTP Code: ", err)
		errStruct.HTTPErr.Code = -1
	}
	errStruct.HTTPErr.Message = errParts[1]
}

func (errStruct *ErrResponse) loadErrData(err error){
	//Store original error
	errStruct.Err = err
	
	//Populate error object
	errStr := err.Error()
	errStruct.LoadAPIErr(errStr)
	errStruct.LoadHTTPErr(errStr)

}

//Implement error interface
func (errStruct *ErrResponse) Error() string {
	return errStruct.Err.Error()
}

//Handles notification of errors to channels
func (errData *ErrResponse) notifyChans(statsch chan<- int, errch chan<- int) {
	//Shortcuts
	HTTPCode := errData.HTTPErr.Code
	APICode := errData.APIErr[0].Code

	//Check if the account has been suspended
	if HTTPCode == 403 && APICode == 64 {
		statsch <- APICode
		//Critical error, must be notified to main process
		errch <- APICode
		//Wait enough time to have printed stats first
		time.Sleep(1 * time.Second)
		log.Fatal("Your account has been suspended. To regain access log into your Twitter account, click on the red banner on top of the page and follow the steps to complete the process. ")
	}

	//If any of the following conditions are met, execution is temporarily stopped until favs can be created again
	//
	//Check if we've reached a limit
	//Twitter Code 88 => Rate Limit Exceeded
	if APICode == 88 {
		statsch <- APICode
		//Notify connection has to be rechecked (any value other than 64)
		errch <- APICode
		return
	}
	//HTTP 429 => Too Many Requests
	if HTTPCode == 429 {
		statsch <- HTTPCode
		//Notify connection has to be rechecked (any value other than 64)
		errch <- HTTPCode
		return
	}
	//HTTP 5XX
	if APICode == 130 || APICode == 131 {
		statsch <- APICode
		//Notify connection has to be rechecked (any value other than 64)
		errch <- APICode
		return
	}



	//Check for other errors
	switch APICode {
		//Irrelevant errors => Already FAV or Tweet not found or blocked by user x
		case 139, 34, 136:
			statsch <- APICode
		//Unknown error
		default:
			log.Println("Unknown Error")
			log.Println(errData.Error())
			statsch <- APICode
			time.Sleep(1 * time.Second)
			log.Fatal("Exit, fix error")
	}
}

