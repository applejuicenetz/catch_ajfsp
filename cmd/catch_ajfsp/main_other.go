//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "catch_ajfsp läuft nur unter Windows.")
	os.Exit(1)
}
