package imgserver

import (
	"github.com/DAddYE/vips"
	"github.com/pilu/traffic"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type (
	ImgServerApi struct {
		host       string
		port       int
		imageDir   Directory
		imageCache ByteCache
	}
)

const (
	maxHeight = 1080
	maxWidth  = 1920
)

func NewImgServerApi(host string, port int, imageDir string, cacheSize uint64) ImgServerApi {

	return ImgServerApi{host, port, NewFsDirectory(imageDir), NewByteCache(cacheSize)}
}

func toInt(input string, deflt int) int {

	result := deflt

	if len(input) > 0 {
		value, err := strconv.Atoi(input)
		if err == nil {
			result = value
		}
	}

	return result
}

func sendJpeg(w traffic.ResponseWriter, data []byte) {

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

func (api *ImgServerApi) imageHandler(w traffic.ResponseWriter, r *traffic.Request) {

	params := r.URL.Query()
	width := toInt(params.Get("width"), 0)
	height := toInt(params.Get("height"), 0)
	imagefile := params.Get("image")
	cacheKey := imagefile + string(width) + string(height)

	modTime := api.imageDir.ModTime(imagefile)
	if modTime == nil {
		api.imageCache.Remove(api.imageCache.FindKeys(imagefile))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if cachedImage := api.imageCache.Get(cacheKey, *modTime); cachedImage != nil {
		sendJpeg(w, cachedImage)
	} else {
		image, err := api.imageDir.ReadFile(imagefile)
		if err != nil {
			traffic.Logger().Print(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if width != 0 || height != 0 {
			image, err = vips.Resize(image, vips.Options{Width: width, Height: height, Enlarge: true, Embed: true, Extend: vips.EXTEND_WHITE, Interpolator: vips.BICUBIC, Quality: 85})
			if err != nil {
				traffic.Logger().Print(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		api.imageCache.Put(cacheKey, image, *modTime)
		sendJpeg(w, image)
	}
}

func (api *ImgServerApi) uploadHandler(w traffic.ResponseWriter, r *traffic.Request) {

	filename := r.URL.Query().Get("image")
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err = vips.Resize(data, vips.Options{Format: vips.JPEG, Width: maxWidth, Height: maxHeight, Crop: true, Interpolator: vips.BICUBIC})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = api.imageDir.WriteFile(filename, data)
	if err != nil {
		traffic.Logger().Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (api *ImgServerApi) copyHandler(w traffic.ResponseWriter, r *traffic.Request) {

	src := r.URL.Query().Get("src")
	dst := r.URL.Query().Get("dst")

	data, err := api.imageDir.ReadFile(src)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = api.imageDir.WriteFile(dst, data)
	if err != nil {
		traffic.Logger().Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *ImgServerApi) deleteHandler(w traffic.ResponseWriter, r *traffic.Request) {

	filename := r.URL.Query().Get("image")
	err := api.imageDir.DeleteFile(filename)
	api.imageCache.Remove(api.imageCache.FindKeys(filename))

	if err != nil {
		traffic.Logger().Print(err.Error())
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (api *ImgServerApi) statsHandler(w traffic.ResponseWriter, r *traffic.Request) {

	w.WriteJSON(api.imageCache.Stats())
}

func (api *ImgServerApi) listHandler(w traffic.ResponseWriter, r *traffic.Request) {

	age, err := strconv.Atoi(r.Param("age"))
	if err != nil {
		age = 0
	}

	files, err := api.imageDir.ListFiles(time.Duration(age) * time.Second)
	if err != nil {
		traffic.Logger().Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteJSON(files)
	}
}

func (api *ImgServerApi) Start() {

	traffic.SetHost(api.host)
	traffic.SetPort(api.port)
	router := traffic.New()
	router.Get("/", api.listHandler)
	router.Put("/", api.statsHandler)
	router.Get("/:image", api.imageHandler)
	router.Post("/:image", api.uploadHandler)
	router.Put("/:src/:dst", api.copyHandler)
	router.Delete("/:image", api.deleteHandler)
	router.Run()
}
