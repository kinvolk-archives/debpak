package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	dependancy  string = "uldep"
	recommended string = "ulrec"
	suggested   string = "ulsug"

	state_pkg = iota
	state_deps
	state_sha
)

type FlatpakSource struct {
	Type   string `json:"type"`
	Url    string `json:"url"`
	Sha256 string `json:"sha256"`
}

type FlatpakModule struct {
	Name       string          `json:"name"`
	ConfigOpts string          `json:"config-opts"`
	Srcs       []FlatpakSource `json:"sources"`
}

type depBuilder struct {
	pkgs    map[string]struct{} // Used to avoid package duplications
	hashes  map[string]struct{} // Used to avoid tarball duplications
	modules []FlatpakModule
	total   int // total packages visited
	dups    int // dependency already picked up from other package(s)
	overlap int // different dep package from the same source
}

func main() {
	u := os.Args[1]
	if u == "" {
		log.Fatal(fmt.Errorf("please provide a url"))
	}
	baseurl, err := url.Parse(u)
	if err != nil {
		log.Fatal(err)
	}
	db := &depBuilder{
		pkgs:   make(map[string]struct{}),
		hashes: make(map[string]struct{}),
	}

	walkDeps(baseurl, db)
	log.Printf("Finished walking %d dependencies, %d of which we're dups and %d overlapping.\n", db.total, db.dups, db.overlap)
	j, err := json.Marshal(db.modules)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", string(j[:]))
}

func walkDeps(u *url.URL, db *depBuilder) {
	// request and parse
	resp, err := http.Get(u.String())
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	deps := scrape.FindAll(root, matchDeps(dependancy))
	for _, dep := range deps {
		db.total++
		relUrl, err := url.Parse(scrape.Attr(dep, "href"))
		if err != nil {
			log.Fatal(err)
		}

		resolvedUrl := u.ResolveReference(relUrl)
		if err != nil {
			log.Fatal(err)
		}

		pkg := scrape.Text(dep)
		_, dup := db.pkgs[pkg]
		if dup {
			db.dups++
		} else {
			db.pkgs[pkg] = struct{}{}
			walkDeps(resolvedUrl, db)

			var orig, s256 string
			// Get the link to the ".orig." tarball.
			tb := scrape.FindAll(root, matchTarball(".orig.tar"))
			for _, t := range tb {
				orig = scrape.Attr(t, "href")
			}
			// Get the description file and extract the sha256 hash.
			dsc := scrape.FindAll(root, matchTarball(".dsc"))
			for _, t := range dsc {
				url := scrape.Attr(t, "href")
				s256 = grabSha256fromDesc(url)
			}
			if _, ok := db.hashes[s256]; !ok {
				// Add to slice
				db.modules = append(db.modules, FlatpakModule{
					Name:       pkg,
					ConfigOpts: "",
					Srcs: []FlatpakSource{FlatpakSource{
						Type:   "archive",
						Url:    orig,
						Sha256: s256,
					}},
				})
				db.hashes[s256] = struct{}{}
			} else {
				db.overlap++
			}
		}
	}
}

// matches dependencies, required, or suggested
func matchDeps(cls string) scrape.Matcher {
	return func(n *html.Node) bool {
		if n.DataAtom == atom.A &&
			n.Parent != nil &&
			n.Parent.Parent != nil &&
			n.Parent.Parent.Parent != nil &&
			n.Parent.Parent.Parent.Parent != nil &&
			n.Parent.Parent.Parent.Parent.DataAtom == atom.Ul {
			return scrape.Attr(n.Parent.Parent.Parent.Parent, "class") == cls
		}
		return false
	}
}

func matchTarball(substr string) scrape.Matcher {
	return func(n *html.Node) bool {
		if n.DataAtom == atom.A {
			return strings.Contains(scrape.Attr(n, "href"), substr)
		}
		return false
	}
}

func grabSha256fromDesc(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	r := regexp.MustCompile("([A-Fa-f0 -9]{64}) .*orig*")
	sha256Match := r.FindStringSubmatch(buf.String())
	if len(sha256Match) == 2 {
		return sha256Match[1]
	}
	return "NONE"
}
