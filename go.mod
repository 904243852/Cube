module cube

go 1.19

require (
	github.com/antchfx/htmlquery v1.3.0
	github.com/dop251/goja v0.0.0-20240731150404-c665f0b58f6e
	github.com/fogleman/gg v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/quic-go/quic-go v0.36.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/shopspring/decimal v1.3.1
)

require (
	github.com/antchfx/xpath v1.2.3 // indirect
	github.com/dlclark/regexp2 v1.11.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/onsi/ginkgo/v2 v2.11.0 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-19 v0.3.2 // indirect
	github.com/quic-go/qtls-go1-20 v0.2.2 // indirect; 这里需要降低至 0.2.2 版本，否则与 quic-go 0.36.0 版本在 go 1.20 下会出现兼容问题，如 "undefined: qtls.Alert"
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/image v0.15.0
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.25.0
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
)
