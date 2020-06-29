package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"github.com/antchfx/htmlquery"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func getURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error downloading url: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error during readall operation for url: %v", err)
	}
	return string(body), nil
}

func getURLExtended(url string, method string, content string, hdr map[string]string) (string, error) {
	if (method != "GET") && (method != "POST") {
		return "", fmt.Errorf("unsupported HTTP method [%s]", method)
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(content)))
	if err != nil {
		return "", fmt.Errorf("cannot init HTTP Request to %s with error: %v", url, err)
	}

	// Fill headers
	for k, v := range hdr {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error during HTTP request to page [%s] with error: %v", url, err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), nil
}

func getEntryURLs(baseurl string) (list []string, err error) {
	var count int
	for {
		data, err := getURLExtended(baseurl, "GET", "", map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0"})
		if err != nil {
			if count < 1 {
				fmt.Println("Cannot download first url:", baseurl)
				return list, err
			}
			return list, err
		}
		r := bytes.NewReader([]byte(data))
		doc, err := htmlquery.Parse(r)
		if err != nil {
			if count < 1 {
				fmt.Println("Cannot load HTML for first url:", baseurl)
				return list, err
			}
			// Don't treat as error if we cannot download 2nd or more url
			return list, nil
		}

		nodes, err := htmlquery.QueryAll(doc, "//li/a/@href")
		if (err != nil) || (nodes == nil) {
			if count < 1 {
				fmt.Println("Cannot parse first url:", baseurl)
				return list, err
			}
		}
		for _, node := range nodes {
			list = append(list, htmlquery.InnerText(node))
			// fmt.Println(htmlquery.InnerText(node))
		}

		count++

		node, err := htmlquery.Query(doc, "//button[@href]/@href")
		if (err == nil) && (node != nil) {
			baseurl = htmlquery.InnerText(node)
			//fmt.Println("Next page: ", htmlquery.InnerText(node))
			continue
		}
		break
	}
	err = nil
	return
}

func getEntry(url string) (map[string]string, error) {
	list := make(map[string]string)

	// Try to use cache
	cacheFileName := fmt.Sprintf("%s/%x", *CacheDir, sha1.Sum([]byte(url)))
	data, err := loadCacheFile(cacheFileName)
	if err != nil {
		data, err = getURLExtended(url, "GET", "", map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0"})
		if err != nil {
			return list, fmt.Errorf("error downloading entry [%s]: %v", url, err)
		}
		saveCacheFile(cacheFileName, data)
	}

	r := bytes.NewReader([]byte(data))
	doc, err := htmlquery.Parse(r)
	if err != nil {
		return list, fmt.Errorf("error parsing entry [%s]: %v", url, err)
	}

	// Id
	list["Id"] = ""

	// Name
	if node, err := htmlquery.Query(doc, "//h1"); (err == nil) && (node != nil) {
		list["Name"] = htmlquery.InnerText(node)
	}

	// DateOfCase<br>PlaseOfCase
	if node, err := htmlquery.Query(doc, "//p[@class=\"summary\"]"); (err == nil) && (node != nil) {
		s := strings.Split(htmlquery.OutputHTML(node, false), "<br/>")
		list["DateOfCase"] = s[0]
		// PlaceOfCase
		if len(s) > 1 {
			list["PlaceOfCase"] = s[1]
		}
	}

	if nodes, err := htmlquery.QueryAll(doc, "//div[@class=\"lightbox-content\"]/img/@src"); (err == nil) && (nodes != nil) {
		// PicUrl
		// PicBase64
		if len(nodes) > 0 {
			list["PicUrl"] = htmlquery.InnerText(nodes[0])
			_, picBase64, _ := downloadImage(*CacheDir, list["PicUrl"])
			list["PicBase64"] = picBase64
		}
		// AdditionalPicUrl
		// AdditionalPicBase64
		if len(nodes) > 1 {
			list["AdditionalPicUrl"] = htmlquery.InnerText(nodes[1])
			_, picBase64, _ := downloadImage(*CacheDir, list["AdditionalPicUrl"])
			list["AdditionalPicBase64"] = picBase64
		}
	}

	// Extra params
	if nodes, err := htmlquery.QueryAll(doc, "//table[@class=\"table table-striped wanted-person-description\"]/tbody/tr"); (err == nil) && (nodes != nil) {
		for _, node := range nodes {
			if nodes2, err := htmlquery.QueryAll(node, "//td"); (err == nil) && (nodes2 != nil) {
				if len(nodes2) == 2 {
					k := htmlquery.InnerText(nodes2[0])
					v := htmlquery.InnerText(nodes2[1])
					// fmt.Printf("%s = %s\n", k, v)

					switch k {
					// DateOfBirth
					case "Date(s) of Birth Used":
						list["DateOfBirth"] = v
						// PlaceOfBirth
					case "Place of Birth":
						list["PlaceOfBirth"] = v
					// Hair
					case "Hair":
						list["Hair"] = v
					// Eyes
					case "Eyes":
						list["Eyes"] = v
					// Height
					case "Height":
						list["Height"] = v
					// Weight
					case "Weight":
						list["Weight"] = v
					// Sex
					case "Sex":
						list["Sex"] = v
					// Race
					case "Race":
						list["Race"] = v
					// Nationality
					case "Nationality":
						list["Nationality"] = v
					}

				}
			}
		}
	}

	// Reward
	if node, err := htmlquery.Query(doc, "//div[@class=\"wanted-person-reward\"]/p"); (err == nil) && (node != nil) {
		list["Reward"] = htmlquery.InnerText(node)
	}

	// Details
	if node, err := htmlquery.Query(doc, "//div[@class=\"wanted-person-details\"]/p"); (err == nil) && (node != nil) {
		list["Details"] = htmlquery.InnerText(node)
	}

	// FieldOffice
	if node, err := htmlquery.Query(doc, "//span[@class=\"field-office\"]/p"); (err == nil) && (node != nil) {
		list["FieldOffice"] = htmlquery.InnerText(node)
	}

	// Source
	list["Source"] = url

	// Remarks
	// Related Case

	return list, nil
}

func saveCacheFile(fname string, content string) error {
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	defer f.Close()

	return err
}

func loadCacheFile(fname string) (string, error) {
	f, err := os.Open(fname)
	if err != nil {
		return "", fmt.Errorf("cannot read datafile [%s] with error: %v", fname, err)
	}
	defer f.Close()

	r, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("error during reading datafile [%s] with error: %v", fname, err)
	}
	return string(r), nil
}

func generateCSV(fname string, list []map[string]string) {
	// Create output file
	if fname == "" {
		log.Fatal("Output file is not specified, skipping result generation")
	}
	fOut, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.Fatal("Error creating output file [", *Output, "]: ", err)
	}
	csvWriter := csv.NewWriter(fOut)
	defer fOut.Close()

	//full answer
	//csvFieldList := []string{"Id", "Name", "DateOfCase", "PlaceOfCase", "PicUrl", "PicBase64", "AdditionalPicUrl",
	// "AdditionalPicBase64", "DateOfBirth", "PlaceOfBirth", "Hair", "Eyes", "Height", "Weight", "Sex", "Race", "Nationality", "Reward",
	// "Remarks", "Details", "FieldOffice", "RelatedCase", "Source"}
	//for site
	csvFieldList := []string{"Id","Name","Sex","DateOfBirth","PlaceOfBirth","Nationality","PlaceOfCase","DateOfCase","Details","Height","Hair","Eyes","Source"}
	csvWriter.Write(csvFieldList)

	for _, e := range list {
		output := []string{}
		for _, k := range csvFieldList {
			output = append(output, e[k])
		}
		csvWriter.Write(output)

		/*
			output := []string{
				strconv.FormatInt(r.MissingSince, 10), // DateOfCase
				fmt.Sprintf("%s,%s,%s", r.Country, r.State, r.City), // PlaceOfCase
				strconv.FormatInt(chld.BirthDate.int64, 10), // DateOfBirth
				fmt.Sprintf("%s %s", chld.Height, chld.HeightUnit), // Height
				fmt.Sprintf("%s %s", chld.Weight, chld.WeightUnit), // Weight
				fmt.Sprintf("%s%s", *URLCase, r.CaseId), // Source

			}
		*/
	}

}

func downloadImage(cacheDir, url string) (cacheFileName, dataBase64 string, err error) {
	// Check for cached image
	if cacheDir != "" {
		cacheFileName = fmt.Sprintf("%s/%x", cacheDir, sha1.Sum([]byte(url)))

		f, err := os.Open(cacheFileName)
		if err == nil {
			defer f.Close()
			r, err := ioutil.ReadAll(f)
			if err == nil {
				return cacheFileName, base64.StdEncoding.EncodeToString(r), nil
			}
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("error downloading image: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error during readall operation for image: %v", err)
	}

	if cacheDir != "" {
		if saveCacheFile(cacheFileName, string(body)) == nil {
			return cacheFileName, base64.StdEncoding.EncodeToString([]byte(body)), nil
		}
	}
	return "", base64.StdEncoding.EncodeToString([]byte(body)), nil
}
