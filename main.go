package main

import (
	"fmt"
)

func main() {
	log := getLogger()

	cfg, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	rto, err := parseManifests(cfg.String("input"))
	if err != nil {
		log.Fatal(err)
	}
	for _, o := range rto {
		fmt.Printf("%v", o)
	}
}
