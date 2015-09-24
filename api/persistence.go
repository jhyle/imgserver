package imgserver

import (
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
	"time"
)

type (
	Directory interface {
		ListFiles(minAge time.Duration) ([]string, error)
		ReadFile(filename string) ([]byte, error)
		ReadImage(filename string) (image.Image, error)
		WriteFile(filename string, data []byte) error
		WriteImage(filename string, image image.Image, quality int) error
		DeleteFile(filename string) error
		GetBasePath() string
		GetFilePath(string) string
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

func (dir *fsDirectory) ListFiles(minAge time.Duration) ([]string, error) {

	fileInfos, err := ioutil.ReadDir(dir.basePath)
	if err != nil {
		return nil, err
	}
	
	modTime := time.Now().Add(-minAge)
	
	files := make([]string, 0, len(fileInfos))
	for _, fileInfo := range fileInfos {
		if !fileInfo.IsDir() && fileInfo.ModTime().Before(modTime) {
			files = append(files, fileInfo.Name())
		}
	}
	
	return files, nil
}

func (dir *fsDirectory) GetFilePath(filename string) string {

	return dir.basePath + string(os.PathSeparator) + filename
}

func (dir *fsDirectory) WriteFile(filename string, data []byte) error {

	return ioutil.WriteFile(dir.GetFilePath(filename), data, 0644)
}

func (dir *fsDirectory) WriteImage(filename string, image image.Image, quality int) error {

	file, err := os.Create(dir.GetFilePath(filename))
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

	return ioutil.ReadFile(dir.GetFilePath(filename))
}

func (dir *fsDirectory) ReadImage(filename string) (image.Image, error) {

	file, err := os.Open(dir.GetFilePath(filename))
	if err != nil {
		return nil, err
	}

	image, _, err := image.Decode(file)
	file.Close()

	return image, err
}

func (dir *fsDirectory) DeleteFile(filename string) error {

	return os.Remove(dir.GetFilePath(filename))
}
