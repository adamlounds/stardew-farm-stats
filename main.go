package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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
	"github.com/go-chi/render"
	"github.com/go-redis/redis"
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
type spiderStatus struct {
	mu         sync.Mutex
	stop       bool
	numRunning int
}

var allFarms farmStats
var status spiderStatus

var serverCtx context.Context
var chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var httpClient = http.DefaultClient

func setupHTTPClient() {
	proxyStr := os.Getenv("http_proxy")
	if proxyStr == "" {
		log.Warn("no http_proxy")
		return
	}

	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		log.Warnf("Invalid http_proxy %v", err)
		return
	}

	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}

	log.Infof("using http proxy %v", proxyURL)
	httpClient.Transport = transport
}

func main() {
	//log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	log.Infof("Starting Innocuous server %s %d", "v1.0", runtime.GOMAXPROCS(0))
	redisdb := redis.NewClient(&redis.Options{
		Addr:     ":6379",
		PoolSize: 0,
	})
	setupHTTPClient()

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
	go telnetServer(defaultTelnetPort, queue, redisdb)
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

	// TODO: graceful shutdown via context.withTimeout?
	// err := http.Shutdown(ctx) (returns err if context expires)
	// grpc.Server.GracefulStop() (blocks until complete)
	// telnet err := tcp_server.Conn.Close()

	// wait for ctrl-c (sigkill)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Infof("Shutting down\n")
	os.Exit(0)

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

func telnetServer(telnetPort string, queue chan string, redisdb *redis.Client) {
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
					"/qsize - size of farm-fetching queue\n" +
					"/show - show current stats\n" +
					"/spiderstatus - show number of running spiders\n" +
					"/fetch - fetch latest farm list and process new ones\n" +
					"/spider - grab latest farms and add to queue\n" +
					"/spider 3 - grab page 3 of historical farms and add to queue\n" +
					"/spiderall 3 - grab from page 3 to 1 of historical farms & add to known farms list in redis\n" +
					"/stopspider - tell \"spiderall\" spiders to stop\n" +
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
			case message == "/spiderstatus":
				status.mu.Lock()
				numSpiders := status.numRunning
				status.mu.Unlock()

				c.Send(fmt.Sprintf("status: %d spiders running\n", numSpiders))
			case message == "/stopspider":
				status.mu.Lock()
				numSpiders := status.numRunning
				status.stop = true
				status.mu.Unlock()

				c.Send(fmt.Sprintf("asked %d spiders to stop\n", numSpiders))
			case message == "/spider":
				go func() {
					fetchMany(queue, redisdb)
				}()
				c.Send("simple spider of homepage farms\n")
			case strings.HasPrefix(message, "/spiderall "):
				pageNum, err := strconv.Atoi(strings.TrimPrefix(message, "/spiderall "))
				if err != nil {
					c.Send("invalid last page number")
					return
				}

				c.Send(fmt.Sprintf("spidering from page %d", pageNum))
				go func() {
					status.mu.Lock()
					status.numRunning++
					status.mu.Unlock()

					var seenIDs []string
					var seenPages int
					for i := pageNum; i >= 0; i-- {
						status.mu.Lock()
						shouldStop := status.stop
						status.mu.Unlock()

						if shouldStop == true {
							log.Info("received stop signal!")
							break
						}
						idsFromPage, err := fetchPage(redisdb, i)
						if err != nil {
							continue
						}
						seenIDs = append(seenIDs, idsFromPage...)
						seenPages++
					}
					status.mu.Lock()
					status.numRunning--
					if status.numRunning == 0 {
						status.stop = false
					}
					status.mu.Unlock()

					log.Infof("finished reading pages, saw %d farms on %d pages", len(seenIDs), seenPages)
				}()
				c.Send("multipage spider\n")
			case strings.HasPrefix(message, "/spider "):
				pageNum, err := strconv.Atoi(strings.TrimPrefix(message, "/spider "))
				if err != nil {
					c.Send("invalid page number")
					return
				}

				c.Send(fmt.Sprintf("valid page number %d", pageNum))

				go func() {
					idsFromPage, err := fetchPage(redisdb, pageNum)
					if err == nil {
						for _, farmID := range idsFromPage {
							fmt.Printf("queueing farmID [%s] [%d]", farmID, len(queue))
							queue <- farmID
							fmt.Printf("queued farmID [%s] [%d]", farmID, len(queue))
						}
					}
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

func fetchPage(redisdb zAddNXer, pageNum int) ([]string, error) {
	v := url.Values{}
	v.Set("sort", "recent")
	v.Set("p", strconv.Itoa(pageNum))
	u := &url.URL{
		Scheme:   "https",
		Host:     "upload.farm",
		Path:     "/all",
		RawQuery: v.Encode(),
	}

	var farmIDs []string

	body, err := fetchURL(u.String())
	if err != nil {
		log.Warnf("[%d] could not fetchURL: %s\n", pageNum, err)
		return farmIDs, err
	}

	farmIDs, err = farmIDsFromSearch(body)
	if err != nil {
		log.Warnf("[%d] could not find farmIDs: %s\n", pageNum, err)
		log.Debugf("[%d] %s", pageNum, body)
		return farmIDs, err
	}

	var zids []redis.Z
	for _, farmID := range farmIDs {

		idScore, err := idToNum(farmID)
		if err != nil {
			fmt.Printf("cannot convert id? [%s] [%v]", farmID, err)
			continue
		}
		zid := redis.Z{Score: float64(idScore), Member: farmID}
		zids = append(zids, zid)
	}

	if len(zids) == 0 {
		fmt.Printf("no farms found!\n")
		return farmIDs, err
	}

	result, err := redisdb.ZAddNX("spidered", zids...).Result()
	if err != nil {
		log.Warnf("could not add farmIDs to redis [spidered]: %v", err)
	}
	fmt.Printf("zadd: [%v] [%v]\n", result, err)
	// TODO when result == 0, no new items were added. Stop spidering...

	return farmIDs, nil
}

type zAddNXer interface {
	ZAddNX(key string, members ...redis.Z) *redis.IntCmd
	PoolStats() *redis.PoolStats
}

func fetchMany(queue chan string, redisdb zAddNXer) {
	body, err := fetchURL("https://upload.farm/all?p=4695&sort=recent")
	if err != nil {
		return
	}

	farmIDs, err := farmIDsFromSearch(body)
	if err != nil {
		log.Warnf("could not find farmIDs: %s\n", err)
		return
	}

	var zids []redis.Z

	for _, farmID := range farmIDs {
		log.Debugf("queueing farmID [%s] [%d]", farmID, len(queue))
		queue <- farmID
		log.Debugf("queued farmID [%s] [%d]", farmID, len(queue))
		idScore, err := idToNum(farmID)
		if err != nil {
			fmt.Printf("cannot convert id? [%s] [%v]", farmID, err)
			continue
		}
		zid := redis.Z{Score: float64(idScore), Member: farmID}
		zids = append(zids, zid)
	}
	if len(zids) == 0 {
		fmt.Printf("no farms found!\n")
		return
	}
	result, err := redisdb.ZAddNX("spidered", zids...).Result()
	fmt.Printf("zadd: [%v] [%v]\n", result, err)
	// fmt.Printf("%#v", redisdb.PoolStats())
}

func farmIDsFromSearch(body []byte) ([]string, error) {
	re := regexp.MustCompile("/([A-Za-z0-9]{6})-f.png")
	result := re.FindAllStringSubmatch(string(body), -1)
	if result == nil {
		log.Warnf("could not find any farms in %s", body)
		return nil, fmt.Errorf("no farms found")
	}

	var farmIDs []string
	for _, match := range result {
		farmID := match[1]
		farmIDs = append(farmIDs, farmID)
	}

	if len(farmIDs) == 0 {
		log.Warnf("could not find any farms in %s", body)
		return nil, fmt.Errorf("no farms in body?")
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
	res, err := httpClient.Get(url)
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

	if res.StatusCode == http.StatusOK {
		return body, nil
	}
	return nil, fmt.Errorf(res.Status)

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

func idToNum(id string) (int64, error) {
	base := int64(len(chars))
	multiplier := int64(1)
	num := int64(0)

	// most significant digit at left, so work from right to left
	for i := len(id) - 1; i >= 0; i-- {
		char := id[i]
		idx := strings.IndexByte(chars, char)
		if idx < 0 {
			return 0, fmt.Errorf("invalid char [%v] at [%d]", char, i)
		}
		num += multiplier * int64(idx)
		multiplier *= base
	}

	return num, nil
}

func numToID(num int64) (string, error) {
	if num < 0 {
		return "", fmt.Errorf("cannot convert negative numbers")
	}
	base := int64(len(chars))
	var id []byte
	for {
		if num <= 0 {
			break
		}
		rem := num % base
		id = append([]byte{chars[rem]}, id...)
		num = (num - rem) / base
	}
	return string(id), nil
}
