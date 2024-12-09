package main

import "os"

func isProduction() bool {
	res := os.Getenv("ENV")

	return res == "production"
}
