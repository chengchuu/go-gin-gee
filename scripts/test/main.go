package main

import (
	"log"
	"strings"
)

func main() {
	test0 := "abc"
	test1 := "abcd"
	if strings.Contains(test0, test1) {
		log.Println("test0 contains test1")
	} else {
		log.Println("test0 does not contain test1")
	}
}
