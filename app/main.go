package main

import (
	"app/app/crawler"
	"app/app/helpers"
)

func main() {
	neoDriver, neoCtx := helpers.Initialize()
	crawler.ExtractWorkflows(neoDriver, neoCtx)
}
