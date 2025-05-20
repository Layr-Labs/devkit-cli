package version

var (
	Version string
	Commit  string
)

func GetVersion() string {
	if Version == "" {
		return "unknown"
	}
	return Version
}

func GetCommit() string {
	if Commit == "" {
		return "unknown"
	}
	return Commit
}
