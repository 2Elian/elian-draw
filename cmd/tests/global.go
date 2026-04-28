package main

import "strings"

func main() {
	var hashString string = "data:hello"
	var preHash string = "data:"
	youMap := strings.HasPrefix(hashString, preHash)
	println(youMap)
}
