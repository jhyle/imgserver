package main

import (
	"flag"
	"fmt"
	"github.com/jhyle/imgserver/api"
	"net/http"
	_ "net/http/pprof"
	"os"
)

const (
	APP_VERSION = "0.1"
)

// The flag package provides a default help printer via -h switch
var (
	versionFlag   *bool   = flag.Bool("v", false, "print the version number")
	portFlag      *int    = flag.Int("p", 3000, "port to listen on")
	hostFlag      *string = flag.String("i", "127.0.0.1", "interface to listen on")
	imageDirFlag  *string = flag.String("imageDir", "", "path to images")
	cacheSizeFlag *uint64 = flag.Uint64("cacheSize", 1024*1024*1024*32, "maximum cache size in bytes")
)

func IsFolder(path string) bool {

	folder, err := os.Stat(path)
	if err != nil {
		return false
	}

	return folder.IsDir()
}

func main() {
	flag.Parse() // Scan the arguments list

	if *versionFlag {
		fmt.Println("Version:", APP_VERSION)
	}

	if *imageDirFlag == "" {
		fmt.Println("You need to specify an image directory (-imageDir)!")
		os.Exit(1)
	}

	if !IsFolder(*imageDirFlag) {
		fmt.Println("Given image directory (-imageDir=" + *imageDirFlag + ") is not a directory!")
		os.Exit(1)
	}

	// start debugging server
	go func() {
		http.ListenAndServe("localhost:6000", nil)
	}()

	imgServer := imgserver.NewImgServerApi(*hostFlag, *portFlag, *imageDirFlag, *cacheSizeFlag)
	imgServer.Start()
}
