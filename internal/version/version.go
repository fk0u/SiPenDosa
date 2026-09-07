package version

import (
	"fmt"
	"runtime"
)

// CurrentVersion is the official version of SiPenDosa
const (
	CurrentVersion = "1.1.1"
	GitRepoOwner   = "fk0u"
	GitRepoName    = "SiPenDosa"
)

// FullVersion returns formatted version with build information
func FullVersion() string {
	return fmt.Sprintf("v%s (%s/%s)", CurrentVersion, runtime.GOOS, runtime.GOARCH)
}
