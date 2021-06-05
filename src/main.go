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

		if i < d.TotalSections - 1 {
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
		Url:           url, // "https://r1---sn-ci5gup-qxay.googlevideo.com/videoplayback?expire=1622899930&ei=eii7YMGeFY-R2_gPr8a1yAE&ip=70.39.77.131&id=o-AAseiVR7txfOGnHjnmQiROqj3UN63G1pKlpD6h-KneH2&itag=22&source=youtube&requiressl=yes&vprv=1&mime=video%2Fmp4&ns=jmgDNeTiDFHl__X0cksi4oMF&cnr=14&ratebypass=yes&dur=45.093&lmt=1622867710227282&fexp=24001373,24007246&c=WEB&txp=5532434&n=YPFUpZu6PvQR2MrZi&sparams=expire%2Cei%2Cip%2Cid%2Citag%2Csource%2Crequiressl%2Cvprv%2Cmime%2Cns%2Ccnr%2Cratebypass%2Cdur%2Clmt&sig=AOq0QJ8wRAIgPSO9bfztgxGiJLIchfYdcdJ0DQJ-Ke93IT3PlQQkuFACIBHnfrfPCdOuEtbN4czdwbZjzJvBKLumHuDpuMilTJz9&title=Chance%20%7C%20Marvel%20Studios%27%20Loki%20%7C%20Disney%2B&redirect_counter=1&rm=sn-qxoee7e&req_id=f78b9618ffaa3ee&cms_redirect=yes&ipbypass=yes&mh=UO&mip=171.50.140.37&mm=31&mn=sn-ci5gup-qxay&ms=au&mt=1622886524&mv=m&mvi=1&pl=22&lsparams=ipbypass,mh,mip,mm,mn,ms,mv,mvi,pl&lsig=AG3C_xAwRQIhAPQfFFgasYtTePlL28-HIDZk3GWOUjKD1O0t4jjezXNAAiAXkfdUZeKfdqrdJB67-_uJAagAYxIEbDYROOr4GGqV-A%3D%3D"
		TargetPath:    filename,
		TotalSections: sections,
	}
	err := d.Start()
	if err != nil {
		log.Fatalf("An error occured while downloading the file: %s", err)
	}
	fmt.Printf("Download completed in %v seconds", time.Now().Sub(startTime).Seconds())
}
