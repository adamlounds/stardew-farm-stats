package main

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/go-redis/redis"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	//log.SetOutput(ioutil.Discard)
}

func TestExtraction(t *testing.T) {
	Convey("When parsing a JSON bytearray", t, func() {
		Convey("Invalid json array entries are ignored", func() {
			body := []byte("[\"1BC123.stuff\", \"\", \"1bc123\"]")
			farmIDs, err := extractFarmIDs(body)
			So(err, ShouldBeNil)
			So(len(farmIDs), ShouldEqual, 2)
			So(farmIDs[0], ShouldEqual, "1BC123")
			So(farmIDs[1], ShouldEqual, "1bc123")
		})
		Convey("an invalid json bytearray returns an error", func() {
			body := []byte("x[]")
			_, err := extractFarmIDs(body)
			So(err, ShouldNotBeNil)
		})
	})
	Convey("Given a valid mini_recents entry", t, func() {
		farmID, err := extractFarmID("1BC123.otherstuff")
		So(err, ShouldBeNil)
		So(farmID, ShouldEqual, "1BC123")
	})

	Convey("Given an empty mini_recents entry", t, func() {
		_, err := extractFarmID("")
		So(err, ShouldNotBeNil)
	})

	Convey("Given a short mini_recents entry", t, func() {
		_, err := extractFarmID("short")
		So(err, ShouldNotBeNil)
	})
}

func TestIDToNum(t *testing.T) {
	Convey("When converting ids to numbers", t, func() {
		num, err := idToNum("0")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 0)
		num, err = idToNum("1")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 1)
		num, err = idToNum("Z")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 61)

		num, err = idToNum("00")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 0)
		num, err = idToNum("01")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 1)

		num, err = idToNum("10")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 62)
		num, err = idToNum("ZZ")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 3843)
		num, err = idToNum("100")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 3844)

		num, err = idToNum("1H0thB")
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 1551627847)

		num, err = idToNum("abc-123")
		So(err, ShouldNotBeNil)
		So(num, ShouldEqual, 0)
	})
}
func TestNumToID(t *testing.T) {
	Convey("When converting numbers to ids", t, func() {
		str, err := numToID(0)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "")
		str, err = numToID(1)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "1")
		str, err = numToID(61)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "Z")

		str, err = numToID(62)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "10")
		str, err = numToID(3843)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "ZZ")
		str, err = numToID(3844)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "100")

		str, err = numToID(1551627847)
		So(err, ShouldBeNil)
		So(str, ShouldEqual, "1H0thB")

		str, err = numToID(-1)
		So(err, ShouldNotBeNil)
		So(str, ShouldEqual, "")
	})
}

func TestTelnet(t *testing.T) {
	Convey("When the telnet server is run", t, func() {
		queue := make(chan string, 100)
		nilRedis := redis.NewClient(&redis.Options{
			Addr:     ":6379",
			PoolSize: 0,
		})
		go telnetServer("3334", queue, nilRedis)
		time.Sleep(100 * time.Millisecond)

		Convey("we can connect to it", func() {
			d := net.Dialer{Timeout: 10 * time.Millisecond}
			conn, err := d.Dial("tcp", "127.0.0.1:3334")
			conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
			defer conn.Close()
			So(err, ShouldBeNil)

			r := bufio.NewReader(conn)
			msg, err := r.ReadString('\n')
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "welcome\n")

			Convey("...ping to get a response", func() {
				fmt.Fprint(conn, "/ping\n")
				msg, err := r.ReadString('\n')
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "pong\n")

				Convey("...and sending an empty line does not crash (issue #1)", func() {
					fmt.Fprint(conn, "\n")
					msg, err := r.ReadString('\n')
					So(err, ShouldBeNil)
					So(msg, ShouldEqual, "/help for help\n")
				})
			})
		})

	})
	Convey("Given a valid mini_recents entry", t, func() {
		farmID, err := extractFarmID("1BC123.otherstuff")
		So(err, ShouldBeNil)
		So(farmID, ShouldEqual, "1BC123")
	})

	Convey("Given an empty mini_recents entry", t, func() {
		_, err := extractFarmID("")
		So(err, ShouldNotBeNil)
	})

	Convey("Given a short mini_recents entry", t, func() {
		_, err := extractFarmID("short")
		So(err, ShouldNotBeNil)
	})
}
