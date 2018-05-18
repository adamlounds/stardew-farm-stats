package main

import (
	"runtime"
	"time"

	pb "github.com/adamlounds/stardew-farm-stats/farmstats"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.Infof("Starting grpc client %s %d", "v1.0", runtime.GOMAXPROCS(0))

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	serverAddr := "127.0.0.1:3334"
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewFarmStatsClient(conn)

	// first req ~ 1.8ms
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
	time.Sleep(2 * time.Second)
	// subsequent reqs (with same connection?) ~ 0.6ms
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
	printStats(client, &pb.FarmID{Id: "1FkeUV"})
}

// printFeature gets the feature for the given point.
func printStats(client pb.FarmStatsClient, farmID *pb.FarmID) {
	log.Infof("Getting feature for farm (%s)", farmID.Id)
	t1 := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stats, err := client.GetStats(ctx, farmID)
	if err != nil {
		log.Fatalf("%v.GetStats(_) = _, %v: ", client, err)
	}
	dur := time.Since(t1)
	log.Infof("got stats: [%v] in %s", stats, dur)
}
