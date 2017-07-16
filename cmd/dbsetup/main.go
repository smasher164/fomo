package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	cfg := &Config{
		Host:      "127.0.0.1",
		AdminPort: "8082",
		SQLPort:   "26257",
		User:      "akhil",
		Dbname:    "fomo",
		Insecure:  true,
	}
	if err := cfg.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg.CreateUser()
	cfg.CreateDb()
}
