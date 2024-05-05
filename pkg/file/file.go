package file

import (
	"fmt"
	"os"
)

type File interface {
	GetFile() (string, error)
}

type file struct {
	path string
}

func NewFileStruct(path string) File {
	return file{
		path: path,
	}
}

// GetFile implements File.
func (f file) GetFile() (string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
	  fmt.Println("File reading error", err)
	  return "", err
	}
	return string(data), err
}