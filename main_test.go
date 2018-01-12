package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtraction(t *testing.T) {
	Convey("Given a valid mini_recents entry", t, func() {
		farmID, err := extractFarmID("aBC123.otherstuff")
		So(err, ShouldBeNil)
		So(farmID, ShouldEqual, "aBC123")
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
