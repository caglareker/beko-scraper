package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type product struct {
	Sku           string   `json:"sku"`
	MarketingCode string   `json:"marketingCode"`
	ComputedTitle string   `json:"computedTitle"`
	Url           string   `json:"url"`
	ImageUrls     []string `json:"imageUrls"`
}

func main() {
	defer timeTrack(time.Now(), "Scraping")
	var countryCodeAndLang string
	fmt.Print("Enter Country and Language code. e.g. us-en\n> ")
	_, err := fmt.Scanf("%s", &countryCodeAndLang)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	sitemapUrl := fmt.Sprintf("https://www.beko.com/%s/sitemap-products.xml", strings.ToLower(countryCodeAndLang))
	productUrls := sitemap(sitemapUrl)
	crawl(productUrls, countryCodeAndLang)
}

func sitemap(sitemapUrl string) []string {
	var knownUrls []string
	c := colly.NewCollector(colly.AllowedDomains("www.beko.com"))
	fmt.Println("Scanning", sitemapUrl)
	c.OnXML("//loc", func(e *colly.XMLElement) {
		url := e.Text
		fmt.Println(url)
		if checkIfWebsiteExist(url) {
			knownUrls = append(knownUrls, url)
		}
	})
	c.Visit(sitemapUrl)
	return knownUrls
}

func crawl(urls []string, countryCodeAndLang string) {
	allProducts := make([]product, 0)
	productUrl := ""

	collector := colly.NewCollector(
		colly.AllowedDomains("www.beko.com"),
	)

	collector.OnHTML("html", func(e *colly.HTMLElement) {
		model := product{}
		model.Sku = e.ChildAttr("section.ProductInfo__root", "data-product-id")
		model.MarketingCode = e.ChildText(".socialShare .pageTitle")
		model.ComputedTitle = e.ChildText("h1.ProductInfo__title")
		model.Url = productUrl
		model.ImageUrls = e.ChildAttrs(".imgcontainer", "data-image-url")

		if len(model.Sku) != 0 {
			allProducts = append(allProducts, model)
		}
	})

	collector.OnRequest(func(request *colly.Request) {
		fmt.Println("Visiting", request.URL.String())
	})

	for _, url := range urls {
		productUrl = url
		collector.Visit(url)
	}
	//enc := json.NewEncoder(os.Stdout)
	//enc.SetIndent("", " ")
	//enc.Encode(allProducts)
	writeJSON(unique(allProducts), countryCodeAndLang)
	writeCSV(unique(allProducts), countryCodeAndLang)
}

func writeCSV(products []product, countryCodeAndLang string) {
	csvFile, err := os.Create(fmt.Sprintf("output/products_%s.csv", countryCodeAndLang))
	defer csvFile.Close()
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	w := csv.NewWriter(csvFile)
	defer w.Flush()
	// Using WriteAll
	var data [][]string
	for _, product := range products {
		row := []string{product.Sku, product.MarketingCode, product.ComputedTitle, product.Url}
		data = append(data, row)
	}
	w.WriteAll(data)
}

func writeJSON(data []product, countryCodeAndLang string) {
	file, err := json.MarshalIndent(data, "", " ")
	inJSON, _ := _UnescapeUnicodeCharactersInJSON(file)
	if err != nil {
		log.Println("Unable to create JSON file")
		return
	}
	fileName := fmt.Sprintf("output/products_%s.json", countryCodeAndLang)
	_ = ioutil.WriteFile(fileName, inJSON, 0644)
}

func _UnescapeUnicodeCharactersInJSON(_jsonRaw json.RawMessage) (json.RawMessage, error) {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(string(_jsonRaw)), `\\u`, `\u`, -1))
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

func checkIfWebsiteExist(s string) bool {
	r, e := http.Head(s)
	return e == nil && r.StatusCode == 200
}

func unique(p []product) []product {
	var unique []product
loop:
	for _, v := range p {
		for i, u := range unique {
			if v.Sku == u.Sku {
				unique[i] = v
				continue loop
			}
		}
		unique = append(unique, v)
	}
	return unique
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
