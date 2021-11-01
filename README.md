# AntiSocialClub
## What?
AntiSocialClub is a program that can dump the account files for any user of the [Rockstar Games Social Club](https://socialclub.rockstargames.com/).
Data for various cloud-enabled Rockstar products is stored in a folder on the Rockstar servers. By imitating a GTA: San Andreas game, AntiSocialClub
can access these files and download them for the user to see.

AntiSocialClub provides no facilities for accessing any account other than one which the user has the login details for.

The program currently works pretty badly. Within the year (or so) where I didn't touch this project, something seems to
have changed about how the Social Club works, and it now takes two or three attempts to fully authenticate with the server
and dump the user files. Hopefully I'll get round to fixing that at some point.

## Why?
In late 2020, I started wondering how hard it would be to write a program that could modify the GTA:SA save files on
my [Social Club](https://socialclub.rockstargames.com/) account. In my research, I discovered that each user has their
own folder on the Social Club server which stores the files for multiple cloud-enabled games, and the GTA:SA files were
only two of a much larger number of files. I managed to achieve my aim of modifying the GTA:SA saves, but I think access
to the other files could be interesting for players of other Rockstar titles as well.

## Files

- `social_club/crypto.go` - a Go implementation of the Social Club encryption algorithm
- `social_club/filesystem.go` - provides an interface for interacting with user files
- `social_club/network.go` - facilitates starting a new session (authentication etc.) and provides networking utils
- `interaction.go` - general user input stuff