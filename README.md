# Cube

A simple web server that can be developed online using typescript/javascript.

## Getting started

1. Clone the git repo.

2. Make sure all dependencies are installed:
    ```bash
    make tidy
    ```

3. Start the server:
    ```bash
    make build && ./cube
    ```
    Or start from source code:
    ```bash
    make run
    ```

4. For more startup parameters, please refer to:
    ```bash
    ./cube --help
    ``` 

4. Open `http://127.0.0.1:8090/` in browser.

## Run with SSL/TLS

1. Ensure that `ca.key`, `ca.crt`, `server.key` and `server.crt` have been created:
    ```bash
    make crt
    ```

2. Start the server:
    ```bash
    ./cube \
        -n 8 \ # using 8 virtual machines
        -p 8443 \ # server with port 8443
        -s \ # enable SSL/TLS
        -v # enable client cert verification
    ```

3. If you are using a self-signed certificate, you can install the `ca.crt` to the local root certificate library.

4. Open `https://127.0.0.1:8443/` in browser.

5. You can run your service with client certificate using curl:
    ```bash
    # Create client.key and client.crt
    make ccrt

    # Run the service with client.crt and ca.crt
    curl --cacert ./ca.crt --cert ./client.crt --key ./client.key https://127.0.0.1:8443/service/foo
    ```
    Or you can access it in chrome:
    ```cmd
    rem Parse client.crt and client.key into client.p12
    openssl pkcs12 -export -in client.crt -inkey client.key -out client.p12 -passout pass:123456

    rem Install client.p12 into My certificate store
    certutil -importPFX -f -p 123456 My client.p12

    rem Open https://127.0.0.1:8443/ and select your client certificate
    chrome https://127.0.0.1:8443/
    ```

## Run with HTTP/3

1. Ensure that `ca.key`, `ca.crt`, `server.key` and `server.crt` have been created:
    ```bash
    make crt
    ```

2. Start the server:
    ```bash
    ./cube \
        -n 8 \ # using 8 virtual machines
        -p 8443 \ # server with port 8443
        -s \ # enable SSL/TLS
        -3 # enable HTTP/3
    ```

3. You can test your service using curl:
    ```bash
    curl --http3 -I https://127.0.0.1:8443/service/foo
    ```
    Or you can access it in chrome with quic enabled:
    ```cmd
    rem Ensure that ca.crt is installed into the Trusted Root Certification Authorities certificate store
    rem Please use "certmgr.exe" instead of "certmgr.msc"
    certmgr.exe /c /add ca.crt /s root

    rem Ensure that all running chrome processes are terminated
    taskkill /f /t /im chrome.exe

    rem Restart chrome with quic enabled and open https://127.0.0.1:8443/
    chrome --enable-quic --origin-to-force-quic-on=127.0.0.1:8443 https://127.0.0.1:8443/
    ```

## Shortcut key of [Editor Online](http://127.0.0.1:8090/editor.html)

- Create or save a source: `Ctrl` + `S`

- Delete a source: `Ctrl` + `0`

## Examples

### Controller

You can create a controller as a http/https service.

- A simple controller.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo
    export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
        return `hello, world`
    }
    ```

- Get request parameters.  
    1. Create a controller with name `greeting`, type `controller` and url `/service/{name}/greeting/{words}`.
        ```typescript
        export default function (ctx: ServiceContext) {
            // get http request body
            String.fromCharCode(...ctx.getBody())

            // get variables in path
            ctx.getPathVariables() // {"name":"zhangsan","words":"hello"}

            // get request form
            ctx.getForm() // {"a":["1","3"],"b":["2"],"c":[""],"d":["4","6"],"e":["5"],"f":[""]}

            // get request url path and params
            ctx.getURL() // {"params":{"a":["1","3"],"b":["2"],"c":[""]},"path":"/service/foo"}
        }
        ```
    2. You can test it using curl:
        ```bash
        curl -XPOST -H "Content-Type: application/x-www-form-urlencoded" "http://127.0.0.1:8090/service/zhangsan/greeting/hello?a=1&b=2&c&a=3" -d "d=4&e=5&f&d=6"
        ```

- Return a custom response.
    ```typescript
    export default function (ctx: ServiceContext): ServiceResponse {
        // return new Uint8Array([104, 101, 108, 108, 111]) // response with body "hello"
        return new ServiceResponse(500, {
            "Content-Type": "text/plain",
        }, new Uint8Array([104, 101, 108, 108, 111]))
    }
    ```

- Websocket server.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo
    export default function (ctx: ServiceContext) {
        const ws = ctx.upgradeToWebSocket() // upgrade http and get a websocket
        console.info(ws.read()) // read a message
        ws.send("hello, world") // write a message
        ws.close() // close the connection
    }
    ```

- Http chunk.
    1. Create a controller with name `foo`, type `controller` and url `/service/foo`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=foo
        export default function (ctx: ServiceContext) {
            ctx.write("hello, chunk 0")
            ctx.flush()
            ctx.write("hello, chunk 1")
            ctx.flush()
            ctx.write("hello, chunk 2")
            ctx.flush()
        }
        ```
    2. You can test it using telnet:
        ```bash
        { echo "GET /service/foo HTTP/1.1"; echo "Host: 127.0.0.1"; echo ""; sleep 1; echo exit; } | telnet 127.0.0.1 8090
        ```

- Read byte(s) from request body. It also can be used as read chunks from a chunked request.
    ```typescript
    export default function (ctx: ServiceContext) {
        const reader = ctx.getReader()

        // String.fromCharCode(...reader.read(10)) // Read 10 bytes from request body as a Uint8Array. Return null if got EOF.

        const arr = []

        let byte = reader.readByte()
        while (byte != -1) { // Return -1 if got EOF
            arr.push(byte)
            byte = reader.readByte()
        }
        
        console.debug(String.fromCharCode(...arr))
    }
    ```

### Module

A module can be imported in the controller.

- A custom module.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=user&type=module
    export const user = {
        name: "zhangsan"
    }
    ```
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo
    import { user } from "./user"

    export default function (ctx: ServiceContext) {
        return `hello, ${user?.name ?? "world"}`
    }
    ```

- A custom module extends Date.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=node_modules/date&type=module
    declare global {
        interface Date {
            toString(layout?: string): string;
        }
        interface DateConstructor {
            toDate(value: string, layout: string): Date;
        }
    }

    const L = {
        "yyyy|yy": ["FullYear"],
        "M{1,2}": ["Month", 1],
        "d{1,2}": ["Date"],
        "H{1,2}": ["Hours"],
        "m{1,2}": ["Minutes"],
        "s{1,2}": ["Seconds"],
        "S{1,3}": ["Milliseconds", 0, -1]
    };

    const toString = Date.prototype.toString;

    Date.prototype.toString = function(layout?: string) {
        if (!layout) {
            return toString();
        }
        for (const l in L) {
            const m = layout.match(new RegExp(`(${l})`));
            if (m) {
                layout = layout.replace(m[1], (this[`get${L[l][0]}`]() + (L[l][1] || 0)).toString().padStart(m[1].length, "0").substr(Math.min(m[1].length * (L[l][2] || 1) * -1, 0), m[1].length));
            }
        }
        return layout;
    };

    Date.toDate = function(value: string, layout: string): Date {
        const t = new Date(0);
        for (const l in L) {
            const r = new RegExp(`(${l})`).exec(layout);
            if (r && r.length) {
                t[`set${L[l][0]}`](Number(value.substr(r.index, r[0].length)) - (L[l][1] || 0));
            }
        }
        return t;
    };

    export default { Date };
    ```
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo
    import "date"

    export default function (ctx) {
        return new Date().toString("yyyy-MM-dd HH:mm:ss.S")
    }
    ```

### Daemon

The daemon is a backend running service with no timeout limit.

- Create a daemon.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo&type=daemon
    export default function () {
        const b = $native("pipe")("default")
        while (true) {
            console.info(b.drain(100, 5000))
        }
    }
    ```

### Builtin

Here are some built-in methods and modules.

- Using console.
    ```typescript
    // ...
    console.error("this is a error message")
    ```

- Using error.
    ```typescript
    // ...
    throw new Error("error message")

    // ...
    throw {
        code: "error code",
        message: "error message"
    }
    ```

- Using buffer.
    ```typescript
    const buf = Buffer.from("hello", "utf8")
    buf // [104, 101, 108, 108, 111]
    buf.toString("base64") // aGVsbG8=
    String.fromCharCode(...buf) // hello
    ```

- Using native module.
    ```typescript
    // bqueue or pipe
    const b = $native("pipe")("default")
    //const b = $native("bqueue")(99)
    b.put(1)
    b.put(2)
    b.drain(4, 2000) // [1, 2]

    // db
    $native("db").query("select name from script") // [{"name":"foo"}, {"name":"user"}]

    // decimal
    const Decimal = $native("decimal"),
        d1 = Decimal("0.1"),
        d2 = Decimal("0.2")
    d2.add(d1) // 0.3
    d2.sub(d1) // 0.1
    d2.mul(d1) // 0.02
    d2.div(d1) // 2

    // email
    const emailc = $native("email")("smtp.163.com", 465, username, password)
    emailc.send(["zhangsan@abc.com"], "greeting", "hello, world")
    emailc.send(["zhangsan@abc.com"], "greeting", "hello, world", [{
        name: "hello.txt",
        contentType: "text/plain",
        base64: "aGVsbG8=",
    }])

    // crypto
    const cryptoc = $native("crypto")
    // hash
    cryptoc.createHash("md5").sum("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // "e4d7f1b4ed2e42d15898f4b27b019da4"
    // hmac
    cryptoc.createHmac("sha1").sum("hello, world", "123456").map(c => c.toString(16).padStart(2, "0")).join("") // "9a231f1dd39a4ff6ea778a5640d1498794f8a9f8"
    // rsa
    // privateKey and publicKey mentioned is PKCS#1 format
    const rsa = cryptoc.createRsa(),
        { privateKey, publicKey } = rsa.generateKey();
    String.fromCharCode(
        ...rsa.decrypt(
            rsa.encrypt("hello, world", publicKey),
            privateKey,
        )
    ) // "hello, world"
    rsa.verifyPss(
        "hello, world",
        rsa.signPss("hello, world", privateKey, "sha256"),
        publicKey,
        "sha256",
    ) // true

    // file
    const filec = $native("file")
    filec.write("greeting.txt", "hello, world")
    String.fromCharCode(...filec.read("greeting.txt")) // "hello, world"

    // http
    const { status, header, data } = $native("http")({
        //caCert: "", // ca certificates for http client
        //cert: "", key: "", // private key and certificate/public key for http client auth
        //insecureSkipVerify: true // disable verify server certificate
        //proxy: "http://127.0.0.1:5566" // proxy server
    }).request("GET", "https://www.baidu.com")
    status // 200
    header // {"Content-Length":["227"],"Content-Type":["text/html"]...]}
    data.toString() // "<html>..."

    // image
    const imagec = $native("image")
    const img0 = imagec.create(100, 200), // create a picture with width 100 and height 200
        img1 = imagec.parse($native("http")().request("GET", "https://www.baidu.com/img/flexible/logo/plus_logo_web_2.png").data) // read a picture from network
    img0.toBytes() // convert this picture to a byte array

    // template
    const content = $native("template")("greeting", { // read template greeting.tpl and render with input
        name: "this is name",
    })

    // xml, see https://github.com/antchfx/xpath for syntax
    const doc = $native("xml")(`
        <Users>
            <User>
                <ID>1</ID>
                <Name>zhangsan</Name>
            </User>
            <User>
                <ID>2</ID>
                <Name>lisi</Name>
            </User>
        </Users>
    `)
    doc.find("//user[id=2]/name").pop().innerText() // lisi
    doc.findOne("//user[1]/name").innerText() // zhangsan
    doc.findOne("//user[1]").findOne("name").innerText() // zhangsan
    ```

### Advance

- Return a view with asynchronous vues.
    1. Create a template with lang `html`.
        ```html
        <!-- http://127.0.0.1:8090/editor.html?name=index&type=template&lang=html -->
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="utf-8" />
            <title>{{ .title }}</title>
            <style>
                * {
                    margin: 0;
                    padding: 0;
                }
                html, body {
                    width: 100%;
                    height: 100%;
                }
                html {
                    overflow: hidden;
                }
            </style>
        </head>
        <body>
            <script src="https://cdn.bootcdn.net/ajax/libs/vue/2.7.14/vue.min.js"></script>
            <script src="https://cdn.bootcdn.net/ajax/libs/vue-router/3.6.5/vue-router.min.js"></script>
            <script src="https://unpkg.com/http-vue-loader"></script>
            <router-view id="container"></router-view>
            <script>
                const router = new VueRouter({
                    mode: "hash"
                })
                router.beforeEach((to, from, next) => {
                    if (to.matched.length) { // 当前路由已匹配上
                        next() // 直接渲染当前路由
                        return
                    }
                    router.addRoute({ // 动态添加路由
                        path: to.path,
                        component: httpVueLoader(`../resource${to.path === "/" ? "/index" : to.path}.vue`), // 远程加载组件
                    })
                    next(to.path) // 重新进入 beforeEach 方法
                })
                new Vue({ router }).$mount("#container")
            </script>
        </body>
        </html>
        ```
    2. Create a resource with lang `vue` and url `/resource/greeting.vue`.
        ```html
        <!-- http://127.0.0.1:8090/editor.html?name=greeting&type=reaource&lang=vue -->
        <template>
            <p>hello, {{ name }}</p>
        </template>

        <script>
            module.exports = {
                data: function() {
                    return {
                        name: "world",
                    }
                }
            }
        </script>

        <style scoped>
            p {
                color: #000;
            }
        </style>
        ```
    3. Create a controller with url `/service/`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=index
        export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
            return $native("template")("index", {
                title: "this is title",
            })
        }
        ```
    4. You can preview at `http://127.0.0.1:8090/service/#/greeting`

- Upload file.
    1. Create a resource with lang `html` and url `/resource/foo.html`.
        ```html
        <!-- http://127.0.0.1:8090/editor.html?name=foo&type=resource&lang=html -->
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <link rel="stylesheet" href="//unpkg.com/element-ui/lib/theme-chalk/index.css">
        </head>
        <body>
            <div id="app" v-cloak>
                <el-upload
                    action="/service/foo"
                    accept="image/jpeg"
                    :auto-upload="true">
                    <el-button icon="el-icon-upload2">Upload</el-button>
                </el-upload>
            </div>
        </body>
        <script src="//cdn.bootcdn.net/ajax/libs/vue/2.7.14/vue.js"></script>
        <script src="//unpkg.com/element-ui"></script>
        <script>
            new Vue({ el: "#app" })
        </script>
        </html>
        ```
    2. Create a controller with url `/service/foo`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=foo
        export default function (ctx: ServiceContext) {
            const file = ctx.getFile("file"),
                hash = $native("crypto").md5(file.data).map(c => c.toString(16).padStart(2, "0")).join("")
            console.info(hash)
        }
        ```
    3. You can preview at `http://127.0.0.1:8090/resource/foo.html`. You can also run it using curl:
        ```bash
        # Upload a file
        curl -F "file=@./abc.txt; filename=abc.txt;" http://127.0.0.1:8090/service/foo
        ```

- Download a mp4 using HTTP Range.
    1. Create a controller with url `/service/foo`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=foo
        export default function (ctx: ServiceContext) {
            const name = "a.mp4"

            const filec = $native("file"),
                size = filec.stat(name).size()

            const range = ctx.getHeader().Range
            if (!range) {
                return new ServiceResponse(200, {
                    "Accept-Ranges": "bytes",
                    "Content-Length": size + "",
                    "Content-Type": "video/mp4",
                })
            }

            const ranges = range.substring(6).split("-"),
                slice = 1024 * 1024 * 2, // The slice size is 2 MB
                start = Number(ranges[0]),
                end = Math.min(Number(ranges[1]) || (start + slice - 1), size - 1)
            
            const buf = filec.readRange(name, start, end - start + 1) // slice the mp4 file from [start, end + 1)

            return new ServiceResponse(206, {
                "Content-Range": `bytes ${start}-${end}/${size}`,
                "Content-Length": end - start + 1 + "",
                "Content-Type": "video/mp4",
            }, buf)
        }
        ```
    2. You can preview at `http://127.0.0.1:8090/service/foo` in browser.

- Play a video online using HTTP-FLV.
    1. Create a flv file under `files/` using ffmpeg:
        ```bash
        ffmpeg \
            -i a.mp4 \
            -vcodec libx264 -r 25 -b:v 800000 \
            -acodec aac -ac 2 -ar 44100 -ab 128k \
            -af "loudnorm" \
            -vf "scale=-1:720" \
            -y a.flv
        ```
        > We need encode with libx264. Otherwise, using flv.js to pull the stream may cause an error: "DemuxException: type = CodecUnsupported, info = Flv: Unsupported codec in video frame: 2"
    2. Create a controller with url `/service/foo`. 
        ```typescript
        export default function (ctx: ServiceContext) {
            const buf = $native("file").read("a.flv")

            // send a chunk: flv header(9 bytes) + previousTagSize0(4 bytes)
            ctx.write(new Uint8Array(buf.slice(0, 9 + 4)))
            ctx.flush()

            let i = 9 + 4
            while (i < buf.length) {
                const dataSize = (buf[i + 1] << 16) + (buf[i + 2] << 8) + buf[i + 3],
                    tagSize = 11 + dataSize,
                    previousTagSize = (buf[i + tagSize] << 24) + (buf[i + tagSize + 1] << 16) + (buf[i + tagSize + 2] << 8) + buf[i + tagSize + 3]
                if (tagSize != previousTagSize) {
                    throw new Error("Invalid previous tag size: " + tagSize + ", expected: " + previousTagSize)
                }

                // send a chunk: flv tag(each video tag is a frame of the video, total 11 + dataSize bytes) + previousTagSize(4 bytes)
                ctx.write(new Uint8Array(buf.slice(i, i + tagSize + 4)))
                ctx.flush()

                i = i + tagSize + 4
            }
        }
        ```
    3. Create a resource with lang `html` and url `/resource/foo.html`.
        ```html
        <script src="https://cdn.bootcdn.net/ajax/libs/flv.js/1.6.2/flv.min.js"></script>
        <video id="videoElement"></video>
        <script>
            if (flvjs.isSupported()) {
                const flvPlayer = flvjs.createPlayer({
                    type: "flv",
                    url: "/service/foo",
                    enableWorker: true, // https://github.com/bilibili/flv.js/issues/322
                })
                flvPlayer.attachMediaElement(document.getElementById("videoElement"))
                flvPlayer.load()
                flvPlayer.play()
            }
        </script>
        ```
    4. You can preview at `http://127.0.0.1:8090/resource/foo.html`.

- Create an rtmpd and convert the rtmp stream into an HTTP-FLV stream.
    1. Create a daemon with name `rtmpd` and start.
        ```typescript
        export default function () {
            const c = new RtmpConnection($native("socket")("tcp").listen(1935).accept())

            // 1. 握手阶段
            c.handshake()

            // 2. 建立连接阶段
            // 2.1 客户端发起连接请求
            c.readChunk()
            // 2.2 服务器设置客户端的应答窗口大小
            c.write(c.toChunk(
                2, // Chunk Stream ID: 2
                0x05, // Type ID: Window Acknowledgement Size
                [0x00, 0x4c, 0x4b, 0x40], // Window acknowledgement size: 5000000
            ))
            // 2.3 服务器设置客户端的发送带宽大小
            c.write(c.toChunk(
                2, // Chunk Stream ID: 2
                0x06, // Type ID: Set Peer Bandwidth
                [
                    0x00, 0x4c, 0x4b, 0x40, // Window acknowledgement size: 5000000
                    0x02, // Limit Type: Dynamic
                ],
            ))
            // 2.4 服务器设置客户端的接收块大小
            c.write(c.toChunk(
                2, // Chunk Stream ID: 2
                0x01, // Type ID: Set Chunk Size
                [
                    // 0x00, 0x00, 0x10, 0x00, // Chunk size: 4096
                    0x00, 0x00, 0xea, 0x60, // Chunk size: 60000
                ],
            ))
            // 2.5 服务器响应连接结果
            c.write(c.toChunk(3, 0x14, AMF0.encodes(
                "_result",
                1,
                {
                    fmsVer: "FMS/3,0,1,123",
                    capabilities: 31,
                },
                {
                    levelc: "status",
                    code: "NetConnection.Connect.Success",
                    description: "Connection succeeded.",
                    objectEncoding: 0,
                },
            )))
            // 2.6 客户端设置服务器的接收块大小
            const chunkSizeData = c.readChunk().data
            c.setMaxChunkSize((chunkSizeData[0] << 24) + (chunkSizeData[1] << 16) + (chunkSizeData[2] << 8) + chunkSizeData[3])

            // 3. 建立流阶段
            // 3.1 客户端通知服务器释放推流点
            c.readChunk()
            // 3.2 客户端通知服务器准备推流点
            c.readChunk()
            // 3.3 客户端发起创建流请求
            c.readChunk()
            // 3.4 服务器响应创建流结果
            c.write(c.toChunk(3, 0x14, AMF0.encodes(
                "_result",
                4, // Number
                null, // Null
                1, // Number
            )))

            // 4. 推流阶段
            // 4.1 客户端发起创建绑定推流点请求
            c.readChunk()
            // 4.2 服务器响应创建绑定推流点结果
            c.write(c.toChunk(5, 0x14, AMF0.encodes(
                "onStatus", // Command Name: onStatus
                0, // Transaction ID: 0
                null, // Command Object: Null
                { // Info Object
                    level: "status",
                    code: "NetStream.Publish.Start", // notify client to push stream
                    description: "/ is now published.",
                },
            )))
            // 4.3 客户端向服务器设置媒体元数据
            const metaData = c.readChunk().data.slice(16)

            const cache = $native("cache")
            cache.set("FLV_HEADER_TAG", new Uint8Array([
                // FLV Header
                0x46, 0x4c, 0x56, // 'F'、'L'、'V'
                0x01, // 版本号为 1
                0x05, // 即 0b00000101，其中第 6 位表示是否存在音频，第 8 位表示是否存在视频，其余位均为 0
                0x00, 0x00, 0x00, 0x09, // FLV Header 的字节长度
                // Previous Tag Size 0
                0x00, 0x00, 0x00, 0x00,
            ]), -1)
            cache.set("FLV_META_TAG", FLVTag.encode(0x12, metaData, 0), -1) // Script Tag

            while (true) {
                // 4.4 客户端向服务器推送媒体数据
                const { header, data } = c.readChunk()
                if (header.messageTypeId === 0x14) {
                    // 客户端通知服务器解绑释放推流点请求
                    console.log(header.messageTypeId, String.fromCharCode(...data))
                    c.close()
                    $native("event").emit("HTTPFLV_STOP", 0)
                    break
                }
                if (!data.length) {
                    throw new Error("no data read")
                }

                const tag = FLVTag.encode(header.messageTypeId, data, c.clocks[header.chunkStreamId]) // messageTypeId: 8 = Audio, 9 = Video, 18 = Script

                // https://zhuanlan.zhihu.com/p/611128149
                if (header.messageTypeId === 0x08) { // Audio
                    const soundFormat = (data[0] >> 4) & 0x0f
                    if ((soundFormat == 10 || soundFormat == 13) && data[1] == 0) {
                        // AAC Sequence Header
                        cache.set("FLV_AACHEADER_TAG", FLVTag.encode(header.messageTypeId, data, 0), -1)
                    }
                }
                if (header.messageTypeId === 0x09) { // Video
                    const videoCodec = data[0] & 0x0f, // 第一个字节后 4 位表示编码 ID：7 表示 AVC
                        frameType = data[0] >> 4 & 0b0111, // 第一个字节的前 4 位表示帧类型：1 表示关键帧
                        isExHeader = (data[0] >> 4 & 0b1000) !== 0
                    if (isExHeader) {
                        throw new Error("video tag ex header not implemented")
                    }
                    if ((videoCodec == 7 || videoCodec == 12 || videoCodec == 13) && frameType == 1 && data[1] == 0) {
                        // AVC Sequence Header
                        cache.set("FLV_AVCHEADER_TAG", FLVTag.encode(header.messageTypeId, data, 0), -1)
                    }
                }

                $native("event").emit("HTTPFLV_TAG", tag)
            }
        }

        class RtmpConnection {
            private socket: {
                read(size?: number): NativeByteArray;
                write(data: string | Uint8Array | NativeByteArray): number;
                close(): void;
            }

            private maxChunkSize = 128

            public clocks = {}

            public headers = {} as { [csid: string]: { chunkStreamId: number, timestamp: number, messageLength: number, messageTypeId: number, messageStreamId: number } }

            constructor(socket) {
                this.socket = socket
            }

            public handshake() { // complex handshake is not supported
                // read c0 and c1
                const c0 = this.socket.read(1),
                    c1 = this.socket.read(1536)

                // write s0, s1 and s2
                this.socket.write(c0)
                this.socket.write(c1)
                this.socket.write(c1)

                // read c2
                this.socket.read(1536)
            }

            public setMaxChunkSize(size: number) {
                if (!size) {
                    throw new Error("unknown chunk size: " + size)
                }
                this.maxChunkSize = size
            }

            public read(size?: number) {
                if (size === 0) {
                    return [] as NativeByteArray
                }

                return this.socket.read(size)
            }

            private readUintBE(size: number) {
                return parseInt(this.read(size).map(i => i.toString(2).padStart(8, "0")).join(""), 2)
            }

            public write(data: string | Uint8Array | NativeByteArray): number {
                return this.socket.write(data)
            }

            public close(): void {
                return this.socket.close()
            }

            public readChunk(size?: number) {
                // 1. Chunk Header
                let header = new Object() as typeof this.headers[0]
                // 1.1 Basic Header（1, 2 or 3 bytes）：Format + Chunk Stream ID
                const a = this.read(1).pop()
                if (!a) {
                    throw new Error("failed to read format and chunk stream id in header")
                }
                const format = (a & 0b11000000) >> 6 // Format
                header.chunkStreamId = a & 0b111111 // Chunk Stream ID，即块流 ID，用于区分消息信道，取值范围 [3, 65599]，值 0, 1 和 2 被保留，3 ~ 8用于固定用途，9 ~ 65599 用于自定义
                // 0 表示 Basic Header 总共要占用 2 个字节
                // 1 表示 Basic Header 总共要占用 3 个字节
                // 2 代表该 chunk 是控制信息和一些命令信息
                // 3 代表该 chunk 是客户端发出的 AMF0 命令以及服务端对该命令的应答
                // 4 代表该 chunk 是客户端发出的音频数据，用于 publish
                // 5 代表该 chunk 是服务端发出的 AMF0 命令和数据
                // 6 代表该 chunk 是服务端发出的音频数据，用于 play；或客户端发出的视频数据，用于 publish
                // 7 代表该 chunk 是服务端发出的视频数据，用于 play
                // 8 代表该 chunk 是客户端发出的 AMF0 命令，专用来发送： getStreamLength, play, publish
                if (header.chunkStreamId === 0b000000) { // Chunk Stream ID use 6 + 8 bits
                    const [b] = this.read(1)
                    header.chunkStreamId = b + 64
                } else if (header.chunkStreamId === 0b000001) { // Chunk Stream ID use 6 + 8 + 8 bits
                    const [b, c] = this.read(2)
                    header.chunkStreamId = (c << 16) + b + 64 // 第三个字节 * 256 + 第二个字节 + 64
                }

                // 1.2 Message Header（0, 3, 7 or 11 bytes）
                switch (format) {
                    case 0x00: // Type 0 (Full), 12 bytes
                        header.timestamp = this.readUintBE(3) // Timestamp，单位毫秒
                        header.messageLength = this.readUintBE(3) // Message Length
                        header.messageTypeId = this.readUintBE(1) // Message Type ID
                        header.messageStreamId = this.readUintBE(4) // Message Stream ID
                        break;
                    case 0x01: // Type 1 (Relative Large), 8 bytes
                        header.timestamp = this.readUintBE(3) // Timestamp Delta
                        header.messageLength = this.readUintBE(3) // Message Length
                        header.messageTypeId = this.readUintBE(1) // Message Type ID
                        // Type 1 情况下省略了 Message Stream ID，这个 ID 与上一个 chunk message 相同
                        break;
                    case 0x02: // Type 2 (Relative Timestamp Only), 4 bytes
                        header.timestamp = this.readUintBE(3) // Timestamp
                        // Type 2 情况下省略了 Message Length、Message Type ID 和 Message Stream ID，省略的这个与上一个 chunk message 相同
                        break;
                    case 0x03: // Type 3 (Relative Single Byte), 1 bytes
                        // Type 3 情况下 Message Header 全部省略了，跟上一个 chunk message 完全相同
                        break;
                    default:
                        throw new Error("unknown chunk format: " + format)
                }
                header = this.headers[header.chunkStreamId] = {
                    ...this.headers[header.chunkStreamId],
                    ...header,
                }

                // 1.3 Extended Timestamp（0 or 4 bytes）
                // 通常我们使用 Timestamp 来表示时间戳（包括绝对时间戳和 Timestamp delta），但当时间戳的值超过 16777215（即 0xffffff）时，Timestamp 会被置为 0xffffff，此时我们使用 Extended Timestamp 字段并由该字段表示时间戳
                if (header.timestamp === 0xffffff) {
                    header.timestamp = this.readUintBE(4) // Extended Timestamp
                }

                if (size === undefined) { // 当前 chunk 没有被拆分多个
                    if (format === 0x00) {
                        // 视频流（或音频流）的第一个时间戳为绝对时间戳，后续（type-1 和 type-2 chunk）的时间戳均为 timestamp delta，即当前时间戳与上一个时间戳的差值
                        this.clocks[header.chunkStreamId] = header.timestamp
                    } else {
                        this.clocks[header.chunkStreamId] += header.timestamp
                    }
                }

                if (header.messageTypeId === 0) {
                    throw new Error("message type id can not be 0: " + header.messageTypeId)
                }

                const msgLength = size === undefined ? header.messageLength : size
                if (!msgLength) {
                    throw new Error("unknown message length: " + msgLength)
                }

                // 2. Chunk Data
                const data = this.read(Math.min(this.maxChunkSize, msgLength)) // A message consists of one or more blocks, such as 1...n, and the chunk data length of the nth block is `messageLength - (n-1) * maximumChunkSize` (maximumChunkSize defaults to 128)
                while (data.length < msgLength) {
                    data.push(...this.readChunk(msgLength - data.length).data)
                }

                return {
                    header,
                    data,
                }
            }

            public toChunk(chunkStreamId: number, typeId: number, body: Uint8Array | number[]): Uint8Array {
                return new Uint8Array([
                    // Header
                    chunkStreamId & 0b00111111, // Format and Chunk Stream ID
                    0x00, 0x00, 0x00, // Timestamp
                    body.length >> 16 & 0xff, body.length >> 8 & 0xff, body.length & 0xff, // Body size
                    typeId & 0xff, // Type ID: AMF0 Command
                    0x00, 0x00, 0x00, 0x00, // Stream id
                    // Body
                    ...body,
                ])
            }
        }

        class AMF0 {
            public static decodea(input: NativeByteArray, start: number): { value: number | boolean | string | object | null; end: number; } {
                if (input[start] === 0x00) { // Number
                    return {
                        value: 0, // TODO
                        end: start + 1 + 8,
                    }
                }
                if (input[start] === 0x01) {
                    return {
                        value: input[start + 1],
                        end: start + 1 + 1,
                    }
                }
                if (input[start] === 0x02) { // String
                    const length = (input[start + 1] << 8) + input[start + 2]
                    return {
                        value: String.fromCharCode(...input.slice(start + 3, start + 3 + length)),
                        end: start + 1 + 2 + length,
                    }
                }
                if (input[start] === 0x08) { // ECMA Array
                    const arrayLength = (input[start + 1] << 24) + (input[start + 2] << 16) + (input[start + 3] << 8) + input[start + 4],
                        output = []
                    let i = start + 1 + 4
                    for (let c = 0; c < arrayLength; c++) {
                        const propertyNameLength = (input[i] << 8) + input[i + 1]
                        const propertyName = String.fromCharCode(...input.slice(i + 2, i + 2 + propertyNameLength))
                        const { value: propertyValue, end: propertyValueEnd } = AMF0.decodea(input, i + 2 + propertyNameLength)
                        output.push({
                            [propertyName]: propertyValue,
                        })
                        i = propertyValueEnd
                    }
                    if (!(input[i] === 0x00 && input[i + 1] === 0x00 && input[i + 2] === 0x09)) {
                        throw new Error("end of object marker is expect")
                    }
                    return {
                        value: output,
                        end: i + 3,
                    }
                }
                throw new Error("unknown amf0 type: " + input[start] + "@" + start)
            }

            public static decodes(input: NativeByteArray): (number | boolean | string | object | null)[] {
                const output = []
                for (let i = 0; i < input.length;) {
                    const { value, end } = AMF0.decodea(input, i)
                    output.push(value)
                    i = end
                }
                return output
            }

            public static encodes(...values: (number | boolean | string | object | null)[]): Uint8Array {
                return new Uint8Array([].concat.apply([], values.map(v => {
                    if (typeof v === "number") {
                        if (v === 0) {
                            return [
                                0x00, // Type is Number
                                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                            ]
                        }
                        const [a, b] = (v.toString(2) + ".").split("."),
                            ab = a + b,
                            s = v > 0 ? "0" : "1",
                            e = (1023 + (ab.indexOf("1") * -1 || (a.length - 1))).toString(2).padStart(11, "0"),
                            t = ab.substring(ab.indexOf("1") + 1).padEnd(52, "0")
                        return [
                            0x00, // Type is Number
                            ...(s + e + t).split(/([01]{8})/).filter(i => i).map(i => parseInt(i, 2)), // Number value
                        ]
                    }
                    if (typeof v === "boolean") {
                        return [
                            0x01, // Type is Boolean
                            v ? 0x01 : 0x00, // Boolean value
                        ]
                    }
                    if (typeof v === "string") {
                        return [
                            0x02, // Type is String
                            v.length >> 8 & 0xff, v.length & 0xff, // String length
                            ...v.split("").map(i => i.charCodeAt(0)), // String value
                        ]
                    }
                    if (typeof v === "object" && v !== null) {
                        return [
                            0x03, // Type is Object
                            ...Object.keys(v).reduce((p, c) => {
                                p.push(...[
                                    // Property name
                                    c.length >> 8 & 0xff, c.length & 0xff, // String length
                                    ...c.split("").map(i => i.charCodeAt(0)), // String value
                                    // Property value
                                    ...AMF0.encodes(v[c]),
                                ])
                                return p
                            }, []),
                            0x00, 0x00, 0x09, // End of Object Marker
                        ]
                    }
                    if (v === null) {
                        return [
                            0x05 // Type is Null
                        ]
                    }
                    throw new Error("unknown amf0 value: " + v)
                })))
            }
        }

        class FLVTag {
            public static encode(dataType: number, data: number[], timestamp: number = 0) {
                if (dataType !== 8 && dataType !== 9 && dataType !== 18) {
                    throw new Error("unknown flv tag data type: " + dataType)
                }
                const dataSize = data.length,
                    previousTagSize = 11 + dataSize
                return new Uint8Array([
                    // Tag Header
                    dataType, // DataType
                    (dataSize >> 16) & 0xff, (dataSize >> 8) & 0xff, dataSize & 0xff, // DataSize
                    (timestamp >> 16) & 0xff, (timestamp >> 8) & 0xff, timestamp & 0xff, (timestamp >> 24) & 0xff, // Timestamp
                    0x00, 0x00, 0x00, // Stream Id
                    // Tag Data
                    ...data,
                    // Previous Tag Size
                    (previousTagSize >> 24) & 0xff, (previousTagSize >> 16) & 0xff, (previousTagSize >> 8) & 0xff, previousTagSize & 0xff,
                ])
            }
        }
        ```
    2. Create a controller with url `/service/httpflv`.
        ```typescript
        export default function (ctx: ServiceContext) {
            ctx.resetTimeout(30 * 60 * 1000)

            const cache = $native("cache")

            let playing = false, c = 0

            const a = $native("event").on("HTTPFLV_TAG", function (tag) {
                if (!playing) {
                    [
                        cache.get("FLV_HEADER_TAG"),
                        cache.get("FLV_META_TAG"),
                        cache.get("FLV_AACHEADER_TAG"),
                        cache.get("FLV_AVCHEADER_TAG"),
                    ].filter(i => i).forEach(i => {
                        ctx.write(i)
                    })
                    playing = true
                }

                ctx.write(tag)
                if (++c % 10 == 0) {
                    ctx.flush()
                }
            })

            const b = $native("event").on("HTTPFLV_STOP", function () {
                a.cancel()
                b.cancel()
            })
        }
        ```
    3. Create a resource with lang `html` and url `/resource/httpflv`.
        ```html
        <script src="https://cdn.bootcdn.net/ajax/libs/flv.js/1.6.2/flv.min.js"></script>
        <video id="videoElement" onclick="player.play()" style="width: 100%; height: 100%;"></video>
        <script>
            if (flvjs.isSupported()) {
                var player = flvjs.createPlayer({
                    type: "flv",
                    url: "/service/1_httpflv_srv",
                    enableWorker: true,
                    enableStashBuffer: true,
                })
                player.attachMediaElement(document.getElementById("videoElement"))
                player.load()
                // player.play() // play() failed here because the user didn't interect with the document first. see https://goo.gl/xX8pDD
            }
        </script>
        ```
    4. Push a stream using ffmpeg.
        ```bash
        ffmpeg -re \
            -stream_loop -1 \
            -i "https://s.xlzys.com/play/9b64Eq9e/index.m3u8" \
            -vcodec libx264 -r 25 -b:v 800000 \
            -acodec aac -ac 2 -ar 44100 -ab 128k \
            -af "loudnorm" \
            -vf "scale=-1:720" \
            -f flv \
            -threads 5 -preset ultrafast \
            rtmp://127.0.0.1:1935
        ```
    5. You can preview at `http://127.0.0.1:8090/resource/httpflv` and click the screen to play.

- Create a smtpd using socket module.
    1. Create a daemon.
        ```typescript
        export default function (ctx: ServiceContext) {
            const tcpd = $native("socket")("tcp").listen(25)
            while(true) {
                const c = tcpd.accept()
                console.debug(toEmail(readData(c)))
            }
        }

        function readData(connection) {
            connection.write("220 My Mail Sever\n")

            let data = "",
                s = String.fromCharCode(...connection.readLine())
            while (s.length) {
                switch (s.substring(0, 4).replace(/[\r\n]*$/, "")) {
                    case "HELO":
                    case "EHLO":
                        connection.write("250 OK\n")
                        break
                    case "MAIL":
                        connection.write("250 OK\n")
                        break
                    case "RCPT":
                        connection.write("250 OK\n")
                        break
                    case "DATA":
                        connection.write("354 OK\n")
                        break
                    case ".":
                        connection.write("250 OK\n")
                        break
                    case "QUIT":
                        connection.write("221 Bye\n")
                        connection.close()
                        return data
                    default:
                        data += s
                        break
                }
                s = String.fromCharCode(...connection.readLine())
            }
            return null
        }

        function toEmail(data) {
            data = (data || "").replace(/\r\n/g, "\n")
            return {
                subject: data.match(/^Subject: (.*)$/m)?.[1],
                from: data.match(/^From: .*(<.*>)$/m)?.[1],
                to: data.match(/^To: .*(<.*>)$/m)?.[1],
                body: Buffer.from(data.match(/\n\n(.*)/m)?.[1] || "", "base64").toString(),
            }
        }
        ```
    2. You can test it using telnet such as:
        ```bash
        telnet 127.0.0.1 25
        ```
        ```
        HELO abc.com
        MAIL FROM: <noone@abc.com>
        RCPT TO: <zhangsan@127.0.0.1>
        DATA
        To: zhangsan@127.0.0.1
        From: noone@abc.com
        Subject: greeting

        aGVsbG8=
        .
        QUIT
        ```

- Using DLNA to cast a video on device in LAN.
    1. Create a controller with url `/dlna`.
        ```typescript
        export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
            const videoUri = "https://s.xlzys.com/play/9b64Eq9e/index.m3u8" // m3u8 or mp4

            // 1. Search for a device using SSDP
            // create a virtual udp connection using a random port
            const ssdpc = $native("socket")("udp").listen(0)
            // send a M-Search broadcast packet to 239.255.255.250:1900 (the source port is exactly what "ssdpc" used)
            ssdpc.write([
                "M-SEARCH * HTTP/1.1",
                "HOST: 239.255.255.250:1900", // broadcast address (239.255.255.250:1900 is the default ip and port of SSDP broadcast)
                `MAN: "ssdp:discover"`,
                "MX: 3", // maximum waiting time
                "ST: urn:schemas-upnp-org:service:AVTransport:1", // service type
                "",
            ].join("\r\n"), "239.255.255.250", 1900)
            // wait for device response
            const devices = [...new Array(1)].reduce((p) => {
                const b = Buffer.from(ssdpc.read()).toString() // listen on the port that "ssdpc" used and read bytes
                return [
                    ...p,
                    {
                        location: b.match(/LOCATION:\s([^\r\n]*)/i)[1],
                        server: b.match(/SERVER:\s([^\r\n]*)/i)[1],
                    },
                ]
            }, [])

            // 2. Query controlURL from a device
            const [{ location }] = devices // here we just use the first device for example
            const host = location.match(/^https?:\/\/[^\/]+/)[0]
            const service = $native("xml")($native("http")().request("get", location).data.toString()) // request location and parse it into an XML dom
                .find("//service").map(i => {
                    return {
                        serviceType: i.findOne("servicetype").innerText(),
                        serviceId: i.findOne("serviceid").innerText(),
                        controlURL: i.findOne("controlurl").innerText(),
                    }
                })
                .filter(i => i.serviceType === "urn:schemas-upnp-org:service:AVTransport:1")
                .pop()
            if (!service) {
                throw new Error("service not found: AVTransport")
            }

            // 3. SetAVTransportURI
            return $native("http")().request("post",
                host + service.controlURL,
                {
                    "Content-Type": `text/xml;charset="utf-8"`,
                    "SOAPACTION": `"urn:schemas-upnp-org:service:AVTransport:1#SetAVTransportURI"`, // SOAPACTION: "${serviceType}#${action}"
                },
                `<?xml version="1.0" encoding="utf-8"?>
        <s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
            <s:Body>
                <u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
                    <InstanceID>0</InstanceID>
                    <CurrentURI>${videoUri}</CurrentURI>
                    <CurrentURIMetaData></CurrentURIMetaData>
                </u:SetAVTransportURI>
            </s:Body>
        </s:Envelope>`
            ).data.toString()
        }
        ```
    2. Now you can call `http://127.0.0.1:8090/service/dlna` to case a video on device in LAN.

- Create a webdav server.
    1. Create a controller with url `/service/webdav/{path}` and method `Any`.
        ```typescript
        const webdav = new class {
            public propfind(ctx: ServiceContext) {
                return new ServiceResponse(207, {
                    "Content-Type": "text/xml; charset=utf-8"
                }, `<?xml version="1.0" encoding="UTF-8"?>
                    <D:multistatus xmlns:D="DAV:">
                        <D:response>
                            <D:href>/dav/root/</D:href>
                            <D:propstat>
                                <D:prop>
                                    <D:displayname>root</D:displayname>
                                    <D:getlastmodified>Mon, 17 Apr 2023 08:11:39 GMT</D:getlastmodified>
                                    <D:resourcetype>
                                        <D:collection xmlns:D="DAV:" />
                                    </D:resourcetype>
                                </D:prop>
                                <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                        </D:response>
                        ${
                            $native("db").query("select name, content, type from source").map(i => {
                                return `
                        <D:response>
                            <D:href>/dav/root/${i.name}.${i.type}</D:href>
                            <D:propstat>
                                <D:prop>
                                    <D:resourcetype></D:resourcetype>
                                    <D:getcontentlength>${i.content.length}</D:getcontentlength>
                                    <D:displayname>${i.name}.${i.type}</D:displayname>
                                    <!--<D:getlastmodified>Sun, 19 Dec 2021 13:36:15 GMT</D:getlastmodified>-->
                                    <D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype>
                                </D:prop>
                                <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                        </D:response>
                                `;
                            }).join("")
                        }                
                    </D:multistatus>`.replace(/>[\s\n\r]*</g, "><")
                );
            }

            public get(ctx: ServiceContext) {
                const [name, type] = ctx.getPathVariables().path.split(".");
                return $native("db").query("select content from source where type = ? and name = ?", type, name).pop()?.content || "";
            }

            public put(ctx: ServiceContext) {
                const [name, type] = ctx.getPathVariables().path.split(".");
                $native("db").exec("update source set content = ? where type = ? and name = ?", Buffer.from(ctx.getBody()).toString(), type, name);
                return new ServiceResponse(201, {}, "Created");
            }
        }

        export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
            switch (<string>ctx.getMethod()) {
                case "PROPFIND":
                    return webdav.propfind(ctx);
                case "GET":
                    return webdav.get(ctx);
                case "PUT":
                    return webdav.put(ctx);
                default:
                    throw new Error("not implemented")
            }
        }
        ```
