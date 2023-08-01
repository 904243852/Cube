package module

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
)

func init() {
	register("zip", func(worker Worker, db Db) interface{} {
		return &ZipClient{}
	})
}

type ZipClient struct{}

func (z *ZipClient) Write(data map[string]interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)

	w := zip.NewWriter(buf)

	for k, v := range data {
		f, err := w.Create(k)
		if err != nil {
			return nil, err
		}
		switch v := v.(type) {
		case string:
			_, err = f.Write([]byte(v))
		case []byte:
			_, err = f.Write(v)
		default:
			err = errors.New("Type of value " + k + " is not supported.")
		}
		if err != nil {
			return nil, err
		}
	}

	w.Close() // 必须在 buf.Bytes() 前关闭，否则 buf.Bytes() 返回空

	return buf.Bytes(), nil
}

func (z *ZipClient) Read(data []byte) (*ZipReader, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	return &ZipReader{reader: r}, nil
}

type ZipFile struct {
	file *zip.File
}

func (z *ZipFile) GetName() string {
	return z.file.Name
}

func (z *ZipFile) GetData() ([]byte, error) {
	fd, err := z.file.Open()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(fd)
}

type ZipReader struct {
	reader *zip.Reader
}

func (z *ZipReader) GetFiles() (files []*ZipFile) {
	for _, f := range z.reader.File {
		files = append(files, &ZipFile{f})
	}
	return
}
