package main

import (
	"encoding/json"
	"fmt"
	"github.com/firstrow/tcp_server"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

var allFarms map[string]svStats

var serverCtx context.Context

func main() {
	//log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	log.Infof("Starting Innocuous server %s %d", "v1.0", runtime.GOMAXPROCS(0))

	allFarms = make(map[string]svStats)

	queue := make(chan string, 100)
	statsQueue := make(chan svStats, 100)

	go func() {
		for stats := range statsQueue {
			log.Infof("processing stats %v", stats)
			allFarms[stats.FarmID] = stats
			log.Infof("processed stats %v", len(allFarms))
		}
	}()

	farmIDProcessor := func() {
		fmt.Printf("processing farm ids[%d]\n", len(queue))
		for farmID := range queue {
			processFarmID(farmID, statsQueue)
		}
	}
	go farmIDProcessor()
	go farmIDProcessor()

	go func() {
		telnetSvr := tcp_server.New("localhost" + defaultTelnetPort)
		telnetSvr.OnNewClient(func(c *tcp_server.Client) {
			c.Send("welcome\n")
		})
		telnetSvr.OnNewMessage(func(c *tcp_server.Client, message string) {
			message = strings.TrimRight(message, "\r\n")
			if message[0] == '/' {
				switch {
				case message == "/fetch":
					fetchRecents(queue)
					c.Send("fetched recent farms\n")
				case message == "/qsize":
					c.Send(fmt.Sprintf("queue size is [%d]\n", len(queue)))
				case message == "/show":
					c.Send(fmt.Sprintf("stats:\n"))
					for _, stats := range allFarms {
						c.Send(fmt.Sprintf("%s likes Abigail %d/10\n", stats.FarmID, stats.Abigail ))
					}
				case message == "/spider":
					fetchMany(queue)
					c.Send("simple spider\n")
				case strings.HasPrefix(message, "/spider "):
					// TODO parse & send page number
					fetchMany(queue)
					c.Send("complex spider\n")
				default:
					c.Send(fmt.Sprintf("unknown command [%s] \n", message))
				}
				return
			}

			farmID, err := extractFarmID(message)
			if err == nil {
				queue <- farmID
				c.Send(fmt.Sprintf("queued farm id %s\n", farmID))
				return
			}
			c.Send("invalid farm id\n")
			return
		})
		telnetSvr.Listen()
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
		fmt.Printf("queueing farmID [%s] [%d]", farmID, len(queue))
		queue <- farmID
		fmt.Printf("queued farmID [%s] [%d]", farmID, len(queue))
	}
}

func fetchMany(queue chan string) {
	body, err := fetchURL("https://upload.farm/all?p=4695&sort=recent")
	if err != nil {
		return
	}

	farmIDs, err := farmIDsFromSearch(body)
	if err != nil {
		fmt.Printf("could not find farmIDs: %s\n", err)
		return
	}

	for _, farmID := range farmIDs {
		fmt.Printf("queueing farmID [%s] [%d]", farmID, len(queue))
		queue <- farmID
		fmt.Printf("queued farmID [%s] [%d]", farmID, len(queue))
	}
}

func farmIDsFromSearch(body []byte) ([]string, error) {
	re := regexp.MustCompile("/([A-Za-z0-9]{6})-f.png")
	result := re.FindAllStringSubmatch(string(body), -1)
	if result == nil {
		return nil, fmt.Errorf("no farms found")
	}

	var farmIDs []string
	for _, match := range result {
		farmID := match[1]
		farmIDs = append(farmIDs, farmID)
	}

	return farmIDs, nil
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

func processFarmID(farmID string, statsQueue chan svStats) {
	log.Debugf("processing farmID %s", farmID)

	if _, ok := allFarms[farmID]; ok {
		log.Debugf("skipping %s - already processed", farmID)
		return
	}

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
	if len(miniRecent) < 6 {
		return "", fmt.Errorf("invalid farmID, must be at least 6 chars long")
	}
	if (miniRecent[0] != '1') {
		return "", fmt.Errorf("invalid FarmID; should start with 1")
	}
	id := miniRecent[:6]
	return id, nil
}
