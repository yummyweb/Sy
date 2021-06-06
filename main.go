package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"
var Bold = "\u001b[1m"

func formattedPrint(s string, color string) {
	switch color {
	case "red":
		fmt.Printf(Red + s + Reset)
	case "yellow":
		fmt.Printf(Yellow + s + Reset)
	case "green":
		fmt.Printf(Green + s + Reset)
	case "bold":
		fmt.Printf(Bold + s + Reset)
	}
}

func (d Download) Start() error {
	formattedPrint("Starting download...\n", "bold")
	_, err := url.ParseRequestURI(d.Url)
	if err != nil {
		return err
	}

	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	if res.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", res.StatusCode))
	}

	size, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	var sections = make([][2]int, d.TotalSections)
	secSize := size / d.TotalSections

	for i := range sections {
		if i == 0 {
			// Starting byte of first section
			sections[i][0] = 0
		} else {
			// Starting byte of other section
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			// Ending byte of other section
			sections[i][1] = sections[i][0] + secSize
		} else {
			// Last section will be size - 1
			sections[i][1] = size - 1
		}
	}

	var wg sync.WaitGroup
	for i, s := range sections {
		wg.Add(1)
		i := i
		s := s
		go func() {
			defer wg.Done()
			err := d.download(i, s)
			if err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()
	d.mergeFiles(sections)
	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)

	if err != nil {
		return nil, err
	}

	r.Header.Set("User-Agent", "Sy Download Manager v0.1")

	return r, nil
}

func (d Download) download(index int, section [2]int) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", section[0], section[1]))

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", index), b, os.ModePerm)

	if err != nil {
		return err
	}

	return nil
}

func (d Download) mergeFiles(sections [][2]int) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	for i := range sections {
		fn := fmt.Sprintf("section-%v.tmp", i)
		// Read the temporary byte file
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}
		// Merge the bytes in the file
		_, err = f.Write(b)
		if err != nil {
			return err
		}
		// Delete file once we merge it
		os.Remove(fn)
	}
	return nil
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
		for range c {
			formattedPrint("\nSIGTERM: Exiting gracefully\n", "red")
			formattedPrint("Bye Bye 👋\n", "red")
			os.Exit(1)
		}
	}()
	var url string
	var filename string
	var sections int

	fmt.Printf("Url: ")
	fmt.Scanln(&url)

	fmt.Printf("Filename: ")
	fmt.Scanln(&filename)

	fmt.Printf("Total sections: ")
	fmt.Scanln(&sections)

	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Start()
	startTime := time.Now()
	d := Download{
		Url:           url,
		TargetPath:    filename,
		TotalSections: sections,
	}
	err := d.Start()
	if err != nil {
		s.Stop()
		log.Fatalf("An error occured while downloading the file: %s", err)
	}
	s.Stop()
	formattedPrint(fmt.Sprintf("🚀 Download completed in %v seconds", time.Now().Sub(startTime).Seconds()), "green")
}
