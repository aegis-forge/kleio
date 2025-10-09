package github

import (
	"io"
	"net/http"
)

func PerformApiCall(uri, bearer string, body io.Reader) (io.ReadCloser, int, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.github.com/"+uri, body)

	if err != nil {
		return nil, -1, err
	}

	req.Header.Set("Authorization", "Bearer "+bearer)
	res, err := client.Do(req)

	if err != nil {
		return nil, -1, err
	}

	// if res.StatusCode != http.StatusOK {
	// 	fmt.Printf("[LOG] %s -- status code: %s", uri, res.Status)
	// 	return nil, res.StatusCode, nil
	// }

	return res.Body, res.StatusCode, nil
}
