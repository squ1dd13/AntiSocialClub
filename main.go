package main

import (
	"fmt"
	"log"
	"socialclub/social_club"
	"time"

	"github.com/briandowns/spinner"
)

func main() {
	loading := spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithHiddenCursor(true))
	loading.Reverse()
	loading.Prefix = "Logging in. Please wait.  "

	session, _ := social_club.LoadSession()

	if session != nil && session.Expired() {
		fmt.Println("Saved session has expired.")
		session = nil
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

	println(session)
}
