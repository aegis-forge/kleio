package helpers

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"path"
	"runtime"
)

func ReadConfig() (*ini.File, error) {
	fmt.Print("\u001B[37m[INIT]\u001B[0m Reading config file")

	_, filename, _, _ := runtime.Caller(0)
	config, err := ini.Load(path.Join(path.Dir(filename), "../../config.ini"))

	if err != nil {
		fmt.Println(" \u001B[31m𐄂\u001B[0m")
		log.Fatal("Make sure that a `config.ini` file exists at the root of the repository")
	}

	fmt.Println(" \u001B[32m✓\u001B[0m")

	return config, err
}
