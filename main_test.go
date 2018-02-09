package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
)

func init() {
	log.SetOutput(ioutil.Discard)
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
