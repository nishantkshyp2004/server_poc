package frontend

type Request struct{
	request_id string
	request_url []string
}

func NewRequest(request_id string, s []string ) *Request{

	return &Request{
		request_id: request_id,
		request_url: s,
	}
}

//created to add url from the file to Request struct
func (r *Request) AddUrl( url string) {
	r.request_url = append(r.request_url, url)
}
//Returns the requestid
func (r *Request) RequestIdReturn() string{
	return r.request_id
}
//Returns the request url
func (r *Request) RequestUrlReturn() []string{
	return r.request_url
}