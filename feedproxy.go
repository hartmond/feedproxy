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

func getModifyFeedHandler(feedURL string, modifyItem func(*feeds.Item)) func() (string, error) {
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
		progressChan := make(chan(int))
		modifyItemFunc := func(item *feeds.Item) {
			modifyItem(item)
			progressChan <- 1
		}

		for index, oldItem := range oldFeed.Items {
			newItem := convertItem(oldItem)
			newFeed.Items[index] = newItem
			go modifyItemFunc(newItem)
		}

		for i := 0;  i<len(oldFeed.Items); i++ {
			<-progressChan
		}

		feedString, err := newFeed.ToRss()
		if err != nil {
			return "", nil
		}
		return feedString, nil
	}
}

func convertItem(oldItem *gofeed.Item) (newItem *feeds.Item) {
	newItem = &feeds.Item{
		Title:   oldItem.Title,
		Content: oldItem.Content,
		Link:    &feeds.Link{Href: oldItem.Link},
		Id:      oldItem.GUID,
	}

	if oldItem.Author != nil {
		newItem.Author = &feeds.Author{
			Name: oldItem.Author.Name,
			Email: oldItem.Author.Name,
		}
	}

	if oldItem.PublishedParsed != nil {
		newItem.Created = *oldItem.PublishedParsed
	}

	return
}

func processDilbertItem(item *feeds.Item) {
	// TODO request item.Link.Href
	// TODO parse title and image url
	// TODO set Title and Content
	fmt.Println(item.Link.Href)
}

func processGamercatItem(item *feeds.Item) {
	item.Content = strings.Replace(item.Content, "-200x150", "", 1)
}

func getCommitstrip() (string, error) {
	feed, err := getUrlAsString("http://www.commitstrip.com/en/feed/")
	if err != nil {
		return "", err
	}
	return strings.Split(feed, "</rss>")[0] + "</rss>", nil
}

func getUrlAsString(url string) (string, error) {
	resp, err := http.Get(url)
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
