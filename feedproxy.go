package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html"
	"github.com/andybalholm/cascadia"
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

		for index, oldItem := range oldFeed.Items {
			newItem := convertItem(oldItem)
			newFeed.Items[index] = newItem
			go func(item *feeds.Item) {
				modifyItem(item)
				progressChan <- 1
			}(newItem)
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
	resp, err := http.Get(item.Link.Href)
	if err != nil {
		return // do nothing; leave feed item unchanged
	}
	defer resp.Body.Close()

	dilbertPage, err := html.Parse(resp.Body)
	if err != nil {
		return // do nothing; leave feed item unchanged
	}

	comicName := cascadia.MustCompile("span.comic-title-name").
			MatchFirst(dilbertPage).FirstChild.Data
	item.Title += " - " + comicName

	comicImageAttributes := cascadia.MustCompile("img.img-comic").
			MatchFirst(dilbertPage).Attr
	comicImage := ""
	for _, attribute := range comicImageAttributes {
		if attribute.Key == "src" {
			comicImage = attribute.Val
			break
		}
	}
	item.Content = fmt.Sprintf("<img width=\"900\" height=\"280\" alt=\"%s - Dilbert by Scott Adams\" src=\"%s\">", comicName, comicImage)
}

func processGamercatItem(item *feeds.Item) {
	item.Content = strings.Replace(item.Content, "-200x150", "", 1)
}

func getCommitstrip() (string, error) {
	resp, err := http.Get("http://www.commitstrip.com/en/feed/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.Split(string(body), "</rss>")[0] + "</rss>", nil
}

func getRuthe() (string, error) {
	//resp, err := http.Get("https://ruthe.de/archiv/3276/datum/asc/")
	resp, err := http.Get("https://ruthe.de/archiv/0/datum/asc/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	archivePage, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	comicItems := cascadia.MustCompile("#archiv_inner li").MatchAll(archivePage)

	img_query := cascadia.MustCompile("img")

	// TODO build base feed

	for _, x := range comicItems {
		imageUrlSmall := img_query.MatchFirst(x).Attr[0].Val
		imageUrl := strings.Replace(imageUrlSmall, "tn_", "", 1)
		dateRaw := strings.Trim(strings.Split(x.LastChild.Data, "eingestellt: ")[1], " ")

		// TODO add items
		fmt.Println(imageUrl, dateRaw)
	}

	// TODO return feed
	return "", nil
}

