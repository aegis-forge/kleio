package main

import (
	"app/pkg/git"
)

func main() {
	//helpers.Initialize()
	_, err := git.GetHistory(
		"/Users/edoriggio/Documents/personal/github/personal/www/",
		".github/workflows/deploy.yml")

	if err != nil {
		panic(err)
	}
}
