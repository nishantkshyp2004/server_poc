package main

import (
	"net/http"
	"log"
	"github.com/gorilla/mux"
	"io"
	"fmt"
	"strings"
	"encoding/json"
	"time"
	"github.com/rs/xid"
	"github.com/mediocregopher/radix.v2/pool"
	"strconv"
	"errors"

)

type URL struct{
	Url string
	State int
}

type Request struct{
	Id string
	Url []URL
	Response map[string][]string
	Rstate string
}

var db *pool.Pool

func init() {
	var err error
	// Establish a pool of 10 connections to the Redis server listening on
	// port 6379 of the local machine.
	db, err = pool.New("tcp", "localhost:6379", 10)
	if err != nil {
		log.Panic(err)
	}
}

func rHandler(w http.ResponseWriter, r *http.Request)(string){


	if r.Method == "GET" {
		requestid := r.URL.Query().Get("id")
		reply, err := db.Cmd("GET", requestid)

		var deserialized Request

		err = json.Unmarshal(reply, &deserialized)
		if err != nil {
			panic(err)
		}

		if deserialized.Rstate == "processing_complete" {
			return string(deserialized.Response)

		}
		return string(reply)
	}

	var requested_url = []string{}
	mr, err := r.MultipartReader()
	if err != nil {
		fmt.Fprintln(w, err)

	}
	length := r.ContentLength

	for {
		var read int64
		part, err := mr.NextPart()
		if err == io.EOF {
			fmt.Printf("\nDone!")
			break
		}
		for {
			buffer := make([]byte, 100000)
			cBytes, err := part.Read(buffer)
			if err == io.EOF {
				fmt.Printf("\n Finished reading buffer!")
				break
			}
			read = read + int64(cBytes)
			fmt.Printf("\r read: %v  length : %v \n", read, length)

			if read > 0 {
				fmt.Printf("Reading : %s", string(buffer[0:cBytes]))
				all_url := string(buffer[0:cBytes])
				url := strings.Split(all_url, "\n")
				//appending the requested url
				requested_url = append(requested_url, url...)
				fmt.Printf("requested_url: %s", requested_url)
			} else {
				fmt.Printf("\nDone reading a part!")
				break
			}
		}
	}

	//Generating the Unique ID and setting the response header.
	guid := xid.New()
	request_id := guid.String()
	fmt.Println("UID:", request_id)

	w.Header().Set("req_id", request_id)

	//creating the channel.
	ch_fb := make(chan string)
	ch_br := make(chan string)

	go frontend_processing(request_id, requested_url, ch_fb)
	go bidder_processing(ch_fb, ch_br)
	go reducer_processing(ch_br)
	return "Done"
}


func frontend_processing(request_id string, requested_url []string, ch chan string){

	var r Request
	r.id = request_id
	r.rstate="frontent_processing"
	r.url = make([]URL, 0)

	for _,url_val :=range requested_url{
		url_struct := URL{ url: url_val, }
		r.url = append( r.url, url_struct)
	}

	request_struct := &r
	serialized, err := json.Marshal(request_struct)
	if err != nil{
		panic(err)
	}
	fmt.Println("serialized data after frontent process: ", string(serialized))
	// will get the pool connection and put it back to the pool.
	db.Cmd("SET", request_id, serialized)
	ch <- request_id
}

//setting timeout
var timeout = time.Duration(5 * time.Second)
var ErrNoRequestId = errors.New("No request id found")

func bidder_processing(ch_fb chan string, ch_br chan string) {

	requestid := <-ch_fb
	reply, err :=db.Cmd("GET", requestid)

	// if the request id is not found.
	if len(reply) == 0{
		panic(ErrNoRequestId)
	}else if err != nil{
			panic(err)
	}

	var deserialized Request

	err = json.Unmarshal(reply, &deserialized)
	if err !=nil{
		panic(err)
	}

	urls_struct := deserialized.url
	deserialized.rstate = "bidder_processing"

	//creating custom client to embed the timeout functionality.
	client := http.Client{
		Timeout: timeout,
	}

	//Using channels and go routine to make the request Async and to collect the response whenever achieved back via channel
	http_chan := make(chan string)
	url_count:=1
	for _, value :=range urls_struct{
		url := value.url
		url_count++
		go func(url string, http_chan chan string){
			res, err := client.Get(url)
			status_code := strconv.Itoa(res.StatusCode)
			url_status := url +'@' + status_code
			http_chan <- url_status

		}(url, http_chan)
	}

	// Collecting the response back via channel using infinite for-loop and select-case.

	http_response_count :=1
	for {
		select{
			case http_response := <- http_chan:
				fmt.Printf("%s response is fetched. ", http_response)
				response := strings.Split(http_response, "@")

				//updating the struct with the url`s status code obtained.

				for indx, value := range urls_struct {
					if value.url == response[0]{
						deserialized.url[indx].state = response[1]
						break
					}

				}
				//break the infinite loop after all url response is finished.
				if http_response_count == url_count{
					break
				}
				http_response_count++
			case <-time.After(50 * time.Millisecond):
				fmt.Printf(".")
		}
	}

	//Serializing the struct again and putting it to the channel
	serialized, err := json.Marshal(deserialized)
	fmt.Println("serialized data after bidder process: ", string(serialized))
	// will get the pool connection and put it back to the pool.
	db.Cmd("SET", request_id, serialized)
	ch_br <-requestid

}

func reducer_processing(ch_br chan string){
	requestid := <-ch_br
	reply, err := db.Cmd("GET", requestid).Map()
	// if the request id is not found.
	if len(reply) ==0 {
		panic(ErrNoRequestId)
	}else if err != nil{
		panic(err)
	}

	var deserialized Request
	err = json.Unmarshal(reply, &deserialized)
	if err !=nil{
		panic(err)
	}

	deserialized.rstate = "processing_complete"

	var result map[string][]string
	for indx, value :=range deserialized.url{

		if val, ok :=result[value.state]; !ok{
			urls :=[]string{}
			for i:=indx; i<len(deserialized.url)-indx;i++{

				result[value.state] = append(urls, result[value.url])
			}

		}else{continue }
	}


	deserialized.response = result

	//Serializing the struct again and putting it to the channel
	serialized, err := json.Marshal(deserialized)
	fmt.Println("serialized data after bidder process: ", string(serialized))
	// will get the pool connection and put it back to the pool.
	db.Cmd("SET", request_id, serialized)

}


func main() {
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/requests", rHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}



