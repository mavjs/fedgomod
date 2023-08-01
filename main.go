package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/net/publicsuffix"
)

var KNOWN_FORGES = map[string]string{
	"github.com":    "github",
	"gitlab.com":    "gitlab",
	"bitbucket.org": "bitbucket",
	"pagure.io":     "pagure",
	"gitea.com":     "gitea",
}

func main() {
	if len(os.Args) <= 1 {
		log.Fatalln("no go.mod file of project given")
	}
	fileToCheck, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatalln("error during file check", err)
	}

	if _, err := os.Stat(fileToCheck); err != nil {
		log.Fatalln(err)
	}

	fileBytes, err := os.ReadFile(fileToCheck)
	if err != nil {
		log.Fatalln("unable to read file", err)
	}

	parsedFile, err := modfile.Parse(fileToCheck, fileBytes, nil)
	if err != nil {
		log.Fatalln("error while parsing go.mod file", err)
	}

	rpmGoVer := regexp.MustCompile(`^v\d`)
	rpmDenyStr := strings.NewReplacer(".", "-", "_", "-", "/", "-", "~", "-")

	for _, dep := range parsedFile.Require {
		if !dep.Indirect {

			splitMod := strings.Split(dep.Mod.Path, "/")

			for idx, modParts := range splitMod {
				// only if the PublicSuffix returns that it is managed by ICANN and index of the splitMod is less than 2,
				// then try to remove the "TLD" from the package path. If not, return as in, and we will deal with it when
				// we replace them later on. This is because most golang package import paths start with a domain.
				// Either they are in known git forge (e.g. Github, Gitlab) or domains (e.g. go.pkg, k8s.io), etc.
				if eTLD, status := publicsuffix.PublicSuffix(modParts); status && idx < 2 {
					splitMod[idx] = strings.Split(modParts, "."+eTLD)[0]
				}
			}

			// prepend "golang" to list of go name
			splitMod = append([]string{"golang"}, splitMod...)
			lenModPath := len(splitMod)

			lastPath := splitMod[lenModPath-1]
			if ok := rpmGoVer.MatchString(lastPath); ok {
				splitMod[lenModPath-1] = strings.TrimPrefix(lastPath, "v")
			}

			// append "devel" to list of go name
			splitMod = append(splitMod, "devel")

			results := []string{}
			tokens := make(map[string]bool, len(splitMod))
			tokens["go"] = true
			var fedoraPackageName string
			for _, elem := range splitMod {
				if _, value := tokens[elem]; !value {
					tokens[elem] = true
					results = append(results, elem)
				}
			}
			fedoraPackageName = strings.Join(results, "-")
			fedoraPackageName = rpmDenyStr.Replace(fedoraPackageName)

			fmt.Println(dep.Mod.Path, "->", fedoraPackageName)
		}
	}
}
