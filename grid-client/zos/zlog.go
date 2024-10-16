package zos

type ZLogs struct {
	// ZMachine stream logs for which zmachine
	ZMachine string `json:"zmachine"`
	// Output url
	Output string `json:"output"`
}
