package main

import (
    "bytes"
    "crypto"
    "crypto/hmac"
    "crypto/md5"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
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
    "html/template"
    "image"
    "image/color"
    _ "image/gif"
    "image/jpeg" // 需要导入 "image/jpeg"、"image/gif"、"image/png" 去解码 jpg、gif、png 图片，否则当使用 image.Decode 处理图片文件时，会报 image: unknown format 错误
    _ "image/png"
    "io/ioutil"
    "log"
    "net/http"
    "net/smtp"
    "os"
    "regexp"
    "strings"
    "sync"
    "time"
    "golang.org/x/net/http2"
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
    "github.com/shirou/gopsutil/process"
    _ "github.com/mattn/go-sqlite3"
    "github.com/gorilla/websocket"
)

type Source struct {
    Name string `json:"name"`
    Type string `json:"type"` // ts, tpl, vue
    Content string `json:"content"`
    Compiled string `json:"compiled"`
}
type Worker struct {
    Runtime *goja.Runtime
    Function goja.Callable
}

//go:embed index.html
var FileList embed.FS

var Database *sql.DB

var WorkerPool struct {
    Channels chan *Worker
    Workers []*Worker
}

func init() {
    var err error
    Database, err = sql.Open("sqlite3", "./my.db")
    if err != nil {
        panic(err)
    }
    Database.Exec(`
        create table if not exists source (
            name varchar(64) not null,
            type varchar(8) not null,
            content text not null,
            compiled text not null,
            primary key(name, type)
        );
    `)
}

func main() {
    // 获取启动参数
    arguments := ParseStartupArguments()

    // 初始化日志文件
    logFile, err := os.OpenFile("cube.log", os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0644)
    if err != nil {
        panic(err)
        return
    }
    log.SetOutput(logFile)

    // 创建虚拟机池
    CreateWorkerPool(arguments.Count)

    http.HandleFunc("/source", func(w http.ResponseWriter, r *http.Request) {
        var (
            data interface{}
            err error
        )
        switch r.Method {
            case "GET":
                data, err = HandleSourceGet(w, r)
            case "POST":
                err = HandleSourcePost(w, r)
            case "DELETE":
                err = HandleSourceDelete(w, r)
            default:
                err = errors.New("Unsupported method " + r.Method)
        }
        if err != nil {
            Error(w, err)
            return
        }
        Success(w, data)
    })
    http.HandleFunc("/service/", func(w http.ResponseWriter, r *http.Request) {
        name := r.URL.Path[9:]
        if name == "" {
            name = "index"
        }
        worker := <-WorkerPool.Channels
        defer func() {
            WorkerPool.Channels <- worker
        }()

        timer := time.AfterFunc(60000*time.Millisecond, func() { // 允许脚本最大执行的时间为 60 秒
            worker.Runtime.Interrupt("The service executed timeout.")
        })
        defer timer.Stop()

        request := ServiceRequest{
            request: r,
            responseWriter: w,
            timer: timer,
        }
        value, err := worker.Function(
            nil,
            worker.Runtime.ToValue(name),
            worker.Runtime.ToValue(&request),
        )
        if request.isWebSocket == true { // 如果是 WebSocket，不需要封装响应
            return
        }

        if err != nil {
            Error(w, err)
            return
        }
        Success(w, ExportGojaValue(value))
    })
    http.HandleFunc("/resource/", func(w http.ResponseWriter, r *http.Request) {
        name := r.URL.Path[10:]
        if ok, _ := regexp.MatchString("^\\w{2,32}\\.vue$", name); !ok {
            Error(w, errors.New("Invalid argument name, not a vue."))
            return
        }
        names := strings.Split(name, ".")

        rows, err := Database.Query("select content from source where name = ? and type = ?", names[0], names[1])
        if err != nil {
            Error(w, err)
        }
        defer rows.Close()
        if rows.Next() == false {
            Error(w, errors.New("The resource is not found: " + name))
            return
        }
        source := Source{}
        err = rows.Scan(&source.Content)
        Success(w, source.Content)
    })
    http.Handle("/", http.FileServer(http.FS(FileList)))

    // 监控当前进程的内存和 cpu 使用率
    go func () {
        pid := os.Getppid()
        p, _ := process.NewProcess(int32(pid))
        ticker := time.NewTicker(time.Millisecond * 1000)
        for _ = range ticker.C {
            c, _ := p.CPUPercent()
            m, _ := p.MemoryInfo()
            fmt.Printf("\rcpu: %.6f%%, memory: %.2fmb, vm: %d/%d" + " ", // 结尾预留一个空格防止刷新过程中因字符串变短导致上一次打印的文本在结尾出溢出
                c,
                float32(m.RSS) / 1024 / 1024,
                len(WorkerPool.Workers) - len(WorkerPool.Channels), len(WorkerPool.Workers),
            )
        }
    }()

    // 启动服务
    if !arguments.Secure {
        fmt.Println("server has started on http://127.0.0.1:" + arguments.Port + " 🚀")
        http.ListenAndServe(":" + arguments.Port, nil)
    } else {
        fmt.Println("server has started on https://127.0.0.1:" + arguments.Port + " 🚀")
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
            Addr: ":" + arguments.Port,
            TLSConfig: config,
        }
        server.ListenAndServeTLS(arguments.ServerCert, arguments.ServerKey)
    }
}

func ParseStartupArguments() (a struct { Count int; Port string; Secure bool; ServerKey string; ServerCert string; ClientCertVerify bool; }) {
    flag.IntVar(&a.Count, "n", 1, "The total count of virtual machines.") // 定义命令行参数 c，表示虚拟机的总个数，返回 Int 类型指针，默认值为 1，其值在 Parse 后会被修改为命令参数指定的值
    flag.StringVar(&a.Port, "p", "8090", "Port to use.")
    flag.BoolVar(&a.Secure, "s", false, "Enable https.")
    flag.StringVar(&a.ServerKey, "k", "server.key", "SSL key file.")
    flag.StringVar(&a.ServerCert, "c", "server.crt", "SSL cert file.")
    flag.BoolVar(&a.ClientCertVerify, "v", false, "Enable client cert verify.")
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
                panic(errors.New("Unsupported type " + t + "."))
        }
        if !success {
            panic(errors.New("The " + name + " is not a " + t + "."))
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
            w.Write(v.data) // Write 必须在 WriteHeader 之后调用，否则将会抛出异常 http: superfluous response.WriteHeader call from ...
        default: // map[string]interface[]
            w.Header().Set("Content-Type", "application/json")
            enc := json.NewEncoder(w)
            enc.SetEscapeHTML(false) // 见 https://pkg.go.dev/encoding/json#Marshal，字符串值编码为强制为有效 UTF-8 的 JSON 字符串，用 Unicode 替换符文替换无效字节。为了使 JSON 能够安全地嵌入 HTML 标记中，字符串使用 HTMLEscape 编码，它将替换 `<`、`>`、`&`、`U+2028` 和 `U+2029`，并转义到 `\u003c`、`\u003e`、`\u0026`、`\u2028` 和 `\u2029`。在使用编码器时，可以通过调用 SetEscapeHTML(false) 禁用此替换。
            enc.Encode(map[string]interface{}{
                "code": "0",
                "message": "success",
                "data": v, // 注：这里的 data 如果为 []byte 类型或包含 []byte 类型的属性，在通过 json 序列化后将会被自动转码为 base64 字符串
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
        "code": code,
        "message": message,
    })
}

//#region Source 接口请求

func HandleSourceGet(w http.ResponseWriter, r *http.Request) (data struct { Sources []Source `json:"sources"`; Total int `json:"total"`; }, err error) {
    r.ParseForm()
    name := r.Form.Get("name")
    stype := r.Form.Get("type")
    if stype == "" {
        stype = "ts"
    }
    from := r.Form.Get("from")
    if from == "" {
        from = "0"
    }
    size := r.Form.Get("size")
    if size == "" {
        size = "10"
    }

    err = Database.QueryRow("select count(1) from source where name like ? and type = ?", "%"+name+"%", stype).Scan(&data.Total)
    if err != nil {
        return
    }

    rows, err := Database.Query("select name, type, content, compiled from source where name like ? and type = ? limit ?, ?", "%"+name+"%", stype, from, size)
    if err != nil {
        return
    }
    defer rows.Close()
    for rows.Next() {
        source := Source{}
        err := rows.Scan(&source.Name, &source.Type, &source.Content, &source.Compiled)
        if err != nil {
            panic(err)
        }
        data.Sources = append(data.Sources, source)
    }
    err = rows.Err()
    if err != nil {
        return
    }

    return
}

func HandleSourcePost(w http.ResponseWriter, r *http.Request) error {
    // 读取请求消息体
    body, err := ioutil.ReadAll(r.Body)
    defer r.Body.Close()
    if err != nil {
        return err
    }

    var source Source
    err = json.Unmarshal(body, &source)
    if err != nil {
        return err
    }

    // 校验脚本名称
    if source.Type == "ts" {
        if ok, _ := regexp.MatchString("^(node_modules/)?\\w{2,32}$", source.Name); !ok {
            return errors.New("The name is required. It must be a letter, number, or underscore with a length of 2 to 32.")
        }
    } else if source.Type == "tpl" || source.Type == "vue" {
        if ok, _ := regexp.MatchString("^\\w{2,32}$", source.Name); !ok {
            return errors.New("The name is required. It must be a letter, number, or underscore with a length of 2 to 32.")
        }
    } else {
        return errors.New("The type is required. It must be ts, tpl or vue.")
    }

    // 新增或修改脚本
    stmt, err := Database.Prepare("insert or replace into source (name, type, content, compiled) values(?, ?, ?, ?)")
    if err != nil {
        return err
    }
    defer stmt.Close()
    _, err = stmt.Exec(source.Name, source.Type, source.Content, source.Compiled)
    if err != nil {
        return err
    }

    // 重新加载 require loader
    RegisterModuleLoader()

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
        return errors.New("The parameter name is required.")
    }

    res, err := Database.Exec("delete from source where name = ? and type = ?", name, stype)
    if err != nil {
        return err
    }
    if res == nil {
        return errors.New("The source is not found.")
    }

    return nil
}

//#endregion

//#region Goja 运行时

func CreateJsRuntime() *goja.Runtime {
    runtime := goja.New()

    runtime.Set("exports", runtime.NewObject())

    runtime.Set("ServiceResponse", func (call goja.ConstructorCall) *goja.Object {
        a0, ok := call.Argument(0).Export().(int64)
        if !ok {
            panic(runtime.NewTypeError("Invalid argument status, not a int."))
        }
        a1, ok := call.Argument(1).Export().(map[string]interface{})
        if !ok {
            panic(runtime.NewTypeError("Invalid argument header, not a map."))
        }
        header := make(map[string]string, len(a1))
        for k, v := range a1 {
            s, ok := v.(string)
            if !ok {
                panic(runtime.NewTypeError("Invalid argument " + k + ", not a string."))
            }
            header[k] = s
        }
        a2 := ExportGojaValue(call.Argument(2))
        data := []byte(nil)
        if s, ok := a2.(string); !ok {
            if data, ok = a2.([]byte); !ok {
                panic(runtime.NewTypeError("The data should be a string or a byte array."))
            }
        } else {
            data = []byte(s)
        }
        i := &ServiceResponse{
            status: int(a0),
            header: header,
            data: data,
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
                module = &Base64Struct{}
            case "bqueue":
                module = func(size int) *BlockingQueueClient {
                    return &BlockingQueueClient{
                        queue: make(chan interface{}, size),
                    }
                }
            case "crypto":
                module = &CryptoClient{}
            case "db":
                module = &DatabaseClient{}
            case "email":
                module = func(host string, port int, username string, password string) *EmailClient {
                    return &EmailClient{
                        host: host,
                        port: port,
                        username: username,
                        password: password,
                    }
                }
            case "http":
                module = func(options map[string]interface{}) *HttpClient {
                    client := &http.Client{}
                    if options != nil {
                        config := &tls.Config{}
                        if caCert, ok := ExportMapValue(options, "caCert", "string"); ok { // 配置 ca 证书
                            config.RootCAs = x509.NewCertPool()
                            config.RootCAs.AppendCertsFromPEM([]byte(caCert.(string)))
                        }
                        if cert, ok := ExportMapValue(options, "cert", "string"); ok {
                            var c tls.Certificate // 参考实现 https://github.com/sideshow/apns2/blob/HEAD/certificate/certificate.go
                            b1, _ := pem.Decode([]byte(cert.(string))) // 读取公钥
                            if b1 == nil {
                                panic(errors.New("No public key found."))
                            }
                            c.Certificate = append(c.Certificate, b1.Bytes) // tls.Certificate 存储了一个证书链（类型为 [][]byte），包含一个或多个 x509.Certificate（类型为 []byte）
                            if key, ok := ExportMapValue(options, "key", "string"); ok {
                                b2, _ := pem.Decode([]byte(key.(string))) // 读取私钥
                                if b2 == nil {
                                    panic(errors.New("No private key found."))
                                }
                                c.PrivateKey, err = x509.ParsePKCS1PrivateKey(b2.Bytes) // 使用 PKCS#1 格式
                                if err != nil {
                                    c.PrivateKey, err = x509.ParsePKCS8PrivateKey(b2.Bytes) // 使用 PKCS#8 格式
                                    if err != nil {
                                        panic(errors.New("Failed to parse private key."))
                                    }
                                }
                            }
                            if len(c.Certificate) == 0 || c.PrivateKey == nil {
                                panic(errors.New("No private key or public key found."))
                            }
                            if a, err := x509.ParseCertificate(c.Certificate[0]); err == nil {
                                c.Leaf = a
                            }
                            config.Certificates = []tls.Certificate{c} // 配置客户端证书
                        }
                        if insecureSkipVerify, ok := ExportMapValue(options, "insecureSkipVerify", "bool"); ok { // 忽略服务端证书校验
                            config.InsecureSkipVerify = insecureSkipVerify.(bool)
                        }
                        httpVersion, ok := ExportMapValue(options, "version", "int")
                        if !ok {
                            httpVersion = 1
                        }
                        switch httpVersion {
                            case 1:
                                client.Transport = &http.Transport{
                                    TLSClientConfig: config,
                                }
                            case 2:
                                client.Transport = &http2.Transport{ // 配置使用 http 2 协议
                                    TLSClientConfig: config,
                                }
                            default:
                                panic(errors.New("Invali http version, it must be 1 or 2."))
                        }
                    }
                    return &HttpClient{
                        client: client,
                    }
                }
            case "image":
                module = &ImageClient{}
            case "pipe":
                module = func(name string) *BlockingQueueClient {
                    if PipePool == nil {
                        PipePool = make(map[string]*BlockingQueueClient, 99)
                    }
                    if PipePool[name] == nil {
                        PipePool[name] = &BlockingQueueClient{
                            queue: make(chan interface{}, 99),
                        }
                    }
                    return PipePool[name]
                }
            case "template":
                module = func(name string, input map[string]interface{}) (output string, err error) {
                    rows, err := Database.Query("select content from source where name = ? and type = ?", name, "tpl")
                    if err != nil {
                        panic(err.Error())
                    }
                    defer rows.Close()
                    if rows.Next() == false {
                        err = errors.New("The template is not found: " + name)
                        return
                    }
                    source := Source{}
                    err = rows.Scan(&source.Content)
                    if err != nil {
                        return
                    }

                    t, err := template.New(name).Parse(source.Content)
                    if err != nil {
                        return
                    }
                    buf := new(bytes.Buffer)
                    t.Execute(buf, input)
                    output = buf.String()
                    return
                }
            default:
                err = errors.New("The module is not found.")
        }
        return
    })

    runtime.SetMaxCallStackSize(2048)

    return runtime
}

func CreateWorkerPool(count int) {
    WorkerPool.Workers = make([]*Worker, count) // 创建 goja 实例池
    WorkerPool.Channels = make(chan *Worker, len(WorkerPool.Workers))
    program, _ := goja.Compile("index", `(function (name, req) { return require("./" + name).default(req); })`, false) // 编译源码为 Program，strict 为 false
    for i := 0; i < len(WorkerPool.Workers); i++ {
        runtime := CreateJsRuntime() // 创建 goja 运行时
        entry, err := runtime.RunProgram(program) // 这里使用 RunProgram，可复用已编译的代码，相比直接调用 RunString 更显著提升性能
        if err != nil {
            panic(err)
        }
        function, ok := goja.AssertFunction(entry)
        if !ok {
            panic(errors.New("The entry is not a function."))
        }
        worker := Worker{Runtime: runtime, Function: function}
        WorkerPool.Workers[i] = &worker
        WorkerPool.Channels <- &worker
    }
    RegisterModuleLoader() // 注册 Module 加载器
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

func RegisterModuleLoader() {
    registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) { // 创建自定义 require loader（脚本每次修改后，registry 需要重新生成，防止 module 被缓存，从而导致 module 修改后不生效）
        // 从数据库中查找模块
        rows, err := Database.Query("select compiled from source where name = ? and type = ?", path, "ts")
        if err != nil {
            panic(err.Error())
        }
        defer rows.Close()
        if rows.Next() == false {
            return nil, errors.New("The module is not found: " + path)
        }
        source := Source{}
        err = rows.Scan(&source.Compiled)
        return []byte(source.Compiled), nil
    })

    for _, runtime := range WorkerPool.Workers {
        _ = registry.Enable(runtime.Runtime) // 启用自定义 require loader
    }
}

//#endregion

//#region Service 请求、响应

// service http request
type ServiceRequest struct {
    request *http.Request
    responseWriter http.ResponseWriter
    timer *time.Timer
    isWebSocket bool
    body interface{} // 用于缓存请求消息体，防止重复读取和关闭 body 流
}
func (s *ServiceRequest) GetHeader() http.Header {
    return s.request.Header
}
func (s *ServiceRequest) GetURL() interface{} {
    u := s.request.URL
    return map[string]interface{}{
        "path": u.Path,
        "params": u.Query(),
    }
}
func (s *ServiceRequest) GetBody() (body interface{}, err error) {
    if s.body != nil {
        body = s.body
        return
    }
    b, err := ioutil.ReadAll(s.request.Body)
    defer s.request.Body.Close()
    if err != nil {
        return
    }
    err = json.Unmarshal(b, &body)
    s.body = body
    return
}
func (s *ServiceRequest) GetMethod() string {
    return s.request.Method
}
func (s *ServiceRequest) GetForm() interface{} {
    s.request.ParseForm() // 需要转换后才能获取表单
    return s.request.Form
}
func (s *ServiceRequest) GetCerts() interface{} { // 获取客户端证书
    return s.request.TLS.PeerCertificates
}
func (s *ServiceRequest) UpgradeToWebSocket() (ws *ServiceWebSocket, err error) {
    s.isWebSocket = true // upgrader.Upgrade 内部已经调用过 WriteHeader 方法了，后续不应再次调用，否则将会出现 http: superfluous response.WriteHeader call from ... 的异常
    s.timer.Stop() // 关闭定时器，WebSocket 不需要设置超时时间
    upgrader := websocket.Upgrader{}
    conn, err := upgrader.Upgrade(s.responseWriter, s.request, nil)
    ws = &ServiceWebSocket{
        connection: conn,
    }
    return
}

// service http response
type ServiceResponse struct {
    status int
    header map[string]string
    data []byte
}
func (s *ServiceResponse) SetStatus(status int) { // 设置响应状态码
    s.status = status
}
func (s *ServiceResponse) SetHeader(header map[string]string) { // 设置响应消息头
    s.header = header
}
func (s *ServiceResponse) SetData(data []byte) { // 设置响应消息体
    s.data = data
}

// service websocket
type ServiceWebSocket struct {
    connection *websocket.Conn
}
func (s *ServiceWebSocket) Read() (output interface{}, err error) {
    messageType, data, err := s.connection.ReadMessage()
    if err != nil {
        panic(err)
        return
    }
    output = map[string]interface{}{
        "messageType": messageType,
        "data": data,
    }
    return
}
func (s *ServiceWebSocket) Send(data []byte) (err error) {
    err = s.connection.WriteMessage(1, data) // message type：0 表示消息是文本格式，1 表示消息是二进制格式。这里 data 是 []byte，因此固定使用二进制格式类型
    if err != nil {
        panic(err)
    }
    return
}
func (s *ServiceWebSocket) Close() {
    s.connection.Close()
}

//#endregion

//#region 内置模块

// base64 module
type Base64Struct struct{}
func (b *Base64Struct) Encode(input []byte) string {
    return base64.StdEncoding.EncodeToString(input)
}
func (b *Base64Struct) Decode(input string) ([]byte, error) { // 返回类型 []byte 将隐式地转换为 js/ts 中的 number[]
    return base64.StdEncoding.DecodeString(input)
}

// blocking queue module
type BlockingQueueClient struct {
    queue chan interface{}
    mutex sync.Mutex
}
func (b *BlockingQueueClient) Put(input interface{}, timeout int) error {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    select {
        case b.queue <- input:
            return nil
        case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列入列最大超时时间为 timeout 毫秒
            return errors.New("The blocking queue is full, waiting for put timeout.")
    }
}
func (b *BlockingQueueClient) Poll(timeout int) (interface{}, error) {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    select {
        case output := <-b.queue:
            return output, nil
        case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列出列最大超时时间为 timeout 毫秒
            return nil, errors.New("The blocking queue is empty, waiting for poll timeout.")
    }
}
func (b *BlockingQueueClient) Drain(size int, timeout int) (output []interface{}) {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    output = make([]interface{}, 0, size) // 创建切片，初始大小为 0，最大为 size
    c := make(chan int, 1)
    go func(ch chan int) {
        for i := 0; i < size; i++ {
            output = append(output, <-b.queue)
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

// console module
type ConsoleClient struct {
    runtime *goja.Runtime
}
func (c *ConsoleClient) Log(a ...interface{}) {
    log.Println(append([]interface{}{"\r"+time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Log"}, a...)...)
}
func (c *ConsoleClient) Debug(a ...interface{}) {
    log.Println(append(append([]interface{}{"\r"+"\033[1;30m"+time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Debug"}, a...), "\033[m")...)
}
func (c *ConsoleClient) Info(a ...interface{}) {
    log.Println(append(append([]interface{}{"\r"+"\033[0;34m"+time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Info"}, a...), "\033[m")...)
}
func (c *ConsoleClient) Warn(a ...interface{}) {
    log.Println(append(append([]interface{}{"\r"+"\033[0;33m"+time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Warn"}, a...), "\033[m")...)
}
func (c *ConsoleClient) Error(a ...interface{}) {
    log.Println(append(append([]interface{}{"\r"+"\033[0;31m"+time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Error"}, a...), "\033[m")...)
}

// crypto module
type CryptoClient struct{}
func (d *CryptoClient) Md5(input []byte) [16]byte {
    return md5.Sum(input)
}
func (d *CryptoClient) Sha256(input []byte) []byte {
    h := sha256.New()
    h.Write(input)
    return h.Sum(nil)
}
func (d *CryptoClient) HmacWithSha256(input []byte, key []byte) []byte {
    h := hmac.New(sha256.New, key)
    h.Write(input)
    return h.Sum(nil)
}
func (d *CryptoClient) RsaWithSha256(input []byte, key []byte) (sign []byte, err error) {
    block, _  := pem.Decode(key)
    privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
    h := sha256.New()
    h.Write(input)
    digest := h.Sum(nil)
    sign, _ = rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, digest)
    return
}
func (d *CryptoClient) RsaWithSha256Verify(input []byte, sign []byte, key []byte) bool {
    block, _ := pem.Decode(key)
    if block == nil {
        panic(errors.New("The public key is invalid."))
    }
    publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        panic(err)
    }
    digest := sha256.Sum256(input)
    err = rsa.VerifyPKCS1v15(publicKey.(*rsa.PublicKey), crypto.SHA256, digest[:], sign)
    if err != nil {
        return false
    }
    return true
}
func (d *CryptoClient) RsaEncrypt(input []byte, key []byte) []byte {
    block, _ := pem.Decode(key)
    if block == nil {
        panic(errors.New("The public key is invalid."))
    }
    publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        panic(err)
    }
    cipher, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), input)
    if err != nil {
        panic(err)
    }
    return cipher
}
func (d *CryptoClient) RsaDecrypt(input []byte, key []byte) []byte {
    block, _ := pem.Decode(key)
    if block == nil {
        panic(errors.New("The private key is invalid."))
    }
    privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil {
        panic(err)
    }
    output, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, input)
    if err != nil {
        panic(err)
    }
    return output
}
func (d *CryptoClient) GenerateRsaKey() (prvkey []byte, pubKey []byte) {
    privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        panic(err)
    }
    derStream := x509.MarshalPKCS1PrivateKey(privateKey)
    block := &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: derStream,
    }
    prvkey = pem.EncodeToMemory(block)
    publicKey := &privateKey.PublicKey
    derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
    if err != nil {
        panic(err)
    }
    block = &pem.Block{
        Type:  "PUBLIC KEY",
        Bytes: derPkix,
    }
    pubKey = pem.EncodeToMemory(block)
    return
}

// db module
type DatabaseClient struct{}
func (d *DatabaseClient) Query(stmt string, param ...interface{}) (records []interface{}, err error) {
    rows, err := Database.Query(stmt, param...)
    if err != nil {
        return
    }
    defer rows.Close()

    fields, _ := rows.Columns()

    for rows.Next() {
        dataset := make([]interface{}, len(fields))
        for i := range dataset {
            dataset[i] = &dataset[i]
        }
        rows.Scan(dataset...)
        record := make(map[string]interface{})
        for i, v := range dataset {
            record[fields[i]] = v
        }
        records = append(records, record)
    }

    return
}
func (d *DatabaseClient) Exec(stmt string, param ...interface{}) (res interface{}, err error) {
    s, err := Database.Prepare(stmt)
    if err != nil {
        return
    }
    defer s.Close()

    res, err = s.Exec(param...)

    return
}

// email module
type EmailClient struct {
    host string
    port int
    username string
    password string
}
func (e *EmailClient) Send(receivers []string, subject string, content string) (err error) {
    address := fmt.Sprintf("%s:%d", e.host, e.port)
    auth := smtp.PlainAuth("", e.username, e.password, e.host)
    msg := append(
        []byte(fmt.Sprintf("To: %s\r\nFrom: %s<%s>\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n", strings.Join(receivers, ","), e.username, e.username, subject)),
        content...,
    )

    if e.port == 25 { // 25 端口直接发送
        err = smtp.SendMail(address, auth, e.username, receivers, msg)
        return
    }

    config := &tls.Config{ // 其他端口如 465 需要 TLS 加密
        InsecureSkipVerify: true, // 不校验服务端证书
        ServerName: e.host,
    }
    conn, err := tls.Dial("tcp", address, config)
    if err != nil {
        return
    }
    client, err := smtp.NewClient(conn, e.host)
    if err != nil {
        return
    }
    defer client.Close()
    if ok, _ := client.Extension("AUTH"); ok {
        if err = client.Auth(auth); err != nil {
            return
        }
    }
    if err = client.Mail(e.username); err != nil {
        return
    }

    for _, receiver := range receivers {
        if err = client.Rcpt(receiver); err != nil {
            return
        }
    }
    w, err := client.Data()
    if err != nil {
        return
    }
    _, err = w.Write(msg)
    if err != nil {
        return
    }
    err = w.Close()
    if err == nil {
        client.Quit()
    }
    return
}

// http module
type HttpClient struct{
    client *http.Client
}
func (h *HttpClient) Request(method string, url string, header map[string]string, body string) (response interface{}, err error) {
    req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(body))
    if err != nil {
        return
    }
    for key, value := range header {
        req.Header.Set(key, value)
    }

    resp, err := h.client.Do(req)
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
        "data": &DataBuffer{data: data},
    }
    return
}

type DataBuffer struct {
    data []byte
}
func (b *DataBuffer) ToBytes() []byte {
    return b.data
}
func (b *DataBuffer) ToString() string {
    return string(b.data)
}
func (b *DataBuffer) ToJson() (obj interface{}, err error) {
    err = json.Unmarshal(b.data, &obj)
    return
}

// image module
type ImageClient struct{}
func (e *ImageClient) New(width int, height int) *ImageBuffer {
    return &ImageBuffer{
        image: image.NewRGBA(image.Rect(0, 0, width, height)),
        Width: width,
        offsetX: 0,
        Height: height,
        offsetY: 0,
    }
}
func (e *ImageClient) Parse(input []byte) (imgBuf *ImageBuffer, err error) {
    m, _, err := image.Decode(bytes.NewBuffer(input)) // 图片文件解码
    if err != nil {
        return
    }

    bounds := m.Bounds()
    imgBuf = &ImageBuffer{
        image: m,
        Width: bounds.Max.X - bounds.Min.X,
        offsetX: bounds.Min.X,
        Height: bounds.Max.Y - bounds.Min.Y,
        offsetY: bounds.Min.Y,
    }
    return
}
func (e *ImageClient) ToBytes(b ImageBuffer) (output []byte, err error) {
    buf := new(bytes.Buffer)
    err = jpeg.Encode(buf, b.image, nil)
    if err != nil {
        return
    }
    output = buf.Bytes()
    return
}

type ImageBuffer struct {
    image image.Image
    Width int
    offsetX int
    Height int
    offsetY int
}
func (e *ImageBuffer) Get(x int, y int) uint32 {
    r, g, b, a := e.image.At(x+e.offsetX, y+e.offsetY).RGBA()
    return r << 24 & g << 16 & b << 8 & a
}
func (e *ImageBuffer) Set(x int, y int, p uint32) {
    e.image.(*image.RGBA).Set(x+e.offsetX, y+e.offsetY, color.RGBA{uint8(p >> 24), uint8(p >> 16), uint8(p >> 8), uint8(p)})
}

// pipe module
var PipePool map[string]*BlockingQueueClient

//#endregion
