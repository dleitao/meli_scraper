package main

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gocolly/colly"
)

var sellers map[string]seller
var m sync.Mutex
var countP int64
var wg sync.WaitGroup
var wgP sync.WaitGroup

type seller struct {
	id    string
	count int
}

type sellerList []seller

func (e sellerList) Len() int {
	return len(e)
}

func (e sellerList) Less(i, j int) bool {
	return e[i].count > e[j].count
}

func (e sellerList) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func scrapSearchResults(url string, searchLinks *chan string, productsLinks *chan string) {
	if url == "" {
		log.Println("missing url argument")
		return
	}

	fmt.Println("visiting", url)

	cls := true
	c := colly.NewCollector()

	c.OnHTML(".andes-pagination__button--next", func(e *colly.HTMLElement) {
		link := e.ChildAttr(".andes-pagination__link", "href")
		*searchLinks <- link
		cls = false
	})
	err := c.Visit(url)
	if err != nil {
		fmt.Println("err", err)
	}
	if cls {
		close(*searchLinks)
		wg.Done()
		return
	}

	c1 := colly.NewCollector()
	c1.OnHTML(".ui-search-result__content-wrapper", func(e *colly.HTMLElement) {
		*productsLinks <- e.ChildAttr("a", "href")
	})
	c1.Visit(url)

}

func scrapProduct(url string) {
	atomic.AddInt64(&countP, 1)
	defer wgP.Done()

	if url == "" {
		log.Println("missing url argument")
		return
	}

	c := colly.NewCollector()
	c.OnHTML(".layout-col--right", func(e *colly.HTMLElement) {
		re := regexp.MustCompile(`\d+`)
		itemCount, _ := strconv.Atoi(string(re.Find([]byte(e.ChildText(".item-conditions")))))
		re1 := regexp.MustCompile("https://perfil.mercadolibre.com.co/")
		id := re1.ReplaceAllLiteralString(e.ChildAttr("#seller-view-more-link", "href"), "")

		m.Lock()
		_, exists := sellers[id]
		if exists {
			sellers[id] = seller{id, sellers[id].count + itemCount}
		} else {
			sellers[id] = seller{id, itemCount}
		}
		m.Unlock()
	})
	c.Visit(url)

}

func initScrapSearchResults(searchLinks *chan string, productsLinks *chan string) {
	for url := range *searchLinks {
		go scrapSearchResults(url, searchLinks, productsLinks)
	}
}

func initScrapProduct(productsLinks *chan string) {
	for url := range *productsLinks {
		wgP.Add(1)
		go scrapProduct(url)

	}
}
func getSortedSeller() []seller {

	values := []seller{}
	for _, value := range sellers {
		values = append(values, value)
	}
	list := sellerList(values)
	sort.Sort(list)
	if len(list) < 200 {
		return list
	}
	res := list[0:199]
	return res
}

func main() {
	fmt.Println("let's start")

	productsLinks := make(chan string, 100)
	searchLinks := make(chan string, 10)
	sellers = make(map[string]seller)
	countP = 0

	searchLinks <- "https://celulares.mercadolibre.com.co/_BestSellers_YES"
	// searchLinks <- "https://celulares.mercadolibre.com.co/_Desde_1901_BestSellers_YES"

	wg.Add(1)

	go initScrapSearchResults(&searchLinks, &productsLinks)
	go initScrapProduct(&productsLinks)

	wg.Wait()
	wgP.Wait()

	fmt.Println(getSortedSeller())
	fmt.Println("#Products", countP)
}
