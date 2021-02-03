package types

type SearchFragment struct {
	Word  string
	URL   string
	Score int
}

type PageData struct {
	URL   string
	Title string
	About string
	Lang  string
}

type Config struct {
	General struct {
		Name string `json:name`
		URL  string `json:url`
		Port int    `json:port`
	} `json:general`
	Data struct {
		Source     string `json:source`
		Database   string `json:database`
		Heuristics string `json:heuristics`
		Wordlist   string `json:wordlist`
	} `json:data`
	Crawler struct {
		Webring        string `json:webring`
		BannedDomains  string `json:bannedDomains`
		BannedSuffixes string `json:bannedSuffixes`
		BoringWords    string `json:boringWords`
		BoringDomains  string `json:boringDomains`
	} `json:crawler`
}
