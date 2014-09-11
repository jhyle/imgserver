package imgserver

import (
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
)

type (
	Directory interface {
		ReadFile(filename string) ([]byte, error)
		ReadImage(filename string) (image.Image, error)
		WriteFile(filename string, data []byte) error
		WriteImage(filename string, image image.Image, quality int) error
		DeleteFile(filename string) error
		GetBasePath() string
	}

	fsDirectory struct {
		basePath string
	}
)

func NewFsDirectory(basePath string) Directory {

	return &fsDirectory{basePath}
}

func (dir *fsDirectory) GetBasePath() string {

	return dir.basePath
}

func (dir *fsDirectory) WriteFile(filename string, data []byte) error {

	return ioutil.WriteFile(dir.basePath + string(os.PathSeparator) + filename, data, 0644)
}

func (dir *fsDirectory) WriteImage(filename string, image image.Image, quality int) error {

	file, err := os.Create(dir.basePath + string(os.PathSeparator) + filename)
	if err != nil {
		return err
	}

	err = jpeg.Encode(file, image, &jpeg.Options{quality})
	file.Close()

	if err != nil {
		os.Remove(filename)
		return err
	}

	return nil
}

func (dir *fsDirectory) ReadFile(filename string) ([]byte, error) {

	return ioutil.ReadFile(dir.basePath + string(os.PathSeparator) + filename)
}

func (dir *fsDirectory) ReadImage(filename string) (image.Image, error) {

	file, err := os.Open(dir.basePath + string(os.PathSeparator) + filename)
	if err != nil {
		return nil, err
	}

	image, _, err := image.Decode(file)
	file.Close()

	return image, err
}

func (dir *fsDirectory) DeleteFile(filename string) error {

	return os.Remove(dir.basePath + string(os.PathSeparator) + filename)
}