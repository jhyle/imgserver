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
	"testing"
)

func TestXYZ(t *testing.T) {

	tmpPath, err := ioutil.TempDir("", "imgserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpPath)

	api := imgserver.NewImgServerApi("localhost", 3030, tmpPath, 1024)
	go api.Start()

	buffer := new(bytes.Buffer)
	url := "http://localhost:3030/test.jpg"
	rgba := image.NewRGBA(image.Rect(0, 0, 100, 200))
	jpeg.Encode(buffer, rgba, &jpeg.Options{90})
	http.Post(url, "image/jpeg", buffer)

	resp, err := http.Get(url + "?width=1000&height=10")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("GET returned " + strconv.Itoa(resp.StatusCode))
	}

	req, err := http.NewRequest("DELETE", url, nil)
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
