package social_club

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var fsSession *Session

func SetFilesystemSession(session *Session) {
	fsSession = session
}

// For the weird .NET date format the server uses.
type fileDate time.Time

func (date *fileDate) UnmarshalJSON(data []byte) error {
	// Remove the /Date()/ from the string.
	dateString := strings.TrimSuffix(strings.TrimPrefix(string(data), "\"\\/Date("), ")\\/\"")

	timeValue, err := strconv.ParseInt(dateString, 10, 64)

	if err != nil {
		return err
	}

	*date = fileDate(time.Unix(timeValue/1000, 0))

	return nil
}

type Item struct {
	Name            string   `json:"Name"`
	Type            string   `json:"Type"`
	LastModifiedUtc fileDate `json:"LastModifiedUtc"`

	Parent *Item
	path   string
}

func (item *Item) IsDirectory() bool {
	return item.Type == "D"
}

// ListContents fixme: Sometimes doesn't work at all. If it times out once, it
// won't work at all. The program has to be restarted in such cases. This
// probably happens 60-70% of the time.
func (item *Item) ListContents() ([]*Item, error) {
	if !item.IsDirectory() {
		return nil, errors.New("not a directory")
	}

	// Open the directory.
	jsonBytes, err := fsSession.Fetch(item.path)

	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON.
	var opened openedDirectory
	err = json.Unmarshal(jsonBytes, &opened)

	if err != nil {
		return nil, err
	}

	contents := opened.Contents

	for _, child := range contents {
		fmt.Println("found a child")
		child.path = filepath.Join(item.path, child.Name)
		child.Parent = item
	}

	return opened.Contents, nil
}

func (item *Item) Dump(basePath string) (err error) {
	fullPath := filepath.Join(basePath, item.path)

	// If there has been a dump to the same path before, we need to remove those files.
	err = os.RemoveAll(fullPath)

	if err != nil {
		return err
	}

	if item.IsDirectory() {
		// Create the directory.
		err = os.MkdirAll(fullPath, 0777)

		if err != nil {
			return err
		}

		// Get all the directory entries.
		contents, err := item.ListContents()

		if err != nil {
			return err
		}

		// Dump the entries.
		for _, child := range contents {
			err = child.Dump(basePath)

			// TODO: Keep going here and return a slice of failures maybe?
			if err != nil {
				return err
			}
		}

		return nil
	}

	data, err := fsSession.Fetch(item.path)

	if err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, 0666)
}

func (item *Item) PrintTree(startLevel int) {
	fmt.Println(item.path)

	if item.IsDirectory() {
		fmt.Println("listing contents")
		contents, err := item.ListContents()
		fmt.Println("done listing contents")

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		for _, child := range contents {
			fmt.Println("found something")
			child.PrintTree(0)
		}
	}
}

type openedDirectory struct {
	Contents []*Item `json:"d"`
}

func UserDirectory() *Item {
	return &Item{
		Name:            "/",
		Type:            "D",
		LastModifiedUtc: fileDate{},
		Parent:          nil,
		path:            "/",
	}
}
