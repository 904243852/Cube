package main

import (
	"crypto/tls"
	"crypto/x509"
	. "cube/internal"
	"cube/internal/handler"
	"embed"
	"flag"
	"fmt"
	"github.com/quic-go/quic-go/http3"
	"net/http"
	"os"
)

//go:embed web/*
var web embed.FS

func init() {
	// 初始化数据库
	InitDb()

	// 初始化日志文件
	InitLog()

	// 初始化缓存
	InitCache()
}

func main() {
	// 获取启动参数
	configs := parseStartupConfigs()

	// 静态页面
	handler.RunHandlers(&web)

	// 创建虚拟机池
	CreateWorkerPool(configs.Count)

	// 监控当前进程的内存和 cpu 使用率
	go RunMonitor()

	// 启动守护任务
	RunDaemons("")

	// 启动定时服务
	RunCrontabs("")

	// 启动服务
	if !configs.Secure { // 启用 HTTP
		fmt.Println("Server has started on http://127.0.0.1:" + configs.Port + " 🚀")
		http.ListenAndServe(":"+configs.Port, nil)
	} else {
		fmt.Println("Server has started on https://127.0.0.1:" + configs.Port + " 🚀")
		config := &tls.Config{
			ClientAuth: tls.RequestClientCert, // 可通过 request.TLS.PeerCertificates 获取客户端证书
		}
		if configs.ClientCertVerify { // 设置对服务端证书校验
			config.ClientAuth = tls.RequireAndVerifyClientCert
			b, _ := os.ReadFile("./ca.crt")
			config.ClientCAs = x509.NewCertPool()
			config.ClientCAs.AppendCertsFromPEM(b)
		}
		if configs.Http3 { // 启用 HTTP/3
			server := &http3.Server{
				Addr:      ":" + configs.Port,
				TLSConfig: config,
			}
			server.ListenAndServeTLS(configs.ServerCert, configs.ServerKey)
		} else { // 启用 HTTPS
			server := &http.Server{
				Addr:      ":" + configs.Port,
				TLSConfig: config,
			}
			server.ListenAndServeTLS(configs.ServerCert, configs.ServerKey)
		}
	}
}

func parseStartupConfigs() (a struct {
	Count            int
	Port             string
	Secure           bool
	Http3            bool
	ServerKey        string
	ServerCert       string
	ClientCertVerify bool
}) {
	flag.IntVar(&a.Count, "n", 1, "Total count of virtual machines.") // 定义命令行参数 c，表示虚拟机的总个数，返回 Int 类型指针，默认值为 1，其值在 Parse 后会被修改为命令参数指定的值
	flag.StringVar(&a.Port, "p", "8090", "Port to use.")
	flag.BoolVar(&a.Secure, "s", false, "Enable https.")
	flag.BoolVar(&a.Http3, "3", false, "Enable http3.")
	flag.StringVar(&a.ServerKey, "k", "server.key", "SSL key file.")
	flag.StringVar(&a.ServerCert, "c", "server.crt", "SSL cert file.")
	flag.BoolVar(&a.ClientCertVerify, "v", false, "Enable client cert verification.")
	flag.Parse() // 在定义命令行参数之后，调用 Parse 方法对所有命令行参数进行解析
	return
}
