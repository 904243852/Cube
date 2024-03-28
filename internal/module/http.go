package module

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"cube/internal/builtin"

	"github.com/quic-go/quic-go/http3"
)

type HttpOptions struct {
	CaCert             string
	Cert               string
	Key                string
	InsecureSkipVerify bool
	IsHttp3            bool
	Proxy              string
}

func init() {
	register("http", func(worker Worker, db Db) interface{} {
		return func(options *HttpOptions) (*HttpClient, error) {
			httpc := &HttpClient{c: &http.Client{}}

			if options == nil {
				return httpc, nil
			}

			cc := &tls.Config{}

			// 设置 ca 证书
			if options.CaCert != "" {
				cc.RootCAs = x509.NewCertPool()
				cc.RootCAs.AppendCertsFromPEM([]byte(options.CaCert))
			}

			// 设置客户端证书和密钥
			if options.Cert != "" || options.Key != "" {
				var err error
				var c tls.Certificate // 参考实现 https://github.com/sideshow/apns2/blob/HEAD/certificate/certificate.go

				bc, _ := pem.Decode([]byte(options.Cert)) // 读取证书
				if bc == nil {
					return nil, errors.New("public key not found")
				}
				c.Certificate = append(c.Certificate, bc.Bytes) // tls.Certificate 存储了一个证书链（类型为 [][]byte），包含一个或多个 x509.Certificate（类型为 []byte）

				bk, _ := pem.Decode([]byte(options.Key)) // 读取密钥
				if bk == nil {
					return nil, errors.New("private key not found")
				}
				c.PrivateKey, err = x509.ParsePKCS1PrivateKey(bk.Bytes) // 使用 PKCS#1 格式
				if err != nil {
					c.PrivateKey, err = x509.ParsePKCS8PrivateKey(bk.Bytes) // 使用 PKCS#8 格式
					if err != nil {
						return nil, errors.New("invalid private key")
					}
				}

				if a, err := x509.ParseCertificate(c.Certificate[0]); err == nil {
					c.Leaf = a
				}
				cc.Certificates = []tls.Certificate{c} // 配置客户端证书
			}

			// 设置是否忽略服务端证书错误
			if options.InsecureSkipVerify {
				cc.InsecureSkipVerify = options.InsecureSkipVerify
			}

			// 设置是否启用 HTTP/3
			if options.IsHttp3 {
				if options.Proxy != "" { // 暂不支持同时启用 HTTP/3 和配置代理
					return nil, errors.New("can not enable http3 and set proxy at the same time")
				}
				httpc.c.Transport = &http3.RoundTripper{
					TLSClientConfig: cc,
				}
			} else {
				t := &http.Transport{ // 创建 transport
					TLSClientConfig: cc,
				}
				// 设置代理服务器
				if options.Proxy != "" {
					u, _ := url.Parse(options.Proxy)
					t.Proxy = http.ProxyURL(u)
				}
				httpc.c.Transport = t
			}

			return httpc, nil
		}
	})
}

type HttpClient struct {
	c *http.Client
}

func (h *HttpClient) Request(method string, url string, header map[string]string, body string) (response interface{}, err error) {
	req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(body))
	if err != nil {
		return
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := h.c.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	headers := map[string]string{}
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	response = map[string]interface{}{
		"status": resp.StatusCode,
		"header": headers,
		"data":   (*builtin.Buffer)(&data),
	}
	return
}
