package main

import (
	"net/http"
	"log"
	"github.com/gorilla/mux"
	"io"
	"fmt"
	"strings"
	"time"
	"io/ioutil"
	"strconv"
	"github.com/rs/xid"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/nishantkshyp2004/frontend"
	"github.com/nishantkshyp2004/bidder"
	//"github.com/nishantkshyp2004/reducer"
)

//func getredis() redis{
//
//	return c
//}

func rHandler(w http.ResponseWriter, r *http.Request) {

	var requested_url = []string{}
	mr, err := r.MultipartReader()
	if err != nil {
		fmt.Fprintln(w, err)
		return
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
	fmt.Println("UID:",request_id)
	w.Header().Set("req_id", request_id)

	// creating new request object
	request_struct := frontend.NewRequest(request_id, requested_url)

	//redis connection
	c, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		panic("error in connecting Redis server.")
	}

	defer c.Close()
	c.Cmd("HSET", "REQUEST", request_id, "")

	frontend_processing(request_struct, requested_url)
	response_data := bidder_processing(request_struct)
	reducer_processing(response_data)

}

func frontend_processing(request_struct *frontend.Request, urls []string){
	//adding url to the request generated
	for _,url_link := range urls{
		request_struct.AddUrl(url_link)
	}
}

//setting timeout
var timeout = time.Duration(5 * time.Second)

func bidder_processing(request_struct *frontend.Request) []*bidder.BidderRequest{
	requestid := request_struct.RequestIdReturn()
	urls_list := request_struct.RequestUrlReturn()
	responses := []*bidder.BidderRequest{}
	urls_response_channel := make(chan *bidder.BidderRequest, len(urls_list))

	//creating custom client to embed the timeout functionality.
	client := http.Client{
		Timeout: timeout,
	}

	//Using channels and go routine to make the request Async and to collect the response whenever achieved back via channel
	for _, url :=range urls_list{
		go func(url string){
			res, err := client.Get(url)
			body, err := ioutil.ReadAll(res.Body)
			status_code := strconv.Itoa(res.StatusCode)
			resp_body := string(body)
			url_response := bidder.NewUrlResponse(url, status_code, resp_body)
			urls_response_channel <- bidder.NewBidderRequest(requestid, *url_response)
		}(url)
	}
	// Collecting the response back via channel using infinite for-loop and select-case.
	for {
		select {
			case bidder_request_response := <- urls_response_channel:
				fmt.Printf("%s response is fetched. ", bidder_request_response.url_response.url)
				responses = append(responses, bidder_request_response)
				if len(responses) == len(urls_list){
					return responses
				}
			case <-time.After(50 * time.Millisecond):
				fmt.Printf(".")
		}
	}

}

func reducer_processing(responses []*bidder.BidderRequest){
	fmt.Println(responses)
}

func main() {
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/requests", rHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}



