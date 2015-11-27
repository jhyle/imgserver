package imgserver

import (
	"bytes"
	"github.com/lazywei/go-opencv/opencv"
	"github.com/nfnt/resize"
	"github.com/pilu/traffic"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"
	"sync"
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

var (
	mutex sync.Mutex
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

func (api *ImgServerApi) detectFaces(img image.Image) image.Rectangle {

	cx, cy := (img.Bounds().Max.X-img.Bounds().Min.X)/2, (img.Bounds().Max.Y-img.Bounds().Min.Y)/2
	center := image.Rect(cx, cy, cx, cy)

	mutex.Lock()

	cascade := opencv.LoadHaarClassifierCascade("/usr/share/opencv/haarcascades/haarcascade_profileface.xml")
	if cascade == nil {
		return center
	}

	srcImg := opencv.FromImage(img)
	if srcImg == nil {
		cascade.Release()
		mutex.Unlock()
		return center
	}

	first := true
	for _, value := range cascade.DetectObjects(srcImg) {
		if value != nil {
			if first {
				first = false
				center = image.Rect(value.X(), value.Y(), value.X()+value.Width(), value.Y()+value.Height())
			} else {
				if value.X() < center.Min.X {
					center.Min.X = value.X()
				}
				if value.X()+value.Width() > center.Max.X {
					center.Max.X = value.X() + value.Width()
				}
				if value.Y() < center.Min.Y {
					center.Min.Y = value.Y()
				}
				if value.Y()+value.Height() > center.Max.Y {
					center.Max.Y = value.Y() + value.Height()
				}
			}
		}
	}

	cascade.Release()
	srcImg.Release()
	mutex.Unlock()
	return center
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
		origImage, err := api.imageDir.ReadImage(imagefile)

		if err != nil {
			traffic.Logger().Print(err.Error())
			w.WriteHeader(http.StatusNotFound)

		} else {
			var sizedImage image.Image
			bounds := origImage.Bounds()
			origWidth := bounds.Max.X - bounds.Min.X
			origHeight := bounds.Max.Y - bounds.Min.Y

			if width > 0 && height > 0 {
				if width > origWidth && height > origHeight {
					sizedImage = drawOnWhite(image.Pt(width, height), image.Pt(0, 0), image.Pt((width-origWidth)/2, (height-origHeight)/2), origImage)
				} else if width > origWidth {
					sizedImage = drawOnWhite(image.Pt(width, height), image.Pt(0, (origHeight-height)/2), image.Pt((width-origWidth)/2, 0), origImage)
				} else if height > origHeight {
					sizedImage = drawOnWhite(image.Pt(width, height), image.Pt((origWidth-width)/2, 0), image.Pt(0, (height-origHeight)/2), origImage)
				} else {
					faces := api.detectFaces(origImage)
					origAspectRatio := float64(origWidth) / float64(origHeight)
					croppedAspectRatio := float64(width) / float64(height)

					if origAspectRatio < croppedAspectRatio {
						scaling := float64(width) / float64(origWidth)
						sizedImage = resize.Resize(uint(width), uint(float64(origHeight)*scaling), origImage, resize.Lanczos3)

						dY := (sizedImage.Bounds().Dy() - height) / 2
						fY2 := int(float64(faces.Max.Y)*scaling) + int(float64(faces.Max.Y-faces.Min.Y)*scaling/5)
						if fY2 > sizedImage.Bounds().Dy() {
							fY2 = sizedImage.Bounds().Dy()
						}
						if fY2 > dY+height {
							dY = fY2 - height
						}
						fY1 := int(float64(faces.Min.Y)*scaling) - int(float64(faces.Max.Y-faces.Min.Y)*scaling/5)
						if fY1 < 0 {
							fY1 = 0
						}
						if fY1 < dY {
							dY = fY1
						}

						sizedImage = drawOnWhite(image.Pt(width, height), image.Pt(0, dY), image.Pt(0, 0), sizedImage)
					} else {
						scaling := float64(height) / float64(origHeight)
						sizedImage = resize.Resize(uint(float64(origWidth)*scaling), uint(height), origImage, resize.Lanczos3)

						dX := (sizedImage.Bounds().Dx() - width) / 2
						fX2 := int(float64(faces.Max.X)*scaling) + int(float64(faces.Max.X-faces.Min.X)*scaling/5)
						if fX2 > sizedImage.Bounds().Dx() {
							fX2 = sizedImage.Bounds().Dx()
						}
						if fX2 > dX+width {
							dX = fX2 - width
						}
						fX1 := int(float64(faces.Min.X)*scaling) - int(float64(faces.Max.X-faces.Min.X)*scaling/5)
						if fX1 < 0 {
							fX1 = 0
						}
						if fX1 < dX {
							dX = fX1
						}

						sizedImage = drawOnWhite(image.Pt(width, height), image.Pt(dX, 0), image.Pt(0, 0), sizedImage)
					}
				}
			} else if width > 0 {
				if width <= origWidth {
					scaling := float64(width) / float64(origWidth)
					sizedImage = resize.Resize(uint(width), uint(float64(origHeight)*scaling), origImage, resize.Lanczos3)
				} else {
					sizedImage = drawOnWhite(image.Pt(width, origHeight), image.Pt(0, 0), image.Pt((width-origWidth)/2, 0), origImage)
				}
			} else if height > 0 {
				if height <= origHeight {
					scaling := float64(height) / float64(origHeight)
					sizedImage = resize.Resize(uint(float64(origWidth)*scaling), uint(height), origImage, resize.Lanczos3)
				} else {
					sizedImage = drawOnWhite(image.Pt(origWidth, height), image.Pt(0, 0), image.Pt(0, (height-origHeight)/2), origImage)
				}
			} else {
				sizedImage = origImage
			}

			buffer := new(bytes.Buffer)
			err = jpeg.Encode(buffer, sizedImage, &jpeg.Options{95})
			if err != nil {
				traffic.Logger().Print(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				data := buffer.Bytes()
				api.imageCache.Put(cacheKey, data, *modTime)
				sendJpeg(w, data)
			}
		}
	}
}

func drawOnWhite(size, srcOfs, dstOfs image.Point, input image.Image) image.Image {

	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	draw.Draw(img, img.Bounds().Add(dstOfs), input, input.Bounds().Min.Add(srcOfs), draw.Over)
	return img
}

func (api *ImgServerApi) uploadHandler(w traffic.ResponseWriter, r *traffic.Request) {

	filename := r.URL.Query().Get("image")
	uploadedImage, _, err := image.Decode(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		img := drawOnWhite(uploadedImage.Bounds().Size(), image.Pt(0, 0), image.Pt(0, 0), uploadedImage)
		err = api.imageDir.WriteImage(filename, img, 100)
		if err != nil {
			traffic.Logger().Print(err.Error())
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
	router.Delete("/:image", api.deleteHandler)
	router.Run()
}
