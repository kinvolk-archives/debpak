package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	dependancy  string = "uldep"
	recommended string = "ulrec"
	suggested   string = "ulsug"
)

var (
	pkgs map[string]string = make(map[string]string)
)

func main() {
	u := os.Args[1]
	if u == "" {
		log.Fatal(fmt.Errorf("please provide a url"))
	}
	baseurl, err := url.Parse(u)
	if err != nil {
		log.Fatal(err)
	}
	chDepth := make(chan int, 5)
	fmt.Println("Starting to walk dependencies for", baseurl.String())
	walkDeps(baseurl, chDepth)
	fmt.Println("Finished walking dependencies")
}

func walkDeps(u *url.URL, cd chan int) {
	// request and parse
	resp, err := http.Get(u.String())
	if err != nil {
		panic(err)
	}
	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	deps := scrape.FindAll(root, matchDeps(dependancy))
	dups := 0
	for i, dep := range deps {
		relUrl, err := url.Parse(scrape.Attr(dep, "href"))
		if err != nil {
			log.Fatal(err)
		}

		resolvedUrl := u.ResolveReference(relUrl)
		if err != nil {
			log.Fatal(err)
		}

		pkg := scrape.Text(dep)
		if _, dup := pkgs[pkg]; dup {
			dups++
		} else {
			pkgs[pkg] = resolvedUrl.String()
			fmt.Printf("%2d %s %s\n", i, pkg, resolvedUrl.String())
			walkDeps(resolvedUrl)
		}
	}
	// Report
	fmt.Println("Deps = ", len(deps))
	fmt.Println("Dup count = ", dups)

	tb := scrape.FindAll(root, matchTarball(".orig.tar"))
	fmt.Println("Found", len(tb), "origs")
	for i, t := range tb {
		fmt.Printf("%2d %s\n", i, scrape.Attr(t, "href"))
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
