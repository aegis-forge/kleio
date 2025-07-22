package github

import (
	"errors"
	"io"
	"net/http"
)

func PerformApiCall(uri string, bearer string, body io.Reader) (io.ReadCloser, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.github.com/"+uri, body)
	
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+bearer)
	res, err := client.Do(req)
	
	if err != nil {
		return nil, err
	}
	
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(uri+" -- status code: "+res.Status)
	}
	
	return res.Body, nil
}
