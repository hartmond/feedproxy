package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
	"github.com/gorilla/mux"
)

var feedDict = map[string]func() (string, error){
	"dilbert":     getModifyFeedHandler("http://dilbert.com/feed", processDilbertItem),
	"gamercat":    getModifyFeedHandler("http://www.thegamercat.com/feed/", processGamercatItem),
	"ruthe":       getRuthe,
	"commitstrip": getCommitstrip,
}

func main() {
	fmt.Println("Starting Server on localhost:8889")
	r := mux.NewRouter()
	r.HandleFunc("/{base}/{feed}", processFeed)
	http.ListenAndServe("0.0.0.0:8889", r)
}

func processFeed(w http.ResponseWriter, r *http.Request) {
	getFeedFunc, ok := feedDict[mux.Vars(r)["feed"]]
	if ok != true {
		http.NotFound(w, r)
		return
	}
	processedFeed, err := getFeedFunc()
	if err != nil {
		w.WriteHeader(412)
		fmt.Fprint(w, err.Error())
		return
	}
	w.Header().Add("Content-Type", "application/rss+xml; charset=utf-8")
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
			Copyright:   oldFeed.Copyright,
		}

		if oldFeed.Author != nil {
			newFeed.Author = &feeds.Author{Name: oldFeed.Author.Name, Email: oldFeed.Author.Email}
		}

		newFeed.Items = make([]*feeds.Item, len(oldFeed.Items))
		progressChan := make(chan (int))

		for index, oldItem := range oldFeed.Items {
			newItem := convertItem(oldItem)
			newFeed.Items[index] = newItem
			go func(item *feeds.Item) {
				modifyItem(item)
				progressChan <- 1
			}(newItem)
		}

		for i := 0; i < len(oldFeed.Items); i++ {
			<-progressChan
		}

		feedString, err := newFeed.ToRss()
		if err != nil {
			return "", err
		}

		newFeed.Updated = newFeed.Items[0].Updated
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
			Name:  oldItem.Author.Name,
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
	item.Content = fmt.Sprintf("<img alt=\"%s - Dilbert by Scott Adams\" src=\"%s\">", comicName, comicImage)
}

func processGamercatItem(item *feeds.Item) {
	item.Content = strings.Replace(item.Content, "-200x150", "", 1)
}

func getCommitstrip() (string, error) {
	resp, err := http.Get("https://www.commitstrip.com/en/feed/")
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

	imgQuery := cascadia.MustCompile("img")

	rutheFeed := feeds.Feed{
		Title: "Ruthe Comics",
		Id:    "tag:ruthe.de,2005:/feed",
		Link:  &feeds.Link{Href: "http://ruthe.de"},
	}

	rutheFeed.Items = make([]*feeds.Item, len(comicItems))

	for i, x := range comicItems {
		imageURLSmall := imgQuery.MatchFirst(x).Attr[0].Val
		id := strings.Replace(imageURLSmall, "/cartoons/tn_strip_", "", 1)
		id = strings.Replace(id, ".jpg", "", 1)
		dateRaw := strings.Trim(strings.Split(x.LastChild.Data, "eingestellt: ")[1], " ")
		dateParsed, _ := time.Parse("02.01.'06", dateRaw)

		rutheFeed.Items[i] = &feeds.Item{
			Title:   fmt.Sprintf("Comic vom %s", dateParsed.Format("02.01.2006")),
			Updated: dateParsed,
			Id:      id,
			Link:    &feeds.Link{Href: fmt.Sprintf("https://ruthe.de/cartoon/%v/", id)},
			Content: fmt.Sprintf("<img alt=\"Comic\" class=\"img-responsive img-comic\" height=\"300\" src=\"https://ruthe.de/cartoons/strip_%s.jpg\">", id),
		}
	}

	rutheFeed.Updated = rutheFeed.Items[0].Updated

	feedString, err := rutheFeed.ToRss()
	if err != nil {
		return "", err
	}
	return feedString, nil
}
