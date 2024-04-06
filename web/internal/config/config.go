package config

import "flag"

var (
	Count            int
	Port             string
	Secure           bool
	Http3            bool
	ServerKey        string
	ServerCert       string
	ClientCertVerify bool
	IdeAuthorization string
)

func init() {
	// 获取启动参数
	flag.IntVar(&Count, "n", 1, "Count of virtual machines.") // 定义命令行参数 c，表示虚拟机的个数，返回 Int 类型指针，默认值为 1，其值在 Parse 后会被修改为命令参数指定的值
	flag.StringVar(&Port, "p", "8090", "Port to listen.")
	flag.BoolVar(&Secure, "s", false, "Enable https.")
	flag.BoolVar(&Http3, "3", false, "Enable http3.")
	flag.StringVar(&ServerKey, "k", "server.key", "SSL key file.")
	flag.StringVar(&ServerCert, "c", "server.crt", "SSL cert file.")
	flag.BoolVar(&ClientCertVerify, "v", false, "Enable client cert verification.")
	flag.StringVar(&IdeAuthorization, "a", "", "<username:password> for ide authorization verification.")

	// 在定义命令行参数之后，调用 Parse 方法对所有命令行参数进行解析
	flag.Parse()
}
