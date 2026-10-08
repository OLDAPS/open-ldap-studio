package main

// versionRequested reports whether the command line asks for the build
// identity rather than the window.
func versionRequested(args []string) bool {
	for _, a := range args {
		if a == "--version" || a == "-version" {
			return true
		}
	}
	return false
}
