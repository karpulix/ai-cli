package version

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("ai-cli %s", Display())
}

func Display() string {
	v := Version
	if v == "" {
		v = "dev"
	}
	if Commit != "none" && Commit != "" {
		return fmt.Sprintf("v%s · %s · %s", v, Commit, Date)
	}
	return "v" + v
}
