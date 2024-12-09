package main

import "os"

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
