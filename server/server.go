package server

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	"html/template"
	"gomod.cblgh.org/lieu/database"
	"gomod.cblgh.org/lieu/types"
	"gomod.cblgh.org/lieu/util"
)

type RequestHandler struct {
	config types.Config
	db     *sql.DB
}

type TemplateView struct {
	SiteName string
	Data     interface{}
}

type SearchData struct {
	Query      string
	Title      string
	Site       string
	Pages      []types.PageData
	IsInternal bool
}

type IndexData struct {
	Tagline     string
	Placeholder string
}

type ListData struct {
	Title string
	URLs  []types.PageData
}

type AboutData struct {
	DomainCount  int
	WebringName  string
	LastCrawl    string
	PageCount    string
	TermCount    string
	FilteredLink string
	RingLink     string
}

var templates = template.Must(template.ParseFiles(
	"html/head.html", "html/nav.html", "html/footer.html",
	"html/about.html", "html/index.html", "html/list.html", "html/search.html", "html/webring.html"))

const useURLTitles = true

func (h RequestHandler) searchRoute(res http.ResponseWriter, req *http.Request) {
	var query string
	var domain string
	view := &TemplateView{}

	var domains = []string{}
	var nodomains = []string{}
	var langs = []string{}
	var queryFields = []string{}
		
	if req.Method == http.MethodGet{
		params := req.URL.Query()
		if words, exists := params["q"]; exists && words[0] != "" {
			query = words[0]
			queryFields = strings.Fields(query)
		}

		// how to use: https://gist.github.com/cblgh/29991ba0a9e65cccbe14f4afd7c975f1
		if parts, exists := params["site"]; exists && parts[0] != "" {
			// make sure we only have the domain, and no protocol prefix
			domain = strings.TrimPrefix(parts[0], "https://")
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimSuffix(domain, "/")
			domains = append(domains, domain)
		}

		// don't process if there are too many fields
		if len(queryFields) <= 100 {
			var newQueryFields []string;
			for _, word := range queryFields {
				// This could be more efficient by splitting arrays, but I'm going with the more readable version for now
				if strings.HasPrefix(word, "site:") {
					domains = append(domains, strings.TrimPrefix(word, "site:"))
				} else if strings.HasPrefix(word, "-site:") {
					nodomains = append(nodomains, strings.TrimPrefix(word, "-site:"))
				} else if strings.HasPrefix(word, "lang:") {
					langs = append(langs, strings.TrimPrefix(word, "lang:"))
				} else {
					newQueryFields = append(newQueryFields, word)
				}
			}
			queryFields = newQueryFields;
		}
		
	}

	if len(queryFields) == 0 || len(queryFields) > 100 || len(query) >= 8192 {
		view.Data = IndexData{Tagline: h.config.General.Tagline, Placeholder: h.config.General.Placeholder}
		h.renderView(res, "index", view)
		return
	}

	var pages = database.SearchWords(h.db, util.Inflect(queryFields), true, domains, nodomains, langs)

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view.Data = SearchData{
		Title:      "Link Results",
		Query:      query,
		Site:       domain,
		Pages:      pages,
		IsInternal: true,
	}
	h.renderView(res, "search", view)
}

func (h RequestHandler) paragraphSearchRoute(res http.ResponseWriter, req *http.Request) {
	var query string
	var domain string
	view := &TemplateView{}

	var queryFields []string
	var domains []string
	var nodomains []string

	if req.Method == http.MethodGet {
		params := req.URL.Query()
		if words, exists := params["q"]; exists && words[0] != "" {
			query = words[0]
			queryFields = strings.Fields(query)
		}


		// how to use: https://gist.github.com/cblgh/29991ba0a9e65cccbe14f4afd7c975f1
		if parts, exists := params["site"]; exists && parts[0] != "" {
			// make sure we only have the domain, and no protocol prefix
			domain = strings.TrimPrefix(parts[0], "https://")
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimSuffix(domain, "/")
			domains = append(domains, domain)
		}

		var newQueryFields []string
		// don't process if there are too many fields
		if len(queryFields) <= 100 {
			for _, word := range queryFields {
				// This could be more efficient by splitting arrays, but I'm going with the more readable version for now
				if strings.HasPrefix(word, "site:") {
					domains = append(domains, strings.TrimPrefix(word, "site:"))
				} else if strings.HasPrefix(word, "-site:") {
					nodomains = append(nodomains, strings.TrimPrefix(word, "-site:"))
				} else {
					newQueryFields = append(newQueryFields, word)
				}
			}
			query = strings.Join(newQueryFields, " ");
		}
	}

	pages := database.FulltextSearchWholeParagraphs(h.db, query, domains, nodomains)

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view.Data = SearchData{
		Title:      "Paragraph Search Results",
		Site:       domain,
		Query:      strings.Join(queryFields, " "),
		Pages:      pages,
		IsInternal: false,
	}
	h.renderView(res, "search", view)
}

func (h RequestHandler) externalSearchRoute(res http.ResponseWriter, req *http.Request) {
	var query string
	view := &TemplateView{}

	if req.Method == http.MethodGet {
		params := req.URL.Query()
		if words, exists := params["q"]; exists && words[0] != "" {
			query = words[0]
		}
	}

	pages := database.FulltextSearchWords(h.db, query)

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view.Data = SearchData{
		Title:      "External Results",
		Query:      query,
		Pages:      pages,
		IsInternal: false,
	}
	h.renderView(res, "search", view)
}

func (h RequestHandler) aboutRoute(res http.ResponseWriter, req *http.Request) {
	view := &TemplateView{}

	pageCount := util.Humanize(database.GetPageCount(h.db))
	wordCount := util.Humanize(database.GetWordCount(h.db))
	domainCount := database.GetDomainCount(h.db)
	lastCrawl := database.GetLastCrawl(h.db)

	view.Data = AboutData{
		WebringName:  h.config.General.Name,
		DomainCount:  domainCount,
		PageCount:    pageCount,
		TermCount:    wordCount,
		LastCrawl:    lastCrawl,
		FilteredLink: "/filtered",
		RingLink:     h.config.General.URL,
	}
	h.renderView(res, "about", view)
}

func (h RequestHandler) filteredRoute(res http.ResponseWriter, req *http.Request) {
	view := &TemplateView{}

	var URLs []types.PageData
	for _, domain := range util.ReadList(h.config.Crawler.BannedDomains, "\n") {
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
	h.renderView(res, "list", view)
}

func (h RequestHandler) randomRoute(res http.ResponseWriter, req *http.Request) {
	link := database.GetRandomPage(h.db)
	http.Redirect(res, req, link, http.StatusSeeOther)
}

func (h RequestHandler) randomExternalRoute(res http.ResponseWriter, req *http.Request) {
	link := database.GetRandomExternalLink(h.db)
	http.Redirect(res, req, link, http.StatusSeeOther)
}

func (h RequestHandler) webringRoute(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, h.config.General.URL, http.StatusSeeOther)
}

func (h RequestHandler) renderView(res http.ResponseWriter, tmpl string, view *TemplateView) {
	view.SiteName = h.config.General.Name
	var errTemp error
	if _, exists := os.LookupEnv("LIEU_DEV"); exists {
		var templates = template.Must(template.ParseFiles(
			"html/head.html", "html/nav.html", "html/footer.html",
			"html/about.html", "html/index.html", "html/list.html", "html/search.html", "html/webring.html"))
		errTemp = templates.ExecuteTemplate(res, tmpl+".html", view)
	} else {
		errTemp = templates.ExecuteTemplate(res, tmpl+".html", view)
	}
	if errors.Is(errTemp, syscall.EPIPE) {
		fmt.Println("had a broken pipe, continuing")
	} else {
		util.Check(errTemp)
	}
}

func WriteTheme(config types.Config) {
	theme := config.Theme
	// no theme is set, use the default
	if theme.Foreground == "" || theme.Background == "" || theme.Links =="" {
		return
	}
	colors := fmt.Sprintf(`/*This file will be automatically regenerated by lieu on startup if the theme colors are set in the configuration file*/
:root {
  --primary: %s;
  --secondary: %s;
  --link: %s;
}`, theme.Foreground, theme.Background, theme.Links)
	err := os.WriteFile("html/assets/theme.css", []byte(colors), 0644)
	util.Check(err)
}

func Serve(config types.Config) {
	WriteTheme(config)
	db := database.InitDB(config.Data.Database)
	handler := RequestHandler{config: config, db: db}

	http.HandleFunc("/about", handler.aboutRoute)
	http.HandleFunc("/", handler.searchRoute)
	http.HandleFunc("/paragraph", handler.paragraphSearchRoute)
	http.HandleFunc("/outgoing", handler.externalSearchRoute)
	http.HandleFunc("/random/outgoing", handler.randomExternalRoute)
	http.HandleFunc("/random", handler.randomRoute)
	http.HandleFunc("/webring", handler.webringRoute)
	http.HandleFunc("/filtered", handler.filteredRoute)

	fileserver := http.FileServer(http.Dir("html/"))
	http.Handle("/assets/", fileserver)
	http.Handle("/robots.txt", fileserver)

	portstr := fmt.Sprintf(":%d", config.General.Port)
	fmt.Println("Listening on port: ", portstr)

	http.ListenAndServe(portstr, nil)
}
