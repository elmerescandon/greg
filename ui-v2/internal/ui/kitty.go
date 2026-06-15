package ui

import "os"

func IsKittySupported() bool {
	tp := os.Getenv("TERM_PROGRAM")
	return tp == "ghostty" || tp == "kitty" || os.Getenv("KITTY_WINDOW_ID") != ""
}
