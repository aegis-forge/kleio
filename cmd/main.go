package main

import (
	"app/cmd/crawler"
	"app/pkg/git"
	"fmt"
)

func main() {
	neoDriver, neoCtx, mongoClient := crawler.Initialize()
	crawler.ExtractWorkflows(neoDriver, neoCtx, mongoClient)

	git.DeleteRepo("./tmp")

	fmt.Println("All Done")
}
