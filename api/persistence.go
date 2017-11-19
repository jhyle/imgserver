package imgserver

import (
	"io/ioutil"
	"os"
	"time"
)

type (
	Directory interface {
		ListFiles(minAge time.Duration) ([]string, error)
		ReadFile(filename string) ([]byte, error)
		WriteFile(filename string, data []byte) error
		DeleteFile(filename string) error
		GetBasePath() string
		GetFilePath(string) string
		ModTime(string) *time.Time
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

func (dir *fsDirectory) ReadFile(filename string) ([]byte, error) {

	return ioutil.ReadFile(dir.GetFilePath(filename))
}

func (dir *fsDirectory) ModTime(filename string) *time.Time {

	fileInfo, err := os.Stat(dir.GetFilePath(filename))
	if err == nil {
		modTime := fileInfo.ModTime()
		return &modTime
	} else {
		return nil
	}
}

func (dir *fsDirectory) DeleteFile(filename string) error {

	return os.Remove(dir.GetFilePath(filename))
}
