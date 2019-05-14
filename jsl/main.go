package main

import (
	"log"
	"os"

	"github.com/graham/jsl/jsl/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
