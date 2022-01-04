package option

import "strings"

// modeType global option
type modeType string

// String to string.
func (mt modeType) String() string {
	return string(mt)
}

const (
	ModeDebug   modeType = "DEBUG"
	ModeRelease modeType = "RELEASE"
)

// default is release mode
var runMode = ModeRelease

// SetRunMode type
func SetRunMode(mt modeType) {
	runMode = mt
}

// SetRunModeByString type
func SetRunModeByString(mt string) {
	mt = strings.ToUpper(mt)
	if mt == ModeDebug.String() {
		SetRunMode(ModeDebug)
	} else if mt == ModeRelease.String() {
		SetRunMode(ModeRelease)
	} else {
		panic("xxl-job-go: invalid mode type value")
	}
}

// RunMode get
func RunMode() string {
	return runMode.String()
}

// IsDebugMode check
func IsDebugMode() bool {
	return runMode == ModeDebug
}

// IsReleaseMode check
func IsReleaseMode() bool {
	return runMode == ModeRelease
}
