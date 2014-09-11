package imgserver

import (
	"bytes"
	"github.com/nfnt/resize"
	"github.com/pilu/traffic"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"
)

type (
	ImgServerApi struct {
		host       string
		port       int
		imageDir   Directory
		imageCache Cache
	}
)

func NewImgServerApi(host string, port int, imageDir string, cacheSize int) ImgServerApi {

	return ImgServerApi{host, port, NewFsDirectory(imageDir), NewMapCache(cacheSize)}
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

func calcSize(paramWidth, paramHeight, origWidth, origHeight int) (uint, uint) {

	if paramWidth > origWidth {
		paramWidth = 0
	}

	if paramHeight > origHeight {
		paramHeight = 0
	}

	if paramWidth <= 0 && paramHeight <= 0 {
		return 0, 0
	} else if paramHeight <= 0 {
		return uint(paramWidth), 0
	} else if paramWidth <= 0 {
		return 0, uint(paramHeight)
	} else {
		widthScaling := float64(paramWidth) / float64(origWidth)
		scaledHeight := widthScaling * float64(origHeight)
		if int(scaledHeight) <= paramHeight {
			return uint(paramWidth), uint(scaledHeight)
		} else {
			heightScaling := float64(paramHeight) / float64(origHeight)
			scaledWidth := heightScaling * float64(origWidth)
			return uint(scaledWidth), uint(paramHeight)
		}
	}
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
	cacheKey := params.Get("image") + string(width) + string(height)

	if cachedImage := api.imageCache.Get(cacheKey); cachedImage != nil {
		sendJpeg(w, cachedImage.([]byte))
	} else {
		origImage, err := api.imageDir.ReadImage(params.Get("image"))

		if err != nil {
			w.WriteHeader(http.StatusNotFound)

		} else {
			var sizedImage image.Image

			if width > 0 || height > 0 {
				bounds := origImage.Bounds()
				newWidth, newHeight := calcSize(width, height, bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y)
				sizedImage = resize.Resize(newWidth, newHeight, origImage, resize.Lanczos3)
			} else {
				sizedImage = origImage
			}

			buffer := new(bytes.Buffer)
			err = jpeg.Encode(buffer, sizedImage, &jpeg.Options{90})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.WriteText("500 jpeg encoder failed")
			} else {
				data := buffer.Bytes()
				api.imageCache.Put(cacheKey, data)
				sendJpeg(w, data)
			}
		}
	}
}

func (api *ImgServerApi) uploadHandler(w traffic.ResponseWriter, r *traffic.Request) {

	filename := r.URL.Query().Get("image")
	uploadedImage, _, err := image.Decode(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		err = api.imageDir.WriteImage(filename, uploadedImage, 90)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func (api *ImgServerApi) deleteHandler(w traffic.ResponseWriter, r *traffic.Request) {

	filename := r.URL.Query().Get("image")
	err := api.imageDir.DeleteFile(filename)
	api.imageCache.Remove(api.imageCache.FindKeys(filename))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (api *ImgServerApi) Start() {

	traffic.SetHost(api.host)
	traffic.SetPort(api.port)
	router := traffic.New()
	router.Get("/:image", api.imageHandler)
	router.Post("/:image", api.uploadHandler)
	router.Delete("/:image", api.deleteHandler)
	router.Run()
}