# Querying Lieu

## Search Syntax

* `cat dog` - search for pages about cats or dogs, most probably both
* `fox site:example.org` - search example.org (if indexed) for the therm "fox"
* `fox -site:example.org` - search the entire index except `example.org` for the term "fox"
* `emoji lang:de` - search pages that claim to mainly contain German content for the term "emoji"
* `rank:count` - rank search results by an unweighted word count
* `rank:score` - rank search results using the usual weighted algorithm - can be used to override an URL parameter

Things that don't matter are capitalisation and inflection.
* All words in the query are converted to lowercase using the go standard library
* All words are passed through [jinzhu's inflection library](https://github.com/jinzhu/inflection) for converting them to a possible singular form (note that this is intended to work with English nouns)

## Search API

Lieu currently only renders its results to HTML. A query can be passed to the `/` endpoint using a `GET` request.

It supports two URL parameters:
* `q` - Used for the search query
* `site` - accepts one domain name and will have the same effect as the `site:<domain>` syntax. You can use this to make your webrings search engine double as a searchbox on your website.
* `rank` - behaves like the `rank:<method>` syntax, if a value is not recognised the `score` algorithm will be used

An example query to search `example.org` for the term "ssh" using `search.webring.example` should look like this: `https://search.webring.example/?q=ssh&site=example.org`

A search-form on example.org could look a bit like this:
```
<form method="GET" action="https://search.webring.example">
	<label for="search">Search example.org</label>
	<input type="search" minlength="1" required="" name="q" placeholder="Your search query here" id="search">
	<input type="hidden" name="site" value="example.org"> <!-- This hidden field tells lieu to only search example.org -->
	<button type="submit">Let's go!</button>
</form>
```
