package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/reujab/wallpaper"
)

var (
	// Command line flags
	outputFile string
	updateTime string
	killFlag   bool
)

func init() {
	flag.StringVar(&outputFile, "o", "", "output file for logs")
	flag.StringVar(&updateTime, "t", "08:00", "24-hour time when wallpaper is updated")
	flag.BoolVar(&killFlag, "k", false, "update wallpaper once and exit")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "OPTIONS:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	output := io.Writer(os.Stdout)

	if outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Printf("[ERR] unable to open output file %q: %v", outputFile, err)
			os.Exit(1)
		}
		defer f.Close()
		output = f
	}

	log.SetPrefix("[bingo] ")
	log.SetOutput(output)

	start()
}

func start() {
	// set first time on launch
	setBingWallpaper()

	if killFlag {
		log.Printf("[INF] kill flag provided, exiting")
		os.Exit(0)
	}

	// set again daily
	if err := gocron.Every(1).Day().At(updateTime).Do(setBingWallpaper); err != nil {
		log.Printf("[ERR] failed to create daily update job at %q: %v", updateTime, err)
		os.Exit(1)
	}
	log.Printf("[INF] wallpaper will be updated again daily at %s", updateTime)

	<-gocron.Start()
}

// setBingWallpaper sets the wallpaper to Bing's current image of the day,
// if it fails an error is logged.
func setBingWallpaper() {
	image, err := bingImageOfTheDay()
	if err != nil {
		log.Printf("[ERR] unable to retrieve Bing image of the day: %v", err)
		return
	}
	// append a dummy appendix for wallpaper module to recognize the filename
	image.URL = image.URL + "/fakefilename=2020.jpg"
	log.Printf("[INF] updating wallpaper, url: %q, copyright: %q", image.URL, image.Copyright)

	if err := wallpaper.SetFromURL(image.URL); err != nil {
		log.Printf("[ERR] unable to set wallpaper: %v", err)
	}
}

var client = http.Client{Timeout: 30 * time.Second}

type image struct {
	URL       string `json:"url"`
	Copyright string `json:"copyright"`
}

// bingImageOfTheDay returns Bing's current image of the day.
func bingImageOfTheDay() (*image, error) {
	resp, err := client.Get("https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1")
	if err != nil {
		return nil, fmt.Errorf("http GET: %v", err)
	}
	defer resp.Body.Close()

	root := struct {
		Images []image `json:"images"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("decode body: %v", err)
	}

	if len(root.Images) == 0 {
		return nil, errors.New("response does not contain an image")
	}

	image := root.Images[0]
	image.URL = "https://www.bing.com" + image.URL

	return &image, nil
}
