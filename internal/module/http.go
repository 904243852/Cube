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
	. "cube/internal/util"

	"github.com/quic-go/quic-go/http3"
)

func init() {
	register("http", func(worker Worker, db Db) interface{} {
		return func(options map[string]interface{}) (*HttpClient, error) {
			client := &http.Client{}
			var err error
			if options != nil {
				config := &tls.Config{}
				// 设置 ca 证书
				if caCert, ok := ExportMapValue(options, "caCert", "string"); ok { // 配置 ca 证书
					config.RootCAs = x509.NewCertPool()
					config.RootCAs.AppendCertsFromPEM([]byte(caCert.(string)))
				}
				// 设置客户端证书和密钥
				if cert, ok := ExportMapValue(options, "cert", "string"); ok {
					var c tls.Certificate                      // 参考实现 https://github.com/sideshow/apns2/blob/HEAD/certificate/certificate.go
					b1, _ := pem.Decode([]byte(cert.(string))) // 读取公钥
					if b1 == nil {
						return nil, errors.New("no public key found")
					}
					c.Certificate = append(c.Certificate, b1.Bytes) // tls.Certificate 存储了一个证书链（类型为 [][]byte），包含一个或多个 x509.Certificate（类型为 []byte）
					if key, ok := ExportMapValue(options, "key", "string"); ok {
						b2, _ := pem.Decode([]byte(key.(string))) // 读取私钥
						if b2 == nil {
							return nil, errors.New("no private key found")
						}
						c.PrivateKey, err = x509.ParsePKCS1PrivateKey(b2.Bytes) // 使用 PKCS#1 格式
						if err != nil {
							c.PrivateKey, err = x509.ParsePKCS8PrivateKey(b2.Bytes) // 使用 PKCS#8 格式
							if err != nil {
								return nil, errors.New("failed to parse private key")
							}
						}
					}
					if len(c.Certificate) == 0 || c.PrivateKey == nil {
						return nil, errors.New("no private key or public key found")
					}
					if a, err := x509.ParseCertificate(c.Certificate[0]); err == nil {
						c.Leaf = a
					}
					config.Certificates = []tls.Certificate{c} // 配置客户端证书
				}
				// 设置是否忽略服务端证书错误
				if insecureSkipVerify, ok := ExportMapValue(options, "insecureSkipVerify", "bool"); ok { // 忽略服务端证书校验
					config.InsecureSkipVerify = insecureSkipVerify.(bool)
				}
				// 设置是否启用 HTTP/3
				if v, ok := ExportMapValue(options, "isHttp3", "bool"); ok && v.(bool) {
					// 暂不支持同时启用 HTTP/3 和配置代理
					if _, ok := ExportMapValue(options, "proxy", "string"); ok {
						return nil, errors.New("can not enable http3 and set proxy at the same time")
					}
					client.Transport = &http3.RoundTripper{
						TLSClientConfig: config,
					}
				} else {
					// 创建 transport
					transport := &http.Transport{
						TLSClientConfig: config,
					}
					// 设置代理服务器
					if proxy, ok := ExportMapValue(options, "proxy", "string"); ok {
						u, _ := url.Parse(proxy.(string))
						transport.Proxy = http.ProxyURL(u)
					}
					client.Transport = transport
				}
			}
			return &HttpClient{
				client,
			}, nil
		}
	})
}

type HttpClient struct {
	client *http.Client
}

func (h *HttpClient) Request(method string, url string, header map[string]string, body string) (response interface{}, err error) {
	req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(body))
	if err != nil {
		return
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
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
