package main

import (
	"app/app/crawler"
	"app/pkg/git"
	"fmt"
)

func main() {
	neoDriver, neoCtx := crawler.Initialize()
	crawler.ExtractWorkflows(neoDriver, neoCtx)
	
	git.DeleteRepo("../tmp")

	fmt.Println("All Done")
}
