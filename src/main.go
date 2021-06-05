package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

func (d Download) Start() error {
	fmt.Println("Making connection")
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Got %v\n", res.StatusCode)

	if res.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", res.StatusCode))
	}

	size, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

	var sections = make([][2]int, d.TotalSections)
	secSize := size / d.TotalSections
	fmt.Printf("Size of each section is %v\n", secSize)

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

	fmt.Printf("Downloaded %v bytes for section %v\n", res.Header.Get("Content-Length"), index)
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
		n, err := f.Write(b)
		if err != nil {
			return err
		}
		// Delete file once we merge it
		os.Remove(fn)
		fmt.Printf("%v bytes merged\n", n)
	}
	return nil
}

func main() {
	var url string
	var filename string
	var sections int

	fmt.Println("Url: ")
	fmt.Scanln(&url)

	fmt.Println("Filename: ")
	fmt.Scanln(&filename)

	fmt.Println("Total sections: ")
	fmt.Scanln(&sections)

	startTime := time.Now()
	d := Download{
		Url:           url,
		TargetPath:    filename,
		TotalSections: sections,
	}
	err := d.Start()
	if err != nil {
		log.Fatalf("An error occured while downloading the file: %s", err)
	}
	fmt.Printf("Download completed in %v seconds", time.Now().Sub(startTime).Seconds())
}
