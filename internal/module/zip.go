package module

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"

	"cube/internal/builtin"
)

func init() {
	register("zip", func(worker Worker, db Db) interface{} {
		return &ZipClient{}
	})
}

type ZipClient struct{}

func (z *ZipClient) Write(data map[string]interface{}) (builtin.Buffer, error) {
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
			err = errors.New("type of value " + k + " is not supported")
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

type ZipEntry struct {
	*zip.File
}

func (z *ZipEntry) GetData() (builtin.Buffer, error) {
	fd, err := z.Open()
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return io.ReadAll(fd)
}

type ZipReader struct {
	reader *zip.Reader
}

func (z *ZipReader) GetEntries() (files []*ZipEntry) {
	for _, f := range z.reader.File {
		files = append(files, &ZipEntry{f})
	}
	return
}

func (z *ZipReader) GetData(name string) (builtin.Buffer, error) {
	fd, err := z.reader.Open(name)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return io.ReadAll(fd)
}
