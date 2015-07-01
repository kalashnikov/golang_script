package note

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func UpdateMenuFile() string {
	os.Chdir("/var/opt/www/go/note/")
	if _, err := os.Stat("menu.md"); err == nil {
		os.RemoveAll("menu.md") // Remove old one

		// Do the conversion before write out
		// No additional encoding config for file needed
		if out, err := os.Create("menu.md"); err == nil {
			out.WriteString("---\n")
			out.WriteString("## Notes Contents List\n")
			if a, err := filepath.Glob("**/*.md"); err == nil {
				sort.Strings(a)

				folderName := ""
				for _, i := range a {
					ary := strings.Split(string(i), "/")
					if folderName != ary[0] {
						out.WriteString("* **" + ary[0] + "**\n")
						folderName = ary[0]
					}
					name := strings.Split(string(ary[1]), ".")[0]
					out.WriteString("  - [" + name + "](" + i + ")\n")
				}
			}
			out.WriteString("---")
		}
	}

	b, _ := ioutil.ReadFile("menu.md")
	os.Chdir("/var/opt/www/go/")
	return string(b)
}

func GetNoteContents(fp string) (name, contents string) {
	os.Chdir("/var/opt/www/go/note/")
	_, err := os.Stat(fp)
	if err != nil {
		panic(err)
	}
	b, _ := ioutil.ReadFile(fp)
	n := strings.Split(path.Base(fp), ".")[0]
	os.Chdir("/var/opt/www/go/")
	return n, string(b)
}

func NoteAuth(username, password string) bool {
	return username == os.Getenv("NOTE_ACCOUNT") && password == os.Getenv("NOTE_PASSWORD")
}