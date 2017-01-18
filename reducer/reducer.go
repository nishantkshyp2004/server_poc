package reducer

type ReducerRequest struct{
	request_id string
	url_response_reduce UrlResponseReduced
}

func NewBidderRequest(reducer_request_id string, reduce_url_response ReducedUrlResponse) *ReducerRequest{
	return *ReducerRequest{
		request_id: reducer_request_id,
		UrlResponse: reduce_url_response,
	}
}

type ReducedUrlResponse struct{
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

