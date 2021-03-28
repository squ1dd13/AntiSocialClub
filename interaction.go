package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"socialclub/social_club"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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

func login() *social_club.Session {
	loading := spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithHiddenCursor(true))
	loading.Reverse()
	loading.Prefix = "Logging in. Please wait.  "

	session, _ := social_club.LoadSession()

	if session != nil && session.Expired() {
		fmt.Println("Saved session has expired.")
		session = nil
	} else if session != nil {
		expirationTime := time.Unix(session.ExpirationTime(), 0)
		fmt.Printf("Saved session will be valid until %s.\n", expirationTime.Local().Format(time.Stamp))
	}

	if session == nil {
		fmt.Println(stringPleaseLogIn)
	}

	failedOnce := false

	var err error

	for session == nil {
		email := inputEmail()
		password := inputPassword()

		loading.Start()
		session, err = social_club.LogIn(email, password)
		loading.Stop()

		if err != nil {
			if err.Error() == "AuthenticationFailed: InvalidCredentials" {
				fmt.Println(stringInvalidCredentials)

				// If this is not the first failure, warn the user.
				if failedOnce {
					fmt.Println(stringIpWarning)
				}

				failedOnce = true
				continue
			}

			fmt.Println("Unknown error.")
			log.Fatal(err)
		}

		// Ask the user if they want to stay logged in. A new session will be valid for 24 hours, so we can
		//  save the ticket and reuse it within that 24h period.
		fmt.Print(stringStayLoggedIn)

		if inputBoolean() {
			err = session.Save()

			if err != nil {
				fmt.Printf("Unable to save session: %v\n", err)
			}
		}

		// Login successful, so break out of the loop.
		break
	}

	user := session.User()
	fmt.Printf("Logged in as '%s' (%s).\n", user.Nickname, user.Email)

	return session
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
