package main

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

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

func TestTelnet(t *testing.T) {
	Convey("When the telnet server is run", t, func() {
		queue := make(chan string, 100)
		go telnetServer("3334", queue)
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
