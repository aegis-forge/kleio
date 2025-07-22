package main

import (
	"app/app/crawler"
	"fmt"
)

func main() {
	neoDriver, neoCtx := crawler.Initialize()
	crawler.ExtractWorkflows(neoDriver, neoCtx)

	fmt.Println("All Done")
}
