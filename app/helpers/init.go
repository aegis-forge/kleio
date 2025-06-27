package helpers

import "app/app/database"

func Initialize() {
	config, err := ReadConfig()

	if err != nil {
		panic(err)
	}

	_, _, err = database.ConnectToNeo(config)

	if err != nil {
		return
	}
}
