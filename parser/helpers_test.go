package parser

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestLoadPackage(t *testing.T) {
	Convey("Should load package with syntax and imports", t, func() {
		_, err := loadPackage("../testdata/subservice/subarithservice.go")
		So(err, ShouldBeNil)
	})
}
