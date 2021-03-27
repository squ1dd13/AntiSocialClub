package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

func inputLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	line := scanner.Text()

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return line
}

func inputEmail() string {
	fmt.Print("Email address: ")
	return inputLine()
}

func inputPassword() string {
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()

	return string(passwordBytes)
}

func inputBoolean() bool {
	for {
		input := strings.ToLower(inputLine())

		if input == "y" {
			return true
		}

		if input == "n" {
			return false
		}

		fmt.Print("Please enter 'y' for 'yes' or 'n' for 'no': ")
	}
}

// Define long strings here so they don't get in the way of code.
const (
	stringPleaseLogIn = `Please log into your Social Club account. 
Your details are only sent to Rockstar games.`
	stringInvalidCredentials = "Error: Invalid credentials. Please try again."
	stringIpWarning          = `Warning: Too many consecutive failed logins may cause Rockstar to block your IP. 
Try to avoid this, but if it does happen you can obtain a new IP by restarting your router.`
	stringStayLoggedIn = "Stay logged in for the next 24 hours? (Note that your password will not be stored.) [y/n] "
)
