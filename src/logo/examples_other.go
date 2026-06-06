//go:build !windows && !darwin

package logo

// sous Linux (et autres Unix), le paquet installe "examples" dans
// /usr/share/doc/gologo/examples
func platformExamplesDir() string {
	return "/usr/share/doc/gologo/examples"
}
