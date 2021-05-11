package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"html/template"
	"lieu/database"
	"lieu/types"
	"lieu/util"
)

type TemplateView struct {
	SiteName string
	Data     interface{}
}

type SearchData struct {
	Query string
	Pages []types.PageData
}

type ListData struct {
	Title string
	URLs  []types.PageData
}

type AboutData struct {
	DomainCount  int
	InstanceName string
	PageCount    string
	TermCount    string
	FilteredLink string
	RingLink     string
}

const useURLTitles = true

var templates = template.Must(template.ParseFiles(
	"html/head.html", "html/nav.html", "html/footer.html",
	"html/about.html", "html/index.html", "html/list.html", "html/search.html", "html/webring.html"))

func searchRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	var query string
	view := &TemplateView{}

	if req.Method == http.MethodGet {
		params := req.URL.Query()
		words, exists := params["q"]
		if !exists || words[0] == "" {
			view.Data = SearchData{}
			renderView(res, config, "index", view)
			return
		}
		query = words[0]
	} else {
		view.Data = SearchData{}
		renderView(res, config, "index", view)
		return
	}

	pages := database.SearchWordsByScore(db, util.Inflect(strings.Fields(query)))

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view.Data = SearchData{
		Query: query,
		Pages: pages,
	}
	renderView(res, config, "search", view)
}

func aboutRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	view := &TemplateView{}

	pageCount := util.Humanize(database.GetPageCount(db))
	wordCount := util.Humanize(database.GetWordCount(db))
	domainCount := database.GetDomainCount(db)

	view.Data = AboutData{
		InstanceName: config.General.Name,
		DomainCount:  domainCount,
		PageCount:    pageCount,
		TermCount:    wordCount,
		FilteredLink: "/filtered",
		RingLink:     config.General.URL,
	}

	renderView(res, config, "about", view)
}

func filteredRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	view := &TemplateView{}
	var URLs []types.PageData
	for _, domain := range util.ReadList(config.Crawler.BannedDomains, "\n") {
		u, err := url.Parse(domain)
		if err != nil {
			continue
		}
		u.Scheme = "https"
		p := types.PageData{Title: domain, URL: u.String()}
		URLs = append(URLs, p)
	}
	view.Data = ListData{
		Title: "Filtered Domains",
		URLs:  URLs,
	}
	renderView(res, config, "list", view)
}

func randomRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	link := database.GetRandomPage(db)
	http.Redirect(res, req, link, http.StatusSeeOther)
}

func renderView(res http.ResponseWriter, config types.Config, tmpl string, view *TemplateView) {
	view.SiteName = config.General.Name
	errTemp := templates.ExecuteTemplate(res, tmpl+".html", view)
	util.Check(errTemp)
}

func Serve(config types.Config) {
	db := database.InitDB(config.Data.Database)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		searchRoute(res, req, config, db)
	})

	http.HandleFunc("/about", func(res http.ResponseWriter, req *http.Request) {
		aboutRoute(res, req, config, db)
	})

	http.HandleFunc("/random", func(res http.ResponseWriter, req *http.Request) {
		randomRoute(res, req, config, db)
	})

	http.HandleFunc("/filtered", func(res http.ResponseWriter, req *http.Request) {
		filteredRoute(res, req, config, db)
	})

	fileserver := http.FileServer(http.Dir("html/assets/"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fileserver))

	portstr := fmt.Sprintf(":%d", config.General.Port)
	fmt.Println("Listening on port: ", portstr)

	http.ListenAndServe(portstr, nil)
}
