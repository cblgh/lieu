# Querying Lieu

## Search Syntax

* `cat dog` - search for pages about cats or dogs, most probably both
* `fox site:example.org` - search example.org (if indexed) for term "fox"
* `fox -site:example.org` - search all indexed sites except `example.org` for term "fox"
* `emoji lang:de` - search pages that claim to mainly contain German content for the term "emoji"

When searching, capitalisation and inflection do not matter, as search terms are:

* Converted to lowercase using the go standard library
* Passed through [jinzhu's inflection library](https://github.com/jinzhu/inflection) for
  converting to a possible singular form (intended to work with English nouns)

## Search API

Lieu currently only renders its results to HTML. A query can be passed to the `/` endpoint using a `GET` request.

It supports two URL parameters:
* `q` - used for the search query
* `site` - accepts one domain name and will have the same effect as the `site:<domain>` syntax.
  You can use this to make your webrings search engine double as a searchbox on your website.

### Examples
To search `example.org` for the term "ssh" using `https://search.webring.example`:

```
https://search.webring.example/?q=ssh&site=example.org
```

Adding a form element, to use Lieu as a search engine, to the HTML at example.org:

```
<form method="GET" action="https://search.webring.example">
	<label for="search">Search example.org</label>
	<input type="search" minlength="1" required="" name="q" placeholder="Your search query here" id="search">
	<input type="hidden" name="site" value="example.org"> <!-- This hidden field tells lieu to only search example.org -->
	<button type="submit">Let's go!</button>
</form>
```
