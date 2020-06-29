package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

var (
	DataURL  = flag.String("dataurl", "https://www.fbi.gov/wanted/kidnap/@@castle.cms.querylisting/querylisting-1", "Basic data search URL")
	Output   = flag.String("output", "../output/output.csv", "Name of output resulting CSV file")
	CacheDir = flag.String("cachedir", "../output/cache", "Directory for storing image cache")
)

func main() {
	fmt.Println("Go!")
	flag.Parse()

	var urlList []string
	cacheFileName := fmt.Sprintf("%s/%s", *CacheDir, "urllist.txt")
	if data, err := loadCacheFile(cacheFileName); err == nil {
		urlList = strings.Split(data, "\n")
	} else {
		urlList, err = getEntryURLs(*DataURL)
		if err == nil {
			output := strings.Join(urlList, "\n")
			saveCacheFile(cacheFileName, output)
		}
	}
	if len(urlList) < 1 {
		log.Fatal("No entries for scan")
	}
	fmt.Printf("Total items found: %d\n", len(urlList))
	entries := new([]map[string]string)

	for _, url := range urlList {
		fmt.Println("Downloading: ", url)
		entry, err := getEntry(url)
		if err == nil {
			*entries = append(*entries, entry)
		}
	}
	generateCSV(*Output, *entries)

	//fmt.Println(urlList)
	//	Parser("")
}
