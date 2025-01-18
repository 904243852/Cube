package builtin

import (
	"encoding/json"
	"github.com/dop251/goja"
	"io"
	"net/http"
	"strings"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		worker.Runtime().Set("fetch", func(url string, options *FetchOptions) (*goja.Promise, error) {
			if options == nil {
				options = &FetchOptions{
					Method: "GET",
				}
			}

			req, err := http.NewRequest(strings.ToUpper(options.Method), url, strings.NewReader(options.Body))
			if err != nil {
				return nil, err
			}
			for k, v := range options.Headers {
				req.Header.Set(k, v)
			}

			runtime := worker.Runtime()
			promise, resolve, reject := runtime.NewPromise()

			t := worker.EventLoop().NewEventTaskTrigger()
			t.AddTask(func() {
				c := &http.Client{}

				resp, err := c.Do(req)
				if err != nil {
					t.AddMicroTask(func() {
						reject(runtime.NewGoError(err))
						t.Cancel()
					})
					return
				}

				data, err := io.ReadAll(resp.Body)
				if err != nil {
					t.AddMicroTask(func() {
						reject(runtime.NewGoError(err))
						t.Cancel()
					})
					return
				}
				defer resp.Body.Close()

				headers := map[string]string{}
				for k, v := range resp.Header {
					headers[k] = v[0]
				}

				t.AddMicroTask(func() {
					resolve(&FetchResponse{
						Status:  resp.StatusCode,
						Headers: headers,
						data:    data,
					})
					t.Cancel()
				})
			})

			return promise, nil
		})
	})
}

type FetchOptions struct {
	Method  string
	Headers map[string]string
	Body    string
}

type FetchResponse struct {
	Status  int
	Headers map[string]string
	data    []byte
}

func (f *FetchResponse) Buffer() Buffer {
	return f.data
}

func (f *FetchResponse) Json() (v *map[string]interface{}, err error) {
	err = json.Unmarshal(f.data, v)
	return
}

func (f *FetchResponse) Text() string {
	return string(f.data)
}
