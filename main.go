package main

import (
	"fmt"
	"os"
	"path/filepath"
	"socialclub/social_club"
)

func main() {
	session := login()

	social_club.SetFilesystemSession(session)

	currentDirectory, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	fmt.Printf("Base URL is %s\n", session.CreateUrl("/"))

	basePath := filepath.Join(currentDirectory, "dump")
	fmt.Printf("Dumping to %s\n", basePath)

	social_club.UserDirectory().PrintTree(0)

	err = social_club.UserDirectory().Dump(basePath)

	if err != nil {
		panic(err)
	}
}
