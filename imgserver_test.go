package main

import (
	"bytes"
	"github.com/jhyle/imgserver/api"
	"image"
	"image/jpeg"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestImgServer(t *testing.T) {

	tmpPath, err := ioutil.TempDir("", "imgserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpPath)

	// start debugging server
	go func() {
		http.ListenAndServe("localhost:6000", nil)
	}()

	api := imgserver.NewImgServerApi("localhost", 3030, tmpPath, 1024*2)
	go api.Start()
	time.Sleep(time.Second * 3)

	buffer := new(bytes.Buffer)
	baseUrl := "http://localhost:3030/"
	filename := "test.jpg"
	url := baseUrl + filename
	rgba := image.NewRGBA(image.Rect(0, 0, 5000, 5000))
	jpeg.Encode(buffer, rgba, &jpeg.Options{90})
	resp, err := http.Post(url, "image/jpeg", buffer)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("POST returned " + strconv.Itoa(resp.StatusCode))
	}

	var wait sync.WaitGroup
	wait.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wait.Done()
			for i := 0; i < 100; i++ {
				for _, w := range []int{100, 200, 400, 800, 1600} {
					resp, err := http.Get(url + "?width=" + strconv.Itoa(w) + "&height=10")
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != http.StatusOK {
						t.Fatal("GET returned " + strconv.Itoa(resp.StatusCode))
					}
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
				}
			}
		}()
	}
	wait.Wait()

	req, err := http.NewRequest("PUT", baseUrl, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("PUT returned " + strconv.Itoa(resp.StatusCode))
	} else {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(data[:]))
		}
	}

	filenameCopy := "test2.jpg"
	req, err = http.NewRequest("PUT", url+"/"+filenameCopy, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("PUT returned " + strconv.Itoa(resp.StatusCode))
	}

	urlCopy := baseUrl + filenameCopy
	resp, err = http.Get(urlCopy)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("GET returned " + strconv.Itoa(resp.StatusCode))
	}

	req, err = http.NewRequest("DELETE", urlCopy, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("DELETE returned " + strconv.Itoa(resp.StatusCode))
	}

	req, err = http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("DELETE returned " + strconv.Itoa(resp.StatusCode))
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatal("image exists after deletion")
	}
}
