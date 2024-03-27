package module

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cube/internal/builtin"
)

func init() {
	register("file", func(worker Worker, db Db) interface{} {
		return &FileClient{}
	})
}

type FileClient struct{}

func (f *FileClient) getPath(name string) (string, error) {
	fp := path.Clean("files/" + name)
	if !strings.HasPrefix(fp+"/", "files/") {
		return "", errors.New("permission denial")
	}
	return fp, nil
}

func (f *FileClient) Read(name string) (builtin.Buffer, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(fp)
}

func (f *FileClient) ReadRange(name string, offset int64, length int64) (builtin.Buffer, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	file.Seek(offset, io.SeekStart) // 设置光标的位置：距离文件开头，offset 个字节处

	data := make([]byte, length)
	file.Read(data)

	return data, nil
}

func (f *FileClient) Write(name string, bytes []byte) error {
	fp, err := f.getPath(name)
	if err != nil {
		return err
	}

	paths, _ := filepath.Split(fp)
	os.MkdirAll(paths, os.ModePerm)
	return os.WriteFile(fp, bytes, 0o664)
}

func (f *FileClient) WriteRange(name string, offset int64, bytes []byte) error {
	fp, err := f.getPath(name)
	if err != nil {
		return err
	}

	file, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer file.Close()

	file.Seek(offset, io.SeekStart)

	file.Write(bytes)
	return nil
}

func (f *FileClient) Stat(name string) (fs.FileInfo, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	return os.Stat(fp)
}

func (f *FileClient) List(name string) ([]string, error) {
	fp, err := f.getPath(name)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fp)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}
