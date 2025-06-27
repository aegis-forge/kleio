package helpers

import (
	"gopkg.in/ini.v1"
	"log"
	"path"
	"runtime"
)

func ReadConfig() (*ini.File, error) {
	_, filename, _, _ := runtime.Caller(1)
	config, err := ini.Load(path.Join(path.Dir(filename), "../../config.ini"))

	if err != nil {
		log.Fatal("Make sure that a `config.ini` file exists at the root of the repository")
	}

	return config, err
}
