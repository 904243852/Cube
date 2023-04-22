package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/process"
	"github.com/shopspring/decimal"
	"html/template"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg" // 需要导入 "image/jpeg"、"image/gif"、"image/png" 去解码 jpg、gif、png 图片，否则当使用 image.Decode 处理图片文件时，会报 image: unknown format 错误
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Source struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // module, controller, daemon, crontab, template, resource
	Lang     string `json:"lang"` // typescript, html, text, vue
	Content  string `json:"content"`
	Compiled string `json:"compiled"`
	Active   bool   `json:"active"`
	Method   string `json:"method"`
	Url      string `json:"url"`
	Cron     string `json:"cron"`
	Status   string `json:"status"`
}

//go:embed index.html editor.html
var FileList embed.FS

var Database *sql.DB

var WorkerPool struct {
	Channels chan *Worker
	Workers  []*Worker
}

var Crontab *cron.Cron // 定时任务

var SourceCache *SourceCacheClient

func init() {
	// 初始化数据库
	if db, err := sql.Open("sqlite3", "./cube.db"); err != nil {
		panic(err)
	} else {
		db.Exec(`
			create table if not exists source (
				name varchar(64) not null,
				type varchar(16) not null,
				lang varchar(16) not null,
				content text not null,
				compiled text not null default '',
				active boolean not null default false,
				method varchar(8) not null default '',
				url varchar(64) not null default '',
				cron varchar(16) not null default '',
				primary key(name, type)
			);
		`)
		Database = db
	}

	// 初始化日志文件
	if fd, err := os.OpenFile("./cube.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
		panic(err)
	} else {
		log.SetOutput(fd)
	}

	// 初始化缓存
	SourceCache = &SourceCacheClient{
		controllers: make(map[string]*Source),
		crontabs:    make(map[string]cron.EntryID),
		daemons:     make(map[string]*Worker),
		modules:     make(map[string]*goja.Program),
	}
}

func main() {
	// 获取启动参数
	arguments := ParseStartupArguments()

	// 创建虚拟机池
	CreateWorkerPool(arguments.Count)

	http.HandleFunc("/source", func(w http.ResponseWriter, r *http.Request) {
		var (
			data interface{}
			err  error
		)
		switch r.Method {
		case http.MethodPost:
			err = HandleSourcePost(w, r)
		case http.MethodDelete:
			err = HandleSourceDelete(w, r)
		case http.MethodPatch:
			err = HandleSourcePatch(w, r)
		case http.MethodGet:
			data, err = HandleSourceGet(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err != nil {
			Error(w, err)
			return
		}
		Success(w, data)
	})
	http.HandleFunc("/service/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/service/")

		// 查询 controller
		source := SourceCache.GetController(name)
		if source == nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if source.Method != "" && source.Method != r.Method { // 校验请求方法
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 获取 vm 实例
		worker := <-WorkerPool.Channels
		defer func() {
			WorkerPool.Channels <- worker
		}()

		// 允许最大执行的时间为 60 秒
		timer := time.AfterFunc(60000*time.Millisecond, func() {
			worker.Interrupt("The service executed timeout.")
		})
		defer timer.Stop()

		ctx := ServiceContext{
			request:        r,
			responseWriter: w,
			timer:          timer,
		}

		// 执行
		value, err := worker.Run(
			worker.Runtime.ToValue("./controller/"+source.Name),
			worker.Runtime.ToValue(&ctx),
		)
		worker.ClearHandle()
		if err != nil {
			Error(w, err)
			return
		}

		if ctx.returnless == true { // 如果是 WebSocket 或 chunk 响应，不需要封装响应
			return
		}

		Success(w, ExportGojaValue(value))
	})
	http.HandleFunc("/resource/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/resource/")

		var content string
		if err := Database.QueryRow("select content from source where url = ? and type = 'resource' and active = true", name).Scan(&content); err != nil {
			Error(w, err)
			return
		}
		Success(w, content)
	})
	http.Handle("/", http.FileServer(http.FS(FileList)))

	// 监控当前进程的内存和 cpu 使用率
	go func() {
		p, _ := process.NewProcess(int32(os.Getppid()))
		ticker := time.NewTicker(time.Millisecond * 1000)
		for range ticker.C {
			c, _ := p.CPUPercent()
			m, _ := p.MemoryInfo()
			fmt.Printf("\rcpu: %.2f%%, memory: %.2fmb, vm: %d/%d"+" ", // 结尾预留一个空格防止刷新过程中因字符串变短导致上一次打印的文本在结尾出溢出
				c,
				float32(m.RSS)/1024/1024,
				len(WorkerPool.Workers)-len(WorkerPool.Channels), len(WorkerPool.Workers),
			)
		}
	}()

	// 启动守护任务
	RunDaemons("")

	// 启动定时服务
	RunCrontabs("")

	// 启动服务
	if !arguments.Secure {
		fmt.Println("Server has started on http://127.0.0.1:" + arguments.Port + " 🚀")
		http.ListenAndServe(":"+arguments.Port, nil)
	} else {
		fmt.Println("Server has started on https://127.0.0.1:" + arguments.Port + " 🚀")
		config := &tls.Config{
			ClientAuth: tls.RequestClientCert, // 可通过 request.TLS.PeerCertificates 获取客户端证书
		}
		if arguments.ClientCertVerify { // 设置对服务端证书校验
			config.ClientAuth = tls.RequireAndVerifyClientCert
			b, _ := ioutil.ReadFile("./ca.crt")
			config.ClientCAs = x509.NewCertPool()
			config.ClientCAs.AppendCertsFromPEM(b)
		}
		server := &http.Server{
			Addr:      ":" + arguments.Port,
			TLSConfig: config,
		}
		server.ListenAndServeTLS(arguments.ServerCert, arguments.ServerKey)
	}
}

func ParseStartupArguments() (a struct {
	Count            int
	Port             string
	Secure           bool
	ServerKey        string
	ServerCert       string
	ClientCertVerify bool
}) {
	flag.IntVar(&a.Count, "n", 1, "Total count of virtual machines.") // 定义命令行参数 c，表示虚拟机的总个数，返回 Int 类型指针，默认值为 1，其值在 Parse 后会被修改为命令参数指定的值
	flag.StringVar(&a.Port, "p", "8090", "Port to use.")
	flag.BoolVar(&a.Secure, "s", false, "Enable https.")
	flag.StringVar(&a.ServerKey, "k", "server.key", "SSL key file.")
	flag.StringVar(&a.ServerCert, "c", "server.crt", "SSL cert file.")
	flag.BoolVar(&a.ClientCertVerify, "v", false, "Enable client cert verification.")
	flag.Parse() // 在定义命令行参数之后，调用 Parse 方法对所有命令行参数进行解析
	return
}

func ExportMapValue(obj map[string]interface{}, name string, t string) (value interface{}, success bool) {
	if obj == nil {
		return
	}
	if o, k := obj[name]; k {
		switch t {
		case "string":
			value, success = o.(string)
		case "bool":
			value, success = o.(bool)
		case "int":
			value, success = o.(int)
		default:
			panic(errors.New("Type " + t + " is not supported."))
		}
	}
	return
}

func Success(w http.ResponseWriter, data interface{}) {
	switch v := data.(type) {
	case string:
		fmt.Fprintf(w, "%s", v)
	case []uint8: // []byte
		w.Write(v)
	case *ServiceResponse: // 自定义响应
		h := w.Header()
		for k, a := range v.header {
			h.Set(k, a)
		}
		w.WriteHeader(v.status) // WriteHeader 必须在 Set Header 之后调用，否则状态码将无法写入
		w.Write(v.data)         // Write 必须在 WriteHeader 之后调用，否则将会抛出异常 http: superfluous response.WriteHeader call from ...
	default: // map[string]interface[]
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false) // 见 https://pkg.go.dev/encoding/json#Marshal，字符串值编码为强制为有效 UTF-8 的 JSON 字符串，用 Unicode 替换符文替换无效字节。为了使 JSON 能够安全地嵌入 HTML 标记中，字符串使用 HTMLEscape 编码，它将替换 `<`、`>`、`&`、`U+2028` 和 `U+2029`，并转义到 `\u003c`、`\u003e`、`\u0026`、`\u2028` 和 `\u2029`。在使用编码器时，可以通过调用 SetEscapeHTML(false) 禁用此替换。
		enc.Encode(map[string]interface{}{
			"code":    "0",
			"message": "success",
			"data":    v, // 注：这里的 data 如果为 []byte 类型或包含 []byte 类型的属性，在通过 json 序列化后将会被自动转码为 base64 字符串
		})
	}
}

func Error(w http.ResponseWriter, err error) {
	code, message := "1", err.Error() // 错误信息默认包含了异常信息和调用栈
	if e, ok := err.(*goja.Exception); ok {
		if o, ok := e.Value().Export().(map[string]interface{}); ok {
			if m, ok := ExportMapValue(o, "message", "string"); ok {
				message = m.(string) // 获取 throw 对象中的 message 和 code 属性，作为失败响应的错误信息和错误码
			}
			if c, ok := ExportMapValue(o, "code", "string"); ok {
				code = c.(string)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest) // 在同一次请求响应过程中，只能调用一次 WriteHeader，否则会抛出异常 http: superfluous response.WriteHeader call from ...
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

//#region 守护任务

func RunDaemons(name string) {
	if name == "" {
		name = "%"
	}

	rows, err := Database.Query("select name from source where name like ? and type = 'daemon' and active = true", name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		rows.Scan(&n)

		if SourceCache.daemons[n] != nil { // 防止重复执行
			continue
		}

		go func() {
			worker := <-WorkerPool.Channels
			defer func() {
				WorkerPool.Channels <- worker
			}()

			SourceCache.daemons[n] = worker

			worker.Run(worker.Runtime.ToValue("./daemon/" + n))

			worker.Runtime.ClearInterrupt()

			delete(SourceCache.daemons, n)
		}()
	}
}

//#endregion

//#region 定时服务

func RunCrontabs(name string) {
	if Crontab == nil { // 首次执行时，先初始化 Crontab
		Crontab = cron.New()
		Crontab.Start()
	}

	if name == "" {
		name = "%"
	}

	rows, err := Database.Query("select name, cron from source where name like ? and type = 'crontab' and active = true", name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n, c string
		rows.Scan(&n, &c)

		if _, ok := SourceCache.crontabs[n]; ok { // 防止重复添加任务
			continue
		}

		id, err := Crontab.AddFunc(c, func() {
			worker := <-WorkerPool.Channels
			defer func() {
				WorkerPool.Channels <- worker
			}()

			worker.Run(worker.Runtime.ToValue("./crontab/" + n))
		})
		if err != nil {
			panic(err)
		} else {
			SourceCache.crontabs[n] = id
		}
	}
}

//#endregion

//#region Source 接口请求

func HandleSourceGet(w http.ResponseWriter, r *http.Request) (data struct {
	Sources []Source `json:"sources"`
	Total   int      `json:"total"`
}, err error) {
	r.ParseForm()
	name := r.Form.Get("name")
	stype := r.Form.Get("type")
	if stype == "" {
		stype = "%"
	}
	from := r.Form.Get("from")
	if from == "" {
		from = "0"
	}
	size := r.Form.Get("size")
	if size == "" {
		size = "10"
	}

	if err = Database.QueryRow("select count(1) from source where name like ? and type like ?", "%"+name+"%", stype).Scan(&data.Total); err != nil { // 调用 QueryRow 方法后，须调用 Scan 方法，否则连接将不会被释放
		return
	}

	rows, err := Database.Query("select name, type, lang, content, compiled, active, method, url, cron from source where name like ? and type like ? order by rowid desc limit ?, ?", "%"+name+"%", stype, from, size)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		source := Source{}
		rows.Scan(&source.Name, &source.Type, &source.Lang, &source.Content, &source.Compiled, &source.Active, &source.Method, &source.Url, &source.Cron)
		if source.Type == "daemon" {
			source.Status = fmt.Sprintf("%v", SourceCache.daemons[source.Name] != nil)
		}
		data.Sources = append(data.Sources, source)
	}
	err = rows.Err()
	return
}

func HandleSourcePost(w http.ResponseWriter, r *http.Request) error {
	// 读取请求消息体
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return err
	}

	if _, bulk := r.URL.Query()["bulk"]; !bulk {
		// 转换为 source 对象
		var source Source
		if err = json.Unmarshal(body, &source); err != nil {
			return err
		}

		// 校验类型
		if ok, _ := regexp.MatchString("^(module|controller|daemon|crontab|template|resource)$", source.Type); !ok {
			return errors.New("The type of the source is required. It must be module, controller, daemon, crontab, template or resource.")
		}
		// 校验名称
		if source.Type == "module" {
			if ok, _ := regexp.MatchString("^(node_modules/)?\\w{2,32}$", source.Name); !ok {
				return errors.New("The name of the module is required. It must be a letter, number or underscore with a length of 2 to 32. It can also start with 'node_modules/'.")
			}
		} else {
			if ok, _ := regexp.MatchString("^\\w{2,32}$", source.Name); !ok {
				return errors.New("The name of the " + source.Type + " is required. It must be a letter, number, or underscore with a length of 2 to 32.")
			}
		}

		// 单个新增或修改，新增的均为去激活状态，无需刷新缓存
		if _, err := Database.Exec(strings.Join([]string{
			"update source set content = ?, compiled = ? where name = ? and type = ?",                          // 先尝试更新，再尝试新增
			"insert or ignore into source (name, type, lang, content, compiled, url) values(?, ?, ?, ?, ?, ?)", // 这里不用 insert or replace，replace 是替换整条记录
		}, ";"), source.Content, source.Compiled, source.Name, source.Type, source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Name); err != nil {
			return err
		}

		// 清空 module 缓存以重建
		SourceCache.modules = make(map[string]*goja.Program)
	} else { // 批量导入
		// 将请求入参转换为 source 对象数组
		var sources []Source
		if err = json.Unmarshal(body, &sources); err != nil {
			return err
		}

		if len(sources) == 0 {
			return errors.New("Nothing added or modified.")
		}

		// 批量新增或修改
		stmt, err := Database.Prepare("insert or replace into source (name, type, lang, content, compiled, active, method, url, cron) values(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, source := range sources {
			if _, err = stmt.Exec(source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Active, source.Method, source.Url, source.Cron); err != nil {
				return err
			}
		}

		// 批量导入后，需要清空 module 缓存以重建
		SourceCache.modules = make(map[string]*goja.Program)
		// 启动守护任务
		RunDaemons("")
		// 启动定时任务
		RunCrontabs("")
	}

	return nil
}

func HandleSourceDelete(w http.ResponseWriter, r *http.Request) error {
	r.ParseForm()
	name := r.Form.Get("name")
	if name == "" {
		return errors.New("The parameter name is required.")
	}
	stype := r.Form.Get("type")
	if stype == "" {
		return errors.New("The parameter type is required.")
	}

	res, err := Database.Exec("delete from source where name = ? and type = ?", name, stype)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("The source is not found.")
	}

	return nil
}

func HandleSourcePatch(w http.ResponseWriter, r *http.Request) error {
	// 读取请求消息体
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return err
	}
	// 转换为 source 对象
	var source Source
	if err = json.Unmarshal(body, &source); err != nil {
		return err
	}

	if source.Type == "controller" || source.Type == "resource" {
		// 校验 url 不能重复
		var count int
		if err = Database.QueryRow("select count(1) from source where type = ? and url = ? and active = true and name != ?", source.Type, source.Url, source.Name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("The url is already existed.")
		}
	}

	// 修改
	res, err := Database.Exec("update source set active = ?, method = ?, url = ?, cron = ? where name = ? and type = ?", source.Active, source.Method, source.Url, source.Cron, source.Name, source.Type)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("The source is not found.")
	}

	// 清空 module 缓存以重建
	SourceCache.modules = make(map[string]*goja.Program)
	// 如果是 daemon，需要启动或停止
	if source.Type == "daemon" {
		if source.Active {
			if SourceCache.daemons[source.Name] == nil && source.Status == "true" {
				RunDaemons(source.Name)
			}
			if SourceCache.daemons[source.Name] != nil && source.Status == "false" {
				SourceCache.daemons[source.Name].Interrupt("Daemon stopped.")
			}
		}
	}
	// 如果是 crontab，需要启动或停止
	if source.Type == "crontab" {
		id, ok := SourceCache.crontabs[source.Name]
		if !ok && source.Active {
			RunCrontabs(source.Name)
		}
		if ok && !source.Active {
			Crontab.Remove(id)
			delete(SourceCache.crontabs, source.Name)
		}
	}

	return nil
}

//#endregion

//#region Goja 运行时

func CreateWorker(program *goja.Program) *Worker {
	runtime := goja.New()

	entry, err := runtime.RunProgram(program) // 这里使用 RunProgram，可复用已编译的代码，相比直接调用 RunString 更显著提升性能
	if err != nil {
		panic(err)
	}
	function, ok := goja.AssertFunction(entry)
	if !ok {
		panic(errors.New("The program is not a function."))
	}

	worker := Worker{Runtime: runtime, function: function, handles: make([]interface{}, 0)}

	runtime.Set("require", func(id string) (goja.Value, error) {
		program := SourceCache.modules[id]
		if program == nil { // 如果已被缓存，直接从缓存中获取
			// 获取名称、类型
			var name, stype string
			if strings.HasPrefix(id, "./controller/") {
				name, stype = id[13:], "controller"
			} else if strings.HasPrefix(id, "./daemon/") {
				name, stype = id[9:], "daemon"
			} else if strings.HasPrefix(id, "./crontab/") {
				name, stype = id[10:], "crontab"
			} else if strings.HasPrefix(id, "./") {
				name, stype = path.Clean(id), "module"
			} else { // 如果没有 "./" 前缀，则视为 node_modules
				name, stype = "node_modules/"+id, "module"
			}

			// 根据名称查找源码
			var src string
			if err := Database.QueryRow("select compiled from source where name = ? and type = ? and active = true", name, stype).Scan(&src); err != nil {
				return nil, err
			}
			// 编译
			parsed, err := goja.Parse(
				name,
				"(function(exports, require, module) {"+src+"\n})",
				parser.WithSourceMapLoader(func(p string) ([]byte, error) {
					return []byte(src), nil
				}),
			)
			if err != nil {
				return nil, err
			}
			program, err = goja.CompileAST(parsed, false)
			if err != nil {
				return nil, err
			}

			// 缓存当前 module 的 program
			// 这里不应该直接缓存 module，因为 module 依赖当前 vm 实例，在开启多个 vm 实例池的情况下，调用会错乱从而导致异常 "TypeError: Illegal runtime transition of an Object at ..."
			SourceCache.modules[id] = program
		}

		exports := runtime.NewObject()
		module := runtime.NewObject()
		module.Set("exports", exports)

		// 运行
		entry, err := runtime.RunProgram(program)
		if err != nil {
			return nil, err
		}
		if function, ok := goja.AssertFunction(entry); ok {
			_, err = function(
				exports,                // this
				exports,                // exports
				runtime.Get("require"), // require
				module,                 // module
			)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("The entry is not a function.")
		}

		return module.Get("exports"), nil
	})

	runtime.Set("exports", runtime.NewObject())

	runtime.Set("ServiceResponse", func(call goja.ConstructorCall) *goja.Object { // 内置构造器不能同时返回 error 类型，否则将会失效
		a0, ok := call.Argument(0).Export().(int64)
		if !ok {
			panic(errors.New("Invalid argument status, not a int."))
		}
		a1, ok := call.Argument(1).Export().(map[string]interface{})
		if !ok {
			panic(errors.New("Invalid argument header, not a map."))
		}
		header := make(map[string]string, len(a1))
		for k, v := range a1 {
			if s, ok := v.(string); !ok {
				panic(errors.New("Invalid argument " + k + ", not a string."))
			} else {
				header[k] = s
			}
		}
		data := []byte(nil)
		if a2 := ExportGojaValue(call.Argument(2)); a2 != nil {
			if s, ok := a2.(string); !ok {
				if data, ok = a2.([]byte); !ok {
					panic(errors.New("The data should be a string or a byte array."))
				}
			} else {
				data = []byte(s)
			}
		}
		i := &ServiceResponse{
			status: int(a0),
			header: header,
			data:   data,
		}
		iv := runtime.ToValue(i).(*goja.Object)
		iv.SetPrototype(call.This.Prototype())
		return iv
	})

	runtime.SetFieldNameMapper(goja.UncapFieldNameMapper()) // 该转换器会将 go 对象中的属性、方法以小驼峰式命名规则映射到 js 对象中
	runtime.Set("console", &ConsoleClient{runtime: runtime})

	runtime.Set("$native", func(name string) (module interface{}, err error) {
		switch name {
		case "base64":
			module = &Base64Client{}
		case "bqueue":
			module = func(size int) *BlockingQueueClient {
				return &BlockingQueueClient{
					queue: make(chan interface{}, size),
				}
			}
		case "cache":
			module = &CacheClient{}
		case "crypto":
			module = &CryptoClient{}
		case "db":
			module = &DatabaseClient{worker: &worker}
		case "decimal":
			module = func(value string) (decimal.Decimal, error) {
				return decimal.NewFromString(value)
			}
		case "email":
			module = func(host string, port int, username string, password string) *EmailClient {
				return &EmailClient{
					host:     host,
					port:     port,
					username: username,
					password: password,
				}
			}
		case "file":
			module = &FileClient{}
		case "http":
			module = func(options map[string]interface{}) (*HttpClient, error) {
				client := &http.Client{}
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
							return nil, errors.New("No public key found.")
						}
						c.Certificate = append(c.Certificate, b1.Bytes) // tls.Certificate 存储了一个证书链（类型为 [][]byte），包含一个或多个 x509.Certificate（类型为 []byte）
						if key, ok := ExportMapValue(options, "key", "string"); ok {
							b2, _ := pem.Decode([]byte(key.(string))) // 读取私钥
							if b2 == nil {
								return nil, errors.New("No private key found.")
							}
							c.PrivateKey, err = x509.ParsePKCS1PrivateKey(b2.Bytes) // 使用 PKCS#1 格式
							if err != nil {
								c.PrivateKey, err = x509.ParsePKCS8PrivateKey(b2.Bytes) // 使用 PKCS#8 格式
								if err != nil {
									return nil, errors.New("Failed to parse private key.")
								}
							}
						}
						if len(c.Certificate) == 0 || c.PrivateKey == nil {
							return nil, errors.New("No private key or public key found.")
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
				return &HttpClient{
					client: client,
				}, nil
			}
		case "image":
			module = &ImageClient{}
		case "lock":
			module = func(name string) *LockClient {
				LockCache.Lock()
				defer LockCache.Unlock()
				if LockCache.clients == nil {
					LockCache.clients = make(map[string]*LockClient)
				}
				client := LockCache.clients[name]
				if client == nil {
					var mutex sync.Mutex
					client = &LockClient{
						name:   &name,
						mutex:  &mutex,
						locked: new(bool),
					}
					LockCache.clients[name] = client
				}
				worker.AddHandle(client)
				return client
			}
		case "pipe":
			module = func(name string) *BlockingQueueClient {
				if PipeCache == nil {
					PipeCache = make(map[string]*BlockingQueueClient, 99)
				}
				if PipeCache[name] == nil {
					PipeCache[name] = &BlockingQueueClient{
						queue: make(chan interface{}, 99),
					}
				}
				return PipeCache[name]
			}
		case "socket":
			module = &Socket{worker: &worker}
		case "template":
			module = func(name string, input map[string]interface{}) (string, error) {
				var content string
				if err := Database.QueryRow("select content from source where name = ? and type = 'template'", name).Scan(&content); err != nil {
					return "", err
				}

				if t, err := template.New(name).Parse(content); err != nil {
					return "", err
				} else {
					buf := new(bytes.Buffer)
					t.Execute(buf, input)
					return buf.String(), nil
				}
			}
		case "ulid":
			module = CreateULID
		default:
			err = errors.New("The module is not found.")
		}
		return
	})

	runtime.SetMaxCallStackSize(2048)

	return &worker
}

func CreateWorkerPool(count int) {
	WorkerPool.Workers = make([]*Worker, count) // 创建 goja 实例池
	WorkerPool.Channels = make(chan *Worker, count)

	// 编译程序
	program, _ := goja.Compile("index", "(function (id, ...params) { return require(id).default(...params); })", false) // 编译源码为 Program，strict 为 false

	for i := 0; i < count; i++ {
		worker := CreateWorker(program) // 创建 goja 运行时

		WorkerPool.Workers[i] = worker
		WorkerPool.Channels <- worker
	}
}

func ExportGojaValue(value goja.Value) interface{} {
	if o, ok := value.(*goja.Object); ok {
		if b, ok := o.Export().(goja.ArrayBuffer); ok { // 如果返回值为 ArrayBuffer 类型，则转换为 []byte
			return b.Bytes()
		}
		if "Uint8Array" == o.Get("constructor").(*goja.Object).Get("name").String() { // 如果返回值为 Uint8Array 类型，则转换为 []byte
			return o.Get("buffer").Export().(goja.ArrayBuffer).Bytes()
		}
	}
	return value.Export()
}

//#region worker

type Worker struct {
	Runtime  *goja.Runtime
	function goja.Callable
	handles  []interface{}
}

func (this *Worker) Run(params ...goja.Value) (goja.Value, error) {
	return this.function(nil, params...)
}
func (this *Worker) AddHandle(handle interface{}) {
	this.handles = append(this.handles, handle)
}
func (this *Worker) Interrupt(reason string) {
	this.Runtime.Interrupt(reason)
	this.ClearHandle()
}
func (this *Worker) ClearHandle() {
	for _, v := range this.handles {
		if l, ok := v.(*net.Listener); ok { // 如果已存在监听端口服务，这里需要先关闭，否则将导致 goja.Runtime.Interrupt 无法关闭
			(*l).Close()
		}
		if l, ok := v.(*LockClient); ok {
			(*l).Unlock()
		}
		if t, ok := v.(*sql.Tx); ok {
			(*t).Rollback()
		}
	}
	if len(this.handles) > 0 {
		this.handles = make([]interface{}, 0) // 清空所有句柄
	}
}

//#endregion

//#region source cache

type SourceCacheClient struct {
	controllers map[string]*Source
	crontabs    map[string]cron.EntryID
	daemons     map[string]*Worker
	modules     map[string]*goja.Program
}

func (this *SourceCacheClient) GetController(id string) *Source {
	source := this.controllers[id]
	if source == nil {
		source = &Source{}
		if err := Database.QueryRow("select name, method from source where url = ? and type = 'controller' and active = true", id).Scan(&source.Name, &source.Method); err != nil {
			return nil
		}
		this.controllers[id] = source
	}
	return source
}

//#endregion

//#endregion

//#region Service 请求、响应

type ServiceContextReader struct {
	reader *bufio.Reader
}

func (this *ServiceContextReader) Read(count int) ([]byte, error) {
	buf := make([]byte, count)
	_, err := this.reader.Read(buf)
	if err == io.EOF {
		return nil, nil
	}
	return buf, err
}
func (this *ServiceContextReader) ReadByte() (interface{}, error) {
	b, err := this.reader.ReadByte() // 如果是 chunk 传输，该方法不会返回 chunk size 和 "\r\n"，而是按 chunk data 到达顺序依次读取每个 chunk data 中的每个字节，如果已到达的 chunk 已读完且下一个 chunk 未到达，该方法将阻塞
	if err == io.EOF {
		return -1, nil
	}
	return b, err
}

// service http context
type ServiceContext struct {
	request        *http.Request
	responseWriter http.ResponseWriter
	timer          *time.Timer
	returnless     bool
	body           interface{} // 用于缓存请求消息体，防止重复读取和关闭 body 流
}

func (this *ServiceContext) GetHeader() map[string]string {
	var headers = make(map[string]string)
	for name, values := range this.request.Header {
		for _, value := range values {
			headers[name] = value
		}
	}
	return headers
}
func (this *ServiceContext) GetURL() interface{} {
	u := this.request.URL

	var params = make(map[string][]string)
	for name, values := range u.Query() {
		params[name] = values
	}

	return map[string]interface{}{
		"path":   u.Path,
		"params": params,
	}
}
func (this *ServiceContext) GetBody() ([]byte, error) {
	if this.body != nil {
		return this.body.([]byte), nil
	}
	defer this.request.Body.Close()
	return ioutil.ReadAll(this.request.Body)
}
func (this *ServiceContext) GetJsonBody() (interface{}, error) {
	bytes, err := this.GetBody()
	if err != nil {
		return nil, err
	}
	return this.body, json.Unmarshal(bytes, &this.body)
}
func (this *ServiceContext) GetMethod() string {
	return this.request.Method
}
func (this *ServiceContext) GetForm() interface{} {
	this.request.ParseForm() // 需要转换后才能获取表单

	var params = make(map[string][]string)
	for name, values := range this.request.Form {
		params[name] = values
	}

	return params
}
func (this *ServiceContext) GetFile(name string) (interface{}, error) {
	file, header, err := this.request.FormFile(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name": header.Filename,
		"size": header.Size,
		"data": data,
	}, nil
}
func (this *ServiceContext) GetCerts() interface{} { // 获取客户端证书
	return this.request.TLS.PeerCertificates
}
func (this *ServiceContext) UpgradeToWebSocket() (*ServiceWebSocket, error) {
	this.returnless = true // upgrader.Upgrade 内部已经调用过 WriteHeader 方法了，后续不应再次调用，否则将会出现 http: superfluous response.WriteHeader call from ... 的异常
	this.timer.Stop()      // 关闭定时器，WebSocket 不需要设置超时时间
	upgrader := websocket.Upgrader{}
	if conn, err := upgrader.Upgrade(this.responseWriter, this.request, nil); err != nil {
		return nil, err
	} else {
		return &ServiceWebSocket{
			connection: conn,
		}, nil
	}
}
func (this *ServiceContext) GetReader() *ServiceContextReader {
	return &ServiceContextReader{
		reader: bufio.NewReader(this.request.Body),
	}
}
func (this *ServiceContext) GetPusher() (http.Pusher, error) {
	pusher, ok := this.responseWriter.(http.Pusher)
	if !ok {
		return nil, errors.New("The server side push is not supported.")
	}
	return pusher, nil
}
func (this *ServiceContext) Write(data []byte) (int, error) {
	return this.responseWriter.Write(data)
}
func (this *ServiceContext) Flush() error {
	flusher, ok := this.responseWriter.(http.Flusher)
	if !ok {
		return errors.New("Failed to get a http flusher.")
	}
	if !this.returnless {
		this.returnless = true
		this.responseWriter.Header().Set("X-Content-Type-Options", "nosniff") // https://stackoverflow.com/questions/18337630/what-is-x-content-type-options-nosniff
	}
	flusher.Flush() // 改操作将自动设置响应头 Transfer-Encoding: chunked，并发送一个 chunk
	return nil
}

// service http response
type ServiceResponse struct {
	status int
	header map[string]string
	data   []byte
}

func (this *ServiceResponse) SetStatus(status int) { // 设置响应状态码
	this.status = status
}
func (this *ServiceResponse) SetHeader(header map[string]string) { // 设置响应消息头
	this.header = header
}
func (this *ServiceResponse) SetData(data []byte) { // 设置响应消息体
	this.data = data
}

// service websocket
type ServiceWebSocket struct {
	connection *websocket.Conn
}

func (this *ServiceWebSocket) Read() (interface{}, error) {
	messageType, data, err := this.connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"messageType": messageType,
		"data":        data,
	}, nil
}
func (this *ServiceWebSocket) Send(data []byte) error {
	return this.connection.WriteMessage(1, data) // message type：0 表示消息是文本格式，1 表示消息是二进制格式。这里 data 是 []byte，因此固定使用二进制格式类型
}
func (this *ServiceWebSocket) Close() {
	this.connection.Close()
}

//#endregion

//#region 内置模块

// base64 module
type Base64Client struct{}

func (this *Base64Client) Encode(input []byte) string { // 在 js 中调用该方法时，入参可接受 string 或 Uint8Array 类型
	return base64.StdEncoding.EncodeToString(input)
}
func (this *Base64Client) Decode(input string) ([]byte, error) { // 返回的 []byte 类型将隐式地转换为 js/ts 中的 Uint8Array 类型
	return base64.StdEncoding.DecodeString(input)
}

// blocking queue module
type BlockingQueueClient struct {
	queue chan interface{}
	sync.Mutex
}

func (this *BlockingQueueClient) Put(input interface{}, timeout int) error {
	this.Lock()
	defer this.Unlock()
	select {
	case this.queue <- input:
		return nil
	case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列入列最大超时时间为 timeout 毫秒
		return errors.New("The blocking queue is full, waiting for put timeout.")
	}
}
func (this *BlockingQueueClient) Poll(timeout int) (interface{}, error) {
	this.Lock()
	defer this.Unlock()
	select {
	case output := <-this.queue:
		return output, nil
	case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列出列最大超时时间为 timeout 毫秒
		return nil, errors.New("The blocking queue is empty, waiting for poll timeout.")
	}
}
func (this *BlockingQueueClient) Drain(size int, timeout int) (output []interface{}) {
	this.Lock()
	defer this.Unlock()
	output = make([]interface{}, 0, size) // 创建切片，初始大小为 0，最大为 size
	c := make(chan int, 1)
	go func(ch chan int) {
		for i := 0; i < size; i++ {
			output = append(output, <-this.queue)
		}
		ch <- 0
	}(c)
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	select {
	case <-c:
	case <-timer.C: // 定时器也是一个通道
	}
	return
}

// cache module
var Cache sync.Map // 存放并发安全的 map

type CacheClient struct{}

func (this *CacheClient) Set(key interface{}, value interface{}, timeout int) {
	Cache.Store(key, value)
	time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		Cache.Delete(key)
	})
}
func (this *CacheClient) Get(key interface{}) interface{} {
	if value, ok := Cache.Load(key); ok {
		return value
	}
	return nil
}

// console module
type ConsoleClient struct {
	runtime *goja.Runtime
}

func (this *ConsoleClient) Log(a ...interface{}) {
	log.Println(append([]interface{}{"\r" + time.Now().Format("2006-01-02 15:04:05.000"), &this.runtime, "Log"}, a...)...)
}
func (this *ConsoleClient) Debug(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[1;30m" + time.Now().Format("2006-01-02 15:04:05.000"), &this.runtime, "Debug"}, a...), "\033[m")...)
}
func (this *ConsoleClient) Info(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[0;34m" + time.Now().Format("2006-01-02 15:04:05.000"), &this.runtime, "Info"}, a...), "\033[m")...)
}
func (this *ConsoleClient) Warn(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[0;33m" + time.Now().Format("2006-01-02 15:04:05.000"), &this.runtime, "Warn"}, a...), "\033[m")...)
}
func (this *ConsoleClient) Error(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[0;31m" + time.Now().Format("2006-01-02 15:04:05.000"), &this.runtime, "Error"}, a...), "\033[m")...)
}

// crypto module
type CryptoHashClient struct {
	hash crypto.Hash
}

func (this *CryptoHashClient) Sum(input []byte) []byte {
	h := this.hash.New()
	h.Write(input)
	return h.Sum(nil)
}

type CryptoHmacClient struct {
	hash crypto.Hash
}

func (this *CryptoHmacClient) Sum(input []byte, key []byte) []byte {
	h := hmac.New(this.hash.New, key)
	h.Write(input)
	return h.Sum(nil)
}

type CryptoRsaClient struct{}

func (this *CryptoRsaClient) GenerateKey(length int) (*map[string][]byte, error) {
	if length == 0 {
		length = 2048
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		return nil, err
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey := pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	pubKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	})
	return &map[string][]byte{
		"privateKey": prvkey,
		"publicKey":  pubKey,
	}, nil
}
func (this *CryptoRsaClient) Encrypt(input []byte, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("The public key is invalid.")
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), input)
}
func (this *CryptoRsaClient) Decrypt(input []byte, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("The private key is invalid.")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, input)
}
func (this *CryptoRsaClient) Sign(input []byte, key []byte, algorithm string) ([]byte, error) {
	hash, err := GetHash(algorithm)
	if err != nil {
		return nil, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.SignPKCS1v15(nil, privateKey, hash, digest)
}
func (this *CryptoRsaClient) SignPss(input []byte, key []byte, algorithm string) ([]byte, error) {
	hash, err := GetHash(algorithm)
	if err != nil {
		return nil, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.SignPSS(rand.Reader, privateKey, hash, digest, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
	})
}
func (this *CryptoRsaClient) Verify(input []byte, sign []byte, key []byte, algorithm string) (bool, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return false, errors.New("The public key is invalid.")
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}
	hash, err := GetHash(algorithm)
	if err != nil {
		return false, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	if err = rsa.VerifyPKCS1v15(publicKey.(*rsa.PublicKey), hash, digest[:], sign); err != nil {
		return false, nil
	}
	return true, nil
}
func (this *CryptoRsaClient) VerifyPss(input []byte, sign []byte, key []byte, algorithm string) (bool, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return false, errors.New("The public key is invalid.")
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}
	hash, err := GetHash(algorithm)
	if err != nil {
		return false, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	if err = rsa.VerifyPSS(publicKey.(*rsa.PublicKey), hash, digest[:], sign, nil); err != nil {
		return false, nil
	}
	return true, nil
}

type CryptoClient struct{}

func GetHash(algorithm string) (crypto.Hash, error) {
	switch strings.ToLower(algorithm) {
	case "md5":
		return crypto.MD5, nil
	case "sha1":
		return crypto.SHA1, nil
	case "sha256":
		return crypto.SHA256, nil
	case "sha512":
		return crypto.SHA512, nil
	default:
		return crypto.SHA256, errors.New("Hash algorithm " + algorithm + " is not supported.")
	}
}
func (this *CryptoClient) CreateHash(algorithm string) (*CryptoHashClient, error) {
	if hash, err := GetHash(algorithm); err != nil {
		return nil, err
	} else {
		return &CryptoHashClient{
			hash: hash,
		}, nil
	}
}
func (this *CryptoClient) CreateHmac(algorithm string) (*CryptoHmacClient, error) {
	if hash, err := GetHash(algorithm); err != nil {
		return nil, err
	} else {
		return &CryptoHmacClient{
			hash: hash,
		}, nil
	}
}
func (this *CryptoClient) CreateRsa() *CryptoRsaClient {
	return &CryptoRsaClient{}
}

// db module
type DatabaseTransaction struct {
	Transaction *sql.Tx
}

func ExportDatabaseRows(rows *sql.Rows) ([]interface{}, error) {
	defer rows.Close()

	columns, _ := rows.Columns()
	buf := make([]interface{}, len(columns))
	for index := range columns {
		var a interface{}
		buf[index] = &a
	}

	var records []interface{}

	for rows.Next() {
		_ = rows.Scan(buf...)

		record := make(map[string]interface{})
		for index, data := range buf {
			record[columns[index]] = *data.(*interface{})
		}
		records = append(records, record)
	}

	return records, rows.Err()
}
func (this *DatabaseTransaction) Query(stmt string, params ...interface{}) ([]interface{}, error) {
	rows, err := this.Transaction.Query(stmt, params...)
	if err != nil {
		return nil, err
	}
	return ExportDatabaseRows(rows)
}
func (this *DatabaseTransaction) Exec(stmt string, params ...interface{}) (int64, error) {
	res, err := this.Transaction.Exec(stmt, params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
func (this *DatabaseTransaction) Commit() error {
	return this.Transaction.Commit()
}
func (this *DatabaseTransaction) Rollback() error {
	return this.Transaction.Rollback()
}

type DatabaseClient struct {
	worker *Worker
}

func (this *DatabaseClient) BeginTx() (*DatabaseTransaction, error) {
	if tx, err := Database.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted}); err != nil { // 开启一个新事务，隔离级别为读已提交
		return nil, err
	} else {
		this.worker.AddHandle(tx)
		return &DatabaseTransaction{
			Transaction: tx,
		}, nil
	}
}
func (this *DatabaseClient) Query(stmt string, params ...interface{}) ([]interface{}, error) {
	rows, err := Database.Query(stmt, params...)
	if err != nil {
		return nil, err
	}
	return ExportDatabaseRows(rows)
}
func (this *DatabaseClient) Exec(stmt string, params ...interface{}) (int64, error) {
	res, err := Database.Exec(stmt, params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// email module
type EmailClient struct {
	host     string
	port     int
	username string
	password string
}

func (this *EmailClient) Send(receivers []string, subject string, content string, attachments []struct {
	Name        string
	ContentType string
	Base64      string
}) error {
	address := fmt.Sprintf("%s:%d", this.host, this.port)
	auth := smtp.PlainAuth("", this.username, this.password, this.host)
	msg := []byte(strings.Join([]string{
		"To: " + strings.Join(receivers, ";"),
		"From: " + this.username + "<" + this.username + ">",
		"Subject: " + subject,
		"Content-Type: multipart/mixed; boundary=WebKitBoundary",
		"",
		"--WebKitBoundary",
		"Content-Type: text/plain; charset=utf-8",
		"",
		content,
	}, "\r\n"))
	for _, a := range attachments {
		msg = append(
			msg,
			[]byte(strings.Join([]string{
				"",
				"--WebKitBoundary",
				"Content-Transfer-Encoding: base64",
				"Content-Disposition: attachment",
				"Content-Type: " + a.ContentType + "; name=" + a.Name,
				"",
				a.Base64,
			}, "\r\n"))...,
		)
	}

	if this.port == 25 { // 25 端口直接发送
		return smtp.SendMail(address, auth, this.username, receivers, msg)
	}

	config := &tls.Config{ // 其他端口如 465 需要 TLS 加密
		InsecureSkipVerify: true, // 不校验服务端证书
		ServerName:         this.host,
	}
	conn, err := tls.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, this.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("AUTH"); ok {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(this.username); err != nil {
		return err
	}

	for _, receiver := range receivers {
		if err = client.Rcpt(receiver); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err == nil {
		client.Quit()
	}
	return nil
}

// file module
type FileClient struct{}

func (this *FileClient) Read(name string) ([]byte, error) {
	fp := path.Clean("files/" + name)
	if !strings.HasPrefix(fp, "files/") {
		return nil, errors.New("Permission denial.")
	}
	return ioutil.ReadFile(fp)
}
func (this *FileClient) Write(name string, bytes []byte) error {
	fp := path.Clean("files/" + name)
	if !strings.HasPrefix(fp, "files/") {
		return errors.New("Permission denial.")
	}
	paths, _ := filepath.Split(fp)
	os.MkdirAll(paths, os.ModePerm)
	return ioutil.WriteFile(fp, bytes, 0664)
}

// http module
type HttpClient struct {
	client *http.Client
}

func (this *HttpClient) Request(method string, url string, header map[string]string, body string) (response interface{}, err error) {
	req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(body))
	if err != nil {
		return
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}

	resp, err := this.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	response = map[string]interface{}{
		"status": resp.StatusCode,
		"header": resp.Header,
		"data":   &DataBuffer{data: data},
	}
	return
}

type DataBuffer struct {
	data []byte
}

func (this *DataBuffer) ToBytes() []byte {
	return this.data
}
func (this *DataBuffer) ToString() string {
	return string(this.data)
}
func (this *DataBuffer) ToJson() (obj interface{}, err error) {
	err = json.Unmarshal(this.data, &obj)
	return
}

// image module
type ImageClient struct{}

func (this *ImageClient) New(width int, height int) *ImageBuffer {
	return &ImageBuffer{
		image:   image.NewRGBA(image.Rect(0, 0, width, height)),
		Width:   width,
		offsetX: 0,
		Height:  height,
		offsetY: 0,
	}
}
func (this *ImageClient) Parse(input []byte) (imgBuf *ImageBuffer, err error) {
	m, _, err := image.Decode(bytes.NewBuffer(input)) // 图片文件解码
	if err != nil {
		return
	}

	bounds := m.Bounds()
	imgBuf = &ImageBuffer{
		image:   m,
		Width:   bounds.Max.X - bounds.Min.X,
		offsetX: bounds.Min.X,
		Height:  bounds.Max.Y - bounds.Min.Y,
		offsetY: bounds.Min.Y,
	}
	return
}
func (this *ImageClient) ToBytes(b ImageBuffer) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, b.image, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type ImageBuffer struct {
	image   image.Image
	Width   int
	offsetX int
	Height  int
	offsetY int
}

func (this *ImageBuffer) Get(x int, y int) uint32 {
	r, g, b, a := this.image.At(x+this.offsetX, y+this.offsetY).RGBA()
	return r << 24 & g << 16 & b << 8 & a
}
func (this *ImageBuffer) Set(x int, y int, p uint32) {
	this.image.(*image.RGBA).Set(x+this.offsetX, y+this.offsetY, color.RGBA{uint8(p >> 24), uint8(p >> 16), uint8(p >> 8), uint8(p)})
}

// lock module
var LockCache struct {
	sync.Mutex
	clients map[string]*LockClient
}

type LockClient struct {
	name   *string
	mutex  *sync.Mutex
	locked *bool
}

func (this *LockClient) lock() bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if *this.locked == true {
		return false
	}
	*this.locked = true
	return true
}
func (this *LockClient) Lock(timeout int) error {
	for i := 0; i < timeout; i++ {
		if this.lock() {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	this.Unlock()
	return errors.New("Acquire lock " + *this.name + " timeout.")
}
func (this *LockClient) Unlock() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	*this.locked = false
}

// pipe module
var PipeCache map[string]*BlockingQueueClient

// socket module
type Socket struct {
	worker *Worker
}
type SocketListener struct {
	listener *net.Listener
}

func (this *Socket) Listen(protocol string, port int) (*SocketListener, error) {
	listener, err := net.Listen(protocol, fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	this.worker.AddHandle(&listener)
	return &SocketListener{
		listener: &listener,
	}, err
}
func (this *Socket) Dial(protocol string, host string, port int) (*SocketConn, error) {
	conn, err := net.Dial(protocol, fmt.Sprintf("%s:%d", host, port))
	return &SocketConn{
		conn:   &conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

type SocketConn struct {
	conn   *net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func (this *SocketListener) Accept() (*SocketConn, error) {
	conn, err := (*this.listener).Accept()
	return &SocketConn{
		conn:   &conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}
func (this *SocketConn) ReadLine() ([]byte, error) {
	line, err := this.reader.ReadBytes('\n')
	if err == io.EOF {
		return nil, nil
	}
	return line, err
}
func (this *SocketConn) Write(data []byte) (int, error) {
	count, err := this.writer.Write(data)
	this.writer.Flush()
	return count, err
}
func (this *SocketConn) Close() {
	(*this.conn).Close()
}

// ulid module
var ULIDCache struct {
	sync.Mutex
	timestamp  *int64
	randomness *[16]byte
	num        *uint64
}

func CreateULID() string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond) // 时间戳，精确到毫秒

	var randomness [16]byte
	var num uint64

	ULIDCache.Lock()
	defer ULIDCache.Unlock()

	if ULIDCache.timestamp != nil && *ULIDCache.timestamp == timestamp {
		randomness = *ULIDCache.randomness
		num = *ULIDCache.num + 1
		ULIDCache.num = &num
	} else {
		rand.Read(randomness[:])
		for i := 8; i < 16; i++ { // 后 8 个字节转数字
			num |= uint64(randomness[i]) << (56 - (i-8)*8)
		}
		ULIDCache.timestamp = &timestamp
		ULIDCache.randomness = &randomness
		ULIDCache.num = &num
	}

	var buf [26]byte

	alphabet := "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for i := 0; i < 10; i++ { // 前 10 个字符为时间戳
		buf[i] = alphabet[timestamp>>(45-i*5)&0b11111]
	}
	for i := 10; i < 18; i++ { // 中 8 个字符为随机数
		buf[i] = alphabet[randomness[i-10]&0b11111]
	}
	for i := 18; i < 26; i++ { // 后 8 个字符为递增随机数
		buf[i] = alphabet[num>>(56-(i-18)*8)&0b11111]
	}

	return string(buf[:])
}

//#endregion
