package main

import (
	"os"
	"strconv"
)

func port() int {
	res := os.Getenv("DIDT_APP_PORT")

	if res == "" {
		return 8019
	}

	resInt, err := strconv.Atoi(res)
	if err != nil {
		return 8019
	}

	return resInt
}

func isProduction() bool {
	res := os.Getenv("ENV")

	return res == "production"
}

func databaseURL() string {
	return os.Getenv("DATABASE_URL")
}

func isFirstRun() bool {
	res := os.Getenv("FIRST_RUN")

	return res == "true"
}
