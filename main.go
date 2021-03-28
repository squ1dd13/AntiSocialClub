package main

import "socialclub/social_club"

func main() {
	session := login()

	social_club.SetFilesystemSession(session)

	println(session.CreateUrl("/"))
	social_club.UserDirectory().PrintTree(0)
}
