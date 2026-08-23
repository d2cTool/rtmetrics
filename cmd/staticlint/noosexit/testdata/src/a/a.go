package main

import "os"

func main() {
	os.Exit(1) // want "direct os.Exit call is forbidden in main"
}

func helper() {
	os.Exit(0)
}
