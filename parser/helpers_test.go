package parser

import (
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thoas/go-funk"
	"testing"
)

func TestLoadPackagesRecursive(t *testing.T) {
	Convey("Should get files from entrypoint", t, func() {
		files, err := GetDependencies("github.com/semrush/zenrpc/v2/testdata/subservice/subarithservice.go")
		So(err, ShouldBeNil)
		So(files, ShouldHaveLength, len(funk.UniqString(files)))
	})
}
