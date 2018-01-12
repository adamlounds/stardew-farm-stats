package main

import (
	"encoding/json"
	"fmt"
	//	"github.com/firstrow/tcp_server"
	//	"github.com/go-chi/chi"
	//	"github.com/go-chi/chi/middleware"
	"github.com/pressly/lg"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	//	"strings"
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
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	lg.RedirectStdlogOutput(logger)
	lg.DefaultLogger = logger
	serverCtx = context.Background()
	serverCtx = lg.WithLoggerContext(serverCtx, logger)
	lg.Log(serverCtx).Infof("Starting Innocuous server %s", "v1.0")

	resp, err := http.Get("https://upload.farm/_mini_recents")
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	var entries []string
	err = json.Unmarshal(body, &entries)
	if err != nil {
		lg.Log(serverCtx).Infof("cannot parse json %s (%v)", body, err)
	}

	var farmIDs []string
	for _, entry := range entries {
		farmID, err := extractFarmID(entry)
		if err != nil {
			lg.Log(serverCtx).Infof("unexpected entry %s", entry)
			continue
		}
		lg.Log(serverCtx).Infof("%s -> %s (%v)", entry, farmID, err)
		farmIDs = append(farmIDs, farmID)
	}

	//farmID, err := extractFarmID("invlid")
	//lg.Log(serverCtx).Infof("%s -> %s (%v)", first, farmID, err)

	fmt.Println(farmIDs)

}

func extractFarmID(miniRecent string) (string, error) {
	if len(miniRecent) < 7 {
		return "", fmt.Errorf("invalid miniRecent, must be at least 7 chars long")
	}
	id := miniRecent[:6]
	return id, nil
}
