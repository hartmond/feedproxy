package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

var feedDict = map[string]func() (string, error){
	"dilbert":        getModifyFeedHandler("http://dilbert.com/feed", processDilbertItem),
	"gamercat":       getModifyFeedHandler("http://www.thegamercat.com/feed/", processGamercatItem),
	"ruthe":          getRuthe,
	"commitstrip":    getCommitstrip,
	"nichtlustig":    getNichtlustig,
	"littlebobby":    getLittlebobby,
	"heiseonline":    getFilterFeedHandler("https://www.heise.de/rss/heise-atom.xml", false, []string{"security", "developer", "select/ix"}),
	"heisesecurity":  getFilterFeedHandler("https://www.heise.de/security/rss/news-atom.xml", true, []string{"security"}),
	"heisedeveloper": getFilterFeedHandler("https://www.heise.de/developer/rss/news-atom.xml", true, []string{"developer"}),
	"heiseix":        getFilterFeedHandler("https://www.heise.de/ix/rss/news-atom.xml", true, []string{"select/ix"}),
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

func contains(list []string, value string) bool {
	for _, x := range list {
		if strings.HasPrefix(value, x) {
			return true
		}
	}
	return false
}

func getFilterFeedHandler(feedURL string, include bool, whitelist []string) func() (string, error) {
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

		newFeed.Items = []*feeds.Item{}

		for _, oldItem := range oldFeed.Items {
			newItem := convertItem(oldItem)
			area := strings.SplitN(newItem.Link.Href, "/", 4)[3]
			if contains(whitelist, area) == include {
				newFeed.Items = append(newFeed.Items, newItem)
			}
		}

		feedString, err := newFeed.ToRss()
		if err != nil {
			return "", err
		}

		if len(newFeed.Items) > 1 {
			newFeed.Updated = newFeed.Items[0].Updated
		} else {
			time.Now()
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

type nichtlustigElement struct {
	Slug         string `json:"slug"`
	Image        string `json:"image"`
	BonusPrivate bool   `json:"bonus"`
	BonusImage   string `json:"bonus_image"`
	BonusPublic  bool   `json:"public_bonus"`
	Tags         string `json:"tags"`
	Title        string `json:"title"`
	Color        string `json:"color"`
}

func getNichtlustig() (string, error) {
	resp, err := http.Get("https://joscha.com/nichtlustig/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	data := buf.Bytes()

	dataList := bytes.SplitN(bytes.SplitN(data, []byte("var cartoonList = "), 2)[1], []byte(";</script>"), 2)[0]

	// make javascript to valid JSON...
	dataList = bytes.Replace(dataList, []byte("'"), []byte("\""), -1)
	dataList = bytes.Replace(dataList, []byte(",\t]"), []byte("]"), -1)

	var images []nichtlustigElement
	err = json.Unmarshal(dataList, &images)
	if err != nil {
		return "", err
	}

	nichtlustigFeed := feeds.Feed{
		Title: "Nicht Lustig Cartoons",
		Id:    "tag:joscha.com/nichtlustig,2005:/feed",
		Link:  &feeds.Link{Href: "https://joscha.com/nichtlustig"},
	}

	nichtlustigFeed.Items = make([]*feeds.Item, 20)

	res := ""
	for i, elem := range images[:20] {
		// TODO build feed
		res += fmt.Sprintf("%s: %s - https://joscha.com/data/media/cartoons/%s\n", elem.Slug, elem.Title, elem.Image)

		dateParsed, _ := time.Parse("060102", elem.Slug)

		var content string
		if elem.BonusPublic {
			content = fmt.Sprintf("<img alt=\"%s\" src=\"https://joscha.com/data/media/cartoons/%s\"><img alt=\"BonusCartoon for %s\" src=\"https://joscha.com/data/media/cartoons/bonus/%s\">", elem.Title, elem.Image, elem.Title, elem.BonusImage)
		} else {
			content = fmt.Sprintf("<img alt=\"%s\" src=\"https://joscha.com/data/media/cartoons/%s\">", elem.Title, elem.Image)
		}

		nichtlustigFeed.Items[i] = &feeds.Item{
			Title:   fmt.Sprintf("NichtLustig Cartoon vom %s - %s", dateParsed.Format("02.01.2006"), elem.Title),
			Updated: dateParsed,
			Id:      elem.Slug,
			Link:    &feeds.Link{Href: fmt.Sprintf("https://joscha.com/nichtlustig/%v/", elem.Slug)},
			Content: content,
		}
	}

	nichtlustigFeed.Updated = nichtlustigFeed.Items[0].Updated

	feedString, err := nichtlustigFeed.ToRss()
	if err != nil {
		return "", err
	}
	return feedString, nil

}

func getLittlebobby() (string, error) {
	resp, err := http.Get("https://www.littlebobbycomic.com/archive/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	archivePage, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	feed := feeds.Feed{
		Title: "Little Bobby",
		Id:    "tag:littlebobbycomic.de,2005:/feed",
		Link:  &feeds.Link{Href: "https://www.littlebobbycomic.com"},
	}

	comicItems := cascadia.MustCompile("div.project-img-wrap a").MatchAll(archivePage)

	feed.Items = make([]*feeds.Item, len(comicItems))

	for i, x := range comicItems {
		var link string
		for _, attr := range x.Attr {
			if attr.Key == "href" {
				link = attr.Val
				break
			}
		}
		week := x.FirstChild.NextSibling.FirstChild.Data
		date, err := time.Parse("January 2, 2006", x.LastChild.FirstChild.Data)
		if err != nil {
			fmt.Println(err)
		}
		var image string
		for _, attr := range x.FirstChild.Attr {
			if attr.Key == "src" {
				image = strings.Replace(attr.Val, "-480x270", "", 1)
				break
			}
		}

		feed.Items[i] = &feeds.Item{
			Title:   fmt.Sprintf("LittleBobbyComic for %s (%s)", week, date.Format("02.01.2006")),
			Updated: date,
			Id:      week,
			Link:    &feeds.Link{Href: link},
			Content: fmt.Sprintf("<img alt=\"Comic\" height=\"300\" src=\"%s\">", image),
		}
	}

	feed.Updated = feed.Items[0].Updated

	feedString, err := feed.ToRss()
	if err != nil {
		return "", err
	}
	return feedString, nil
}
