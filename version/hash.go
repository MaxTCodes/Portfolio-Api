package version

import (
	"github.com/gosimple/hashdir"
)

func GetVersionHash() string {
	dirHash, _ := hashdir.Make("./", "sha1")
	return dirHash[:10]
}
