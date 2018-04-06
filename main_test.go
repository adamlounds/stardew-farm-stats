package main

import (
	"bytes"
	"fmt"
	"io"
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
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			defer conn.Close()
			So(err, ShouldBeNil)
			// var buf bytes.Buffer
			// io.Copy(&buf, conn)
			// msg, err := buf.ReadString('\n')
			// So(err, ShouldBeNil)

			// So(msg, ShouldEqual, "welcome\n")

			// r := bufio.NewReader(conn)
			// w := bufio.NewWriter(conn)
			// scanner := bufio.NewScanner(r)
			// time.Sleep(1000 * time.Millisecond)
			// // w := bufio.NewWriter(conn)
			// w.Write([]byte("/ping\n"))
			// w.Flush()
			// log.Printf("can read %d bytes from conn", r.Buffered())
			// w.Write([]byte("/ping\n"))
			// w.Flush()
			// time.Sleep(1000 * time.Millisecond)
			// log.Printf("can read %d bytes from conn", r.Buffered())
			// for scanner.Scan() {
			// 	fmt.Printf("scanned %s", scanner.Text())
			// 	break
			// }
			// log.Printf("scanned")
			var buf bytes.Buffer
			_, err = io.Copy(&buf, conn)
			msg, err := buf.ReadString('\n') // copies until EOF, 'welcome'
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "welcome\n")

			fmt.Fprint(conn, "/ping\n")
			time.Sleep(500 * time.Millisecond)

			var buf2 bytes.Buffer
			numBytes, err := io.Copy(&buf2, conn)
			fmt.Printf("copied %d bytes (%s)", numBytes, err) // gets timeout error!
			msg, err = buf2.ReadString('\n')
			fmt.Printf("copied %s, %s", msg, err)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "pong\n")

			// for {
			// 	count := r.Buffered()
			// 	log.Printf("can read %d bytes from r", count)
			// 	if count > 0 {
			// 		break
			// 	}
			// 	break
			// }

			// Convey("and send a ping", func() {
			// 	conn.Write([]byte("/ping\n"))
			// 	time.Sleep(1000 * time.Millisecond)
			// 	// var buf bytes.Buffer
			// 	// io.Copy(&buf, conn)
			// 	// msg, err := buf.ReadString('\n')
			// 	So(err, ShouldBeNil)

			// 	// So(msg, ShouldEqual, "welcome\n")
			// })
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
