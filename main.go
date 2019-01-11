package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/adamlounds/stardew-farm-stats/farmstats"
	"github.com/firstrow/tcp_server"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/render"
	resque "github.com/kavu/go-resque"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type statistics struct {
	NumLinesReceived int      `json:"numLines"`
	NumWordsReceived int      `json:"count"`
	Top5Words        []string `json:"top_5_words"`
	Top5Letters      []string `json:"top_5_letters"`
}

const (
	defaultHTTPport   = ":8080"
	defaultTelnetPort = "3333"
)

type svStats struct {
	FarmID                           string
	Abigail, Alex, Caroline, Clint   uint32
	Demetrius, Dwarf, Elliott, Emily uint32
	Evelyn, George, Gus, Haley       uint32
	Harvey, Henchman, Jas, Jodi      uint32
	Kent, Krobus, Leah, Lewis        uint32
	Linus, Marnie, Maru, Pam         uint32
	Penny, Pierre, Robin, Sam        uint32
	Sandy, Sebastian, Shane, Vincent uint32
	Willy, Wizard                    uint32
}

type farmStats struct {
	mu    sync.Mutex
	stats map[string]svStats
}

var allFarms farmStats

var serverCtx context.Context

func main() {
	//log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	log.Infof("Starting Innocuous server %s %d", "v1.0", runtime.GOMAXPROCS(0))

	allFarms.stats = make(map[string]svStats)

	queue := make(chan string, 100)
	statsQueue := make(chan svStats, 100)

	go func() {
		for stats := range statsQueue {
			log.Debugf("processing stats %v", stats)
			allFarms.mu.Lock()
			allFarms.stats[stats.FarmID] = stats
			log.Debugf("processed stats %v", len(allFarms.stats))
			allFarms.mu.Unlock()
		}
	}()

	farmIDProcessor := func() {
		log.Debugf("processing farm ids[%d]\n", len(queue))
		for farmID := range queue {
			processFarmID(farmID, statsQueue)
		}
	}
	go farmIDProcessor()
	go farmIDProcessor()
	go telnetServer(defaultTelnetPort, queue)
	go httpServer()
	go grpcServer()

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

func enqueueRedis(enqueuer *resque.RedisEnqueuer, farmID string) error {
	_, err := enqueuer.Enqueue("farm", "Process::Farm", farmID)
	return err
}

func grpcServer() {
	lis, err := net.Listen("tcp", "localhost:3334")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterFarmStatsServer(grpcServer, &allFarms)
	grpcServer.Serve(lis)
}

func (s *farmStats) GetStats(ctx context.Context, farmID *pb.FarmID) (*pb.Farm, error) {
	allFarms.mu.Lock()
	stats, ok := allFarms.stats[farmID.Id]
	allFarms.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("404 not found")
	}
	return &pb.Farm{
		Id:        farmID.Id,
		Abigail:   stats.Abigail,
		Alex:      stats.Alex,
		Caroline:  stats.Caroline,
		Clint:     stats.Clint,
		Demetrius: stats.Demetrius,
		Dwarf:     stats.Dwarf,
		Elliott:   stats.Elliott,
		Emily:     stats.Emily,
		Evelyn:    stats.Evelyn,
		George:    stats.George,
		Gus:       stats.Gus,
		Haley:     stats.Haley,
		Harvey:    stats.Harvey,
		Henchman:  stats.Henchman,
		Jas:       stats.Jas,
		Jodi:      stats.Jodi,
		Kent:      stats.Kent,
		Krobus:    stats.Krobus,
		Leah:      stats.Leah,
		Lewis:     stats.Lewis,
		Linus:     stats.Linus,
		Marnie:    stats.Marnie,
		Maru:      stats.Maru,
		Pam:       stats.Pam,
		Penny:     stats.Penny,
		Pierre:    stats.Pierre,
		Robin:     stats.Robin,
		Sam:       stats.Sam,
		Sandy:     stats.Sandy,
		Sebastian: stats.Sebastian,
		Shane:     stats.Shane,
		Vincent:   stats.Vincent,
		Willy:     stats.Willy,
		Wizard:    stats.Wizard,
	}, nil
}

func httpServer() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", middleware.GetReqID(r.Context()))
		allFarms.mu.Lock()
		render.JSON(w, r, allFarms.stats)
		allFarms.mu.Unlock()

	})

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/favicon.ico")
	})

	http.ListenAndServe(":8080", r)
}

func telnetServer(telnetPort string, queue chan string) {
	telnetSvr := tcp_server.New("127.0.0.1:" + telnetPort)
	telnetSvr.OnNewClient(func(c *tcp_server.Client) {
		// log.Println("new connection")
		c.Send("welcome\n")
		// log.Println("sent welcome message")
	})
	telnetSvr.OnNewMessage(func(c *tcp_server.Client, message string) {
		// log.Printf("received message %s", message)
		message = strings.TrimRight(message, "\r\n")
		if len(message) == 0 {
			// empty line, so message[0] is not present :-) issue #1
			c.Send("/help for help\n")
			return
		}

		if message[0] == '/' {
			switch {
			case message == "/ping":
				c.Send("pong\n")
			case message == "/help":
				c.Send("usage:\n" +
					"1F4Tjc - fetch farm 1F4Tjc if it's a valid id\n" +
					"/ping - check connection, returns 'pong'\n" +
					"/qsize - query size of queue\n" +
					"/show - show current stats\n" +
					"/fetch - fetch latest farm list and process new ones\n" +
					"/spider - grab latest farms and add to queue\n" +
					"/spider 3 - grab page 3 of historical farms and add to queue\n" +
					"/quit - terminate connection\n")
			case message == "/fetch":
				fetchRecents(queue)
				c.Send("fetched recent farms\n")
			case message == "/qsize":
				c.Send(fmt.Sprintf("queue size is [%d]\n", len(queue)))
			case message == "/quit":
				c.Close()
			case message == "/show":
				c.Send(fmt.Sprintf("stats:\n"))
				allFarms.mu.Lock()
				for _, stats := range allFarms.stats {
					c.Send(fmt.Sprintf("%s likes Abigail %d/10\n", stats.FarmID, stats.Abigail))
				}
				allFarms.mu.Unlock()
			case message == "/spider":
				go func() {
					fetchMany(queue)
				}()
				c.Send("simple spider\n")
			case strings.HasPrefix(message, "/spider "):
				pageNum, err := strconv.Atoi(strings.TrimPrefix(message, "/spider "))
				if err != nil {
					c.Send("invalid page number")
					return
				}

				c.Send(fmt.Sprintf("valid page number %d", pageNum))

				go func() {
					fetchPage(queue, pageNum)
				}()
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
		c.Send("invalid farm id (/help for help)\n")
		return
	})
	telnetSvr.Listen()
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

func fetchPage(queue chan string, pageNum int) {
	pageNum++

	v := url.Values{}
	v.Set("sort", "recent")
	v.Set("p", strconv.Itoa(pageNum))
	u := &url.URL{
		Scheme:   "https",
		Host:     "upload.farm",
		Path:     "/all",
		RawQuery: v.Encode(),
	}

	body, err := fetchURL(u.String())
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

func fetchMany(queue chan string) {
	body, err := fetchURL("https://upload.farm/all?p=4695&sort=recent")
	if err != nil {
		return
	}

	farmIDs, err := farmIDsFromSearch(body)
	if err != nil {
		log.Warnf("could not find farmIDs: %s\n", err)
		return
	}

	for _, farmID := range farmIDs {
		log.Debugf("queueing farmID [%s] [%d]", farmID, len(queue))
		queue <- farmID
		log.Debugf("queued farmID [%s] [%d]", farmID, len(queue))
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
		log.Debugf("extract farmID: %s -> %s (%v)", entry, farmID, err)
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

	allFarms.mu.Lock()
	_, ok := allFarms.stats[farmID]
	allFarms.mu.Unlock()
	if ok {
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
	if miniRecent[0] != '1' {
		return "", fmt.Errorf("invalid FarmID; should start with 1")
	}
	id := miniRecent[:6]
	return id, nil
}
