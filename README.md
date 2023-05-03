# Cube

A simple web server that can be developed online using typescript.

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

1. Make sure `ca.key`, `ca.crt`, `server.key` and `server.crt` is created:
    ```bash
    make crt
    ```

2. Start the server:
    ```bash
    ./cube \
        -n 8 \ # create a pool with 8 virtual machines
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
        console.info("The body of request is:", String.fromCharCode(...ctx.getBody())) // print http request body
        return `hello, world`
    }
    ```

- Get request parameters.
    ```bash
    curl -XPOST -H "Content-Type: application/x-www-form-urlencoded" "http://127.0.0.1:8090/service/foo?a=1&b=2&c&a=3" -d "d=4&e=5&f&d=6"
    ```
    ```typescript
    export default function (ctx: ServiceContext) {
        ctx.getForm() // {"a":["1","3"],"b":["2"],"c":[""],"d":["4","6"],"e":["5"],"f":[""]}
        ctx.getURL() // {"params":{"a":["1","3"],"b":["2"],"c":[""]},"path":"/service/foo"}
    }
    ```

- Return a custom response.
    ```typescript
    export default function (ctx: ServiceContext): ServiceResponse {
        // return new Uint8Array([104, 101, 108, 108, 111]) // response with body "hello"
        return new ServiceResponse(500, {
            "Content-Type": "text/plain"
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

- Using native module.
    ```typescript
    // base64
    const base64 = $native("base64")
    base64.encode("hello") // aGVsbG8=
    base64.decode("aGVsbG8=") // [104, 101, 108, 108, 111]
    String.fromCharCode(...base64.decode("aGVsbG8=")) // hello

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
    const c = $native("email")("smtp.163.com", 465, username, password)
    c.send(["zhangsan@abc.com"], "greeting", "hello, world")
    c.send(["zhangsan@abc.com"], "greeting", "hello, world", [{
        name: "hello.txt",
        contentType: "text/plain",
        base64: "aGVsbG8="
    }])

    // crypto
    const crypto = $native("crypto")
    // hash
    crypto.createHash("md5").sum("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // "e4d7f1b4ed2e42d15898f4b27b019da4"
    // hmac
    crypto.createHmac("sha1").sum("hello, world", "123456").map(c => c.toString(16).padStart(2, "0")).join("") // "9a231f1dd39a4ff6ea778a5640d1498794f8a9f8"
    // rsa
    const rsa = crypto.createRsa(),
        { privateKey, publicKey } = rsa.generateKey();
    String.fromCharCode(...
        rsa.decrypt(
            rsa.encrypt("hello, world", publicKey),
            privateKey
        )
    ) // "hello, world"
    rsa.verifyPss(
        "hello, world",
        rsa.signPss("hello, world", privateKey, "sha256"),
        publicKey,
        "sha256"
    ) // true

    // file
    const file = $native("file")
    file.write("greeting.txt", "hello, world")
    String.fromCharCode(...file.read("greeting.txt")) // "hello, world"

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
    const image = $native("image")
    const img0 = image.new(100, 200), // create a picture with width 100 and height 200
        img1 = image.parse($native("http")().request("GET", "https://www.baidu.com/img/flexible/logo/plus_logo_web_2.png").data.toBytes()) // read a picture from network
    image.toBytes(img0) // convert this picture to a byte array

    // template
    const content = $native("template")("greeting", { // read template greeting.tpl and render with input
        name: "this is name"
    })
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
                });
                router.beforeEach((to, from, next) => {
                    if (to.matched.length) { // 当前路由已匹配上
                        next(); // 直接渲染当前路由
                        return;
                    }
                    router.addRoute({ // 动态添加路由
                        path: to.path,
                        component: httpVueLoader(`../resource${to.path === "/" ? "/index" : to.path}.vue`) // 远程加载组件
                    });
                    next(to.path); // 重新进入 beforeEach 方法
                });
                new Vue({ router }).$mount("#container");
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
                        name: "world"
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
                title: "this is title"
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
            new Vue({ el: "#app" });
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

            const client = $native("file"),
                size = client.stat(name).size()

            const range = ctx.getHeader().Range
            if (!range) {
                return new ServiceResponse(200, {
                    "Accept-Ranges": "bytes",
                    "Content-Length": size + "",
                    "Content-Type": "video/mp4"
                })
            }

            const ranges = range.substring(6).split("-"),
                slice = 1024 * 1024 * 2, // The slice size is 2 MB
                start = Number(ranges[0]),
                end = Math.min(Number(ranges[1]) || (start + slice - 1), size - 1)
            
            const buf = client.readRange(name, start, end - start + 1) // slice the mp4 file from [start, end + 1)

            return new ServiceResponse(206, {
                "Content-Range": `bytes ${start}-${end}/${size}`,
                "Content-Length": end - start + 1 + "",
                "Content-Type": "video/mp4"
            }, buf)
        }
        ```
    2. You can preview at `http://127.0.0.1:8090/service/foo` in browser.

- Play a video online using HTTP-FLV.
    1. Create a flv file under `files/` using ffmpeg:
        ```bash
        ffmpeg \
            -i a.mp4 \
            -vcodec libx264 -r 25 -b 800000 \ # We need encode with libx264. Otherwise, using flv.js to pull the stream may cause an error: "DemuxException: type = CodecUnsupported, info = Flv: Unsupported codec in video frame: 2"
            -acodec aac -ac 2 -ar 44100 -ab 128k \
            -af "loudnorm" \
            a.flv
        ```
    2. Create a controller with url `/service/foo`. 
        ```typescript
        export default function (ctx: ServiceContext) {
            const buf = $native("file").read("a.flv")

            // send a chunk: flv header, total 9 + 4 bytes
            ctx.write(new Uint8Array(buf.slice(0, 9 + 4)))
            ctx.flush()

            let i = 9 + 4;
            while (i < buf.length) {
                const dataSize = (buf[i + 1] << 16) + (buf[i + 2] << 8) + buf[i + 3],
                    tagSize = 11 + dataSize,
                    previousTagSize = (buf[i + tagSize] << 24) + (buf[i + tagSize + 1] << 16) + (buf[i + tagSize + 2] << 8) + buf[i + tagSize + 3];
                if (tagSize != previousTagSize) {
                    throw new Error("Invalid previous tag size: " + tagSize + ", expected: " + previousTagSize);
                }

                // send a chunk: flv tag, total 11 + dataSize + 4 bytes
                ctx.write(new Uint8Array(buf.slice(i, i + tagSize + 4)))
                ctx.flush()

                i = i + tagSize + 4;
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
                    enableWorker: true // https://github.com/bilibili/flv.js/issues/322
                });
                flvPlayer.attachMediaElement(document.getElementById("videoElement"));
                flvPlayer.load();
                flvPlayer.play();
            }
        </script>
        ```
    4. You can preview at `http://127.0.0.1:8090/resource/foo.html`.

- Create a smtpd using socket module.
    1. Create a daemon.
        ```typescript
        export default function (ctx: ServiceContext) {
            const listener = $native("socket").listen("tcp", 25)
            while(true) {
                const conn = listener.accept()
                console.debug(onParseEmail(onReadData(conn)))
            }
        }

        function onReadData(conn) {
            conn.write("220 My Mail Sever\n")

            let data = "",
                s = String.fromCharCode(...conn.readLine())
            while (s.length) {
                switch (s.substring(0, 4).replace(/[\r\n]*$/, "")) {
                    case "HELO":
                    case "EHLO":
                        conn.write("250 OK\n")
                        break
                    case "MAIL":
                        conn.write("250 OK\n")
                        break
                    case "RCPT":
                        conn.write("250 OK\n")
                        break
                    case "DATA":
                        conn.write("354 OK\n")
                        break
                    case ".":
                        conn.write("250 OK\n")
                        break
                    case "QUIT":
                        conn.write("221 Bye\n")
                        conn.close()
                        return data
                    default:
                        data += s
                        break
                }
                s = String.fromCharCode(...conn.readLine())
            }
            return null
        }

        function onParseEmail(data) {
            data = (data || "").replace(/\r\n/g, "\n")
            return {
                subject: data.match(/^Subject: (.*)$/m)?.[1],
                from: data.match(/^From: .*(<.*>)$/m)?.[1],
                to: data.match(/^To: .*(<.*>)$/m)?.[1],
                body: String.fromCharCode(...$native("base64").decode(data.match(/\n\n(.*)/m)?.[1] || ""))
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
