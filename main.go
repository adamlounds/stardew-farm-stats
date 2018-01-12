package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

type statistics struct {
	NumLinesReceived int      `json:"numLines"`
	NumWordsReceived int      `json:"count"`
	Top5Words        []string `json:"top_5_words"`
	Top5Letters      []string `json:"top_5_letters"`
}

const (
	defaultHTTPport   = ":8080"
	defaultTelnetPort = ":3333"
)

var serverCtx context.Context

func main() {
	//log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	log.Infof("Starting Innocuous server %s", "v1.0")

	queue := make(chan string)
	statsQueue := make(chan svStats)

	go func() {
		for stats := range statsQueue {
			log.Infof("processing stats %v", stats)
		}
	}()

	go func() {
		for farmID := range queue {
			processFarmID(farmID, statsQueue)
		}
	}()

	go func() {
		for {
			fetchRecents(queue)
			time.Sleep(30 * time.Second)
			fetchRecents(queue)
			time.Sleep(30 * time.Second)
			break
		}
	}()

	select {}

}

func fetchRecents(queue chan string) {
	body, err := fetchURL("https://upload.farm/_mini_recents")
	if err != nil {
		return
	}

	farmIDs, err := extractFarmIDs(body)
	if err != nil {
		return
	}

	for _, farmID := range farmIDs {
		queue <- farmID
	}
}

func extractFarmIDs(body []byte) ([]string, error) {

	var entries []string
	err := json.Unmarshal(body, &entries)
	if err != nil {
		log.Infof("cannot parse json %s (%v)", body, err)
		return nil, err
	}

	var farmIDs []string
	for _, entry := range entries {
		farmID, err := extractFarmID(entry)
		if err != nil {
			log.Infof("unexpected entry [%s]", entry)
			continue
		}
		log.Infof("%s -> %s (%v)", entry, farmID, err)
		farmIDs = append(farmIDs, farmID)
	}

	return farmIDs, nil
}

func fetchURL(url string) ([]byte, error) {
	startTime := time.Now()
	res, err := http.Get(url)
	dur := time.Since(startTime)
	log.Infof("fetched %s in %v", url, dur)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, nil
}

type svStats struct {
	FarmID                           string
	Abigail, Alex, Caroline, Clint   uint8
	Demetrius, Dwarf, Elliott, Emily uint8
	Evelyn, George, Gus, Haley       uint8
	Harvey, Henchman, Jas, Jodi      uint8
	Kent, Krobus, Leah, Lewis        uint8
	Linus, Marnie, Maru, Pam         uint8
	Penny, Pierre, Robin, Sam        uint8
	Sandy, Sebastian, Shane, Vincent uint8
	Willy, Wizard                    uint8
}

func processFarmID(farmID string, statsQueue chan svStats) {
	log.Debugf("processing farmID %s", farmID)

	u, _ := url.Parse("https://upload.farm")
	u.Path = path.Join(u.Path, farmID)

	body, err := fetchURL(u.String())
	if err != nil {
		return
	}

	re := regexp.MustCompile("><br>([A-Z][a-z]+): ([0-9]+)/10'>")
	result := re.FindAllStringSubmatch(string(body), -1)
	if result == nil {
		return
	}

	stats := svStats{FarmID: farmID}
	v := reflect.ValueOf(&stats).Elem()

	for _, match := range result {
		name := match[1]
		rating := match[2]
		i, err := strconv.ParseUint(rating, 10, 8)
		if err != nil {
			log.Debugf("cannot parse %s's score: %s/10", name, rating)
			continue
		}

		f := v.FieldByName(name)
		if !f.IsValid() {
			log.Warnf("%s CANNOT BE SET\n", name)
			continue
		}

		f.SetUint(i)
	}

	statsQueue <- stats
}

func extractFarmID(miniRecent string) (string, error) {
	if len(miniRecent) < 7 {
		return "", fmt.Errorf("invalid miniRecent, must be at least 7 chars long")
	}
	id := miniRecent[:6]
	return id, nil
}
