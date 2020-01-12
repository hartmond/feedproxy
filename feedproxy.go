package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
)

var feedDict = map[string]func() (string, error){
	"dilbert":     getModifyFeedHandler("http://dilbert.com/feed", processDilbertItem),
	"gamercat":    getModifyFeedHandler("http://www.thegamercat.com/feed/", processGamercatItem),
	"ruthe":       getRuthe,
	"commitstrip": getCommitstrip,
}

func main() {
	fmt.Println("Starting Server on localhost:8889")
	http.HandleFunc("/feeds/", processFeed)
	http.ListenAndServe("localhost:8889", nil)
}

func processFeed(w http.ResponseWriter, r *http.Request) {
	getFeedFunc, ok := feedDict[r.URL.String()[7:]]
	if ok != true {
		http.NotFound(w, r)
		return
	}
	processedFeed, err := getFeedFunc()
	if err != nil {
		w.WriteHeader(412)
		return
	}
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	// w.Header().Add("Content-Type", "application/rss+xml; charset=utf-8")
	fmt.Fprint(w, processedFeed)
}

func getModifyFeedHandler(feedURL string, modifyItem func(*gofeed.Item) *feeds.Item) func() (string, error) {
	return func() (string, error) {
		oldFeed, err := gofeed.NewParser().ParseURL(feedURL)
		if err != nil {
			return "", err
		}

		newFeed := feeds.Feed{
			Title:       oldFeed.Title,
			Link:        &feeds.Link{Href: oldFeed.Link},
			Description: oldFeed.Description,
			// TODO Author:      &feeds.Author{Name: oldFeed.Author.Name, Email: oldFeed.Author.Email},
			// TODO updated date
			Copyright: oldFeed.Copyright,
		}

		newFeed.Items = make([]*feeds.Item, len(oldFeed.Items))
		for index, oldItem := range oldFeed.Items {
			// TODO Parallelize
			newFeed.Items[index] = modifyItem(oldItem)
		}

		feedString, err := newFeed.ToAtom()
		if err != nil {
			return "", nil
		}
		fmt.Println(feedString)
		return feedString, nil
	}
}

func processDilbertItem(*gofeed.Item) *feeds.Item {
	// TODO implement
	return &feeds.Item{
		Title: "Title",
		Id:    "Id",
		Link:  &feeds.Link{Href: "Link"},
		// Description: "Dsc",
		// Author:      &feeds.Author{Name: "Name", Email: "Email"},
		// Created:     time.Now(),
	}
}

func processGamercatItem(*gofeed.Item) *feeds.Item {
	// TODO implement
	return &feeds.Item{
		Title: "",
		Link:  &feeds.Link{Href: ""},
	}
}

func getCommitstrip() (string, error) {
	feed, err := getFeed("http://www.commitstrip.com/en/feed/")
	if err != nil {
		return "", err
	}
	return strings.Split(feed, "</rss>")[0] + "</rss>", nil
}

func getFeed(feedURL string) (string, error) {
	resp, err := http.Get(feedURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func getRuthe() (string, error) {
	// TODO implement
	return "ruthe - not implemented", nil
}
