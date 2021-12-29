package option

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

var runMode modeType = ModeRelease

// SetRunMode type
func SetRunMode(mt modeType) {
	runMode = mt
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
