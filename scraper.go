package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	dependancy string = "uldep"
	debPath    string = "https://packages.debian.org/%s/%s"
)

var (
	pkgName *string = flag.String("pkg", "", "debian package to be flatpaked")
	pkgType *string = flag.String("type", "deb", "module type: valid types are 'deb' & 'tarball'")
	debVer  *string = flag.String("deb-version", "jessie", "debian code name to use")
	arch    *string = flag.String("arch", "amd64", "architecture of packages to download")
	mirror  *string = flag.String("mirror", "ftp.us.debian.org/debian", "mirror to use for downloading .deb packages")
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
	modules []FlatpakModule
	modType string
	total   int // total packages visited
	dups    int // dependency already picked up from other package(s)
}

func main() {
	flag.Parse()
	u := fmt.Sprintf(debPath, *debVer, *pkgName)
	baseurl, err := url.Parse(u)
	if err != nil {
		log.Fatal(err)
	}
	db := &depBuilder{
		pkgs:    make(map[string]struct{}),
		modType: pkgTypeStr(*pkgType),
	}

	walkDeps(baseurl, db, *pkgName)
	log.Printf("Finished walking %d dependencies, %d of which were dups.\n", db.total, db.dups)
	j, err := json.Marshal(db.modules)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", string(j[:]))
}

func walkDeps(u *url.URL, db *depBuilder, pkg string) {
	_, dup := db.pkgs[pkg]
	if dup {
		db.dups++
		return
	}

	db.pkgs[pkg] = struct{}{}
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

	var debURL string
	if *pkgType == "deb" {
		pkgMirs := scrape.FindAll(root, matchDebPkg(*arch))
		for _, m := range pkgMirs {
			debURL = scrape.Attr(m, "href")
		}
	}

	deps := scrape.FindAll(root, matchDeps(dependancy))
	for _, dep := range deps {
		db.total++
		relURL, err := url.Parse(scrape.Attr(dep, "href"))
		if err != nil {
			log.Fatal(err)
		}

		resolvedURL := u.ResolveReference(relURL)
		if err != nil {
			log.Fatal(err)
		}

		dPkg := scrape.Text(dep)
		walkDeps(resolvedURL, db, dPkg)
	}

	var pkgURL, s256 string
	if *pkgType == "tarball" {
		pkgURL, s256 = getOrigTarInfo(root)
	} else {
		pkgURL, s256 = getDebianPkgInfo(*u, debURL, *arch, *mirror)
	}
	addDep(db, pkg, pkgURL, s256)
}

func pkgTypeStr(pt string) string {
	t := "file"
	if pt == "tarball" {
		t = "archive"
	}
	return t
}

func getOrigTarInfo(root *html.Node) (string, string) {
	var orig, s256 string
	// Get the link to the ".orig." tarball.
	tb := scrape.FindAll(root, matchTarball(".orig.tar"))
	for _, t := range tb {
		orig = scrape.Attr(t, "href")
	}
	// extract the sha256 hash.
	dsc := scrape.FindAll(root, matchTarball(".dsc"))
	for _, t := range dsc {
		url := scrape.Attr(t, "href")
		s256 = grabSha256fromDesc(url)
	}
	return orig, s256
}
func getDebianPkgInfo(u url.URL, debURL, arch, mirror string) (string, string) {
	rURL, err := url.Parse(debURL)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.Get(u.ResolveReference(rURL).String())
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	dlRoot, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var url, s256 string
	// Get the description file and extract the sha256 hash.
	mirrors := scrape.FindAll(dlRoot, matchMirror(mirror))
	for _, m := range mirrors {
		url = scrape.Attr(m, "href")
	}
	// Get the description file and extract the sha256 hash.
	sha := scrape.FindAll(dlRoot, matchDebPkgSha256())
	for _, s := range sha {
		s256 = scrape.Text(s)
	}
	return url, s256
}

func addDep(db *depBuilder, pkg, url, s256 string) {
	db.modules = append(db.modules, FlatpakModule{
		Name:       pkg,
		ConfigOpts: "",
		Srcs: []FlatpakSource{FlatpakSource{
			Type:   db.modType,
			Url:    url,
			Sha256: s256,
		}},
	})
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

func matchDebPkg(arch string) scrape.Matcher {
	return func(n *html.Node) bool {
		if n.DataAtom == atom.A &&
			n.Parent.DataAtom == atom.Th {
			return strings.Contains(scrape.Text(n), arch)
		}
		return false
	}
}

func matchMirror(mirror string) scrape.Matcher {
	return func(n *html.Node) bool {
		if n.DataAtom == atom.A &&
			n.Parent != nil &&
			n.Parent.Parent != nil &&
			n.Parent.Parent.Parent != nil &&
			n.Parent.Parent.Parent.Parent != nil &&
			n.Parent.Parent.Parent.Parent.DataAtom == atom.Div {
			return scrape.Text(n) == mirror &&
				strings.Contains(scrape.Attr(n.Parent.Parent.Parent, "class"), "card")
		}
		return false
	}
}

func matchDebPkgSha256() scrape.Matcher {
	return func(n *html.Node) bool {
		if n.DataAtom == atom.Tt &&
			n.Parent != nil &&
			n.Parent.Parent.FirstChild != nil {
			return scrape.Text(n.Parent.Parent.FirstChild) == "SHA256 checksum"
		}
		return false
	}
}
