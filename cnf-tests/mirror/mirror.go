package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type image struct {
	Registry string `json:"registry"`
	Image    string `json:"image"`
}

// TODO in a second step this could even do the mirror itself
func main() {
	imagesFile := flag.String("images", "/usr/local/etc/cnf/images.json", "the json file containing the images")
	targetRegistry := flag.String("registry", "", "the target registry we want to mirror to")

	flag.Parse()

	if *imagesFile == "" || *targetRegistry == "" {
		flag.Usage()
		log.Fatal("Missing mandatory fields")
	}

	bytes, err := os.ReadFile(*imagesFile)
	if err != nil {
		log.Fatalf("Failed to read %s - %v", *imagesFile, err)
	}

	var images []image
	err = json.Unmarshal(bytes, &images)
	if err != nil {
		log.Fatalf("Failed to read %s - %v", *imagesFile, err)
	}

	registryURL := *targetRegistry
	if !strings.HasSuffix(*targetRegistry, "/") {
		registryURL += "/"
	}

	for _, img := range images {
		fmt.Printf("%s%s %s%s\n", img.Registry, img.Image, registryURL, img.Image)
	}
}
