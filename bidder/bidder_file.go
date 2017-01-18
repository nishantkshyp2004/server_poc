package bidder

type BidderRequest struct{
	request_id string
	url_response UrlResponse
}

func NewBidderRequest(bidding_request_id string, UrlResponseStruct UrlResponse) *BidderRequest{
	return &BidderRequest{
			      request_id: bidding_request_id,
			      url_response: UrlResponseStruct,
			      }
}

type UrlResponse struct{
	url string
	status_code string
	body string
}

func NewUrlResponse(url string, status_code string, body string) *UrlResponse{
	return &UrlResponse{
		url : url,
		status_code : status_code,
		body: body,
	}
}


