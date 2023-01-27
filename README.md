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

- Save as a script: `Ctrl` + `S`

- Delete this script: `Ctrl` + `0`

## Examples

### Controller

You can create a controller as a http/https service.

- A simple controller.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo
    export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
        console.info("The body of request is:", String.fromCharCode(...req.getBody())) // print http request body
        return `hello, world`
    }
    ```

- Return a custom response.
    ```typescript
    export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
        // return new Uint8Array([104, 101, 108, 108, 111]) // response with body "hello"
        return new ServiceResponse(500, {
            "Content-Type": "text/plain"
        }, new Uint8Array([104, 101, 108, 108, 111]))
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

    export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
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

    export default function (req) {
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
    crypto.md5("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // e4d7f1b4ed2e42d15898f4b27b019da4
    crypto.sha256("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // 09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b

    // file
    const file = $native("file")
    file.write("greeting.txt", "hello, world")
    String.fromCharCode(...file.read("greeting.txt")) // "hello, world"

    // http
    const { status, header, data } = $native("http")({
        //caCert: "", // ca certificates for http client
        //cert: "", key: "", // private key and certificate/public key for http client auth
        //insecureSkipVerify: true // disable verify server certificate
    }).request("GET", "https://www.baidu.com")
    status // 200
    header // {"Content-Length":["227"],"Content-Type":["text/html"]...]}
    data.toString() // "<html>..."

    // image
    const image = $native("image")
    const img0 = image.new(100, 200), // create a picture with width 100 and height 200
        img1 = image.parse(request("GET", "https://www.baidu.com/img/flexible/logo/plus_logo_web_2.png").data) // read a picture from network
    image.toBytes(img0) // convert this picture to a byte array

    // template
    const content = $native("template")("greeting", { // read template greeting.tpl and render with input
        name: "this is name"
    })
    ```

### Advance

- Return a view with asynchronous vues.
    1. Create a template with name `index`, type `template` and lang `html`.
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
    2. Create a vue SFC resource with name `greeting`, type `resource`, lang `vue` and url `/resource/greeting.vue`.
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
    3. Create a controller with name `index`, type `controller` and url `/service/`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=index
        export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
            return $native("template")("index", {
                title: "this is title"
            })
        }
        ```
    4. You can preview at `http://127.0.0.1:8090/service/#/greeting`

- Websocket server.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=foo
    export default function (req: ServiceRequest) {
        const ws = req.upgradeToWebSocket() // upgrade http and get a websocket
        console.info(ws.read()) // read a message
        ws.send("hello, world") // write a message
        ws.close() // close the connection
    }
    ```

- Upload file.
    1. Create a resource with name `foo`, type `resource`, lang `html` and url `/resource/foo.html`.
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
    2. Create a controller with name `foo`, type `controller` and url `/service/foo`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=foo
        export default function (req: ServiceRequest) {
            const file = req.getFile("file"),
                hash = $native("crypto").md5(file.data).map(c => c.toString(16).padStart(2, "0")).join("")
            console.info(hash)
        }
        ```
    3. You can preview at `http://127.0.0.1:8090/resource/foo.html`. You can also run it using curl:
        ```bash
        # Upload a file
        curl -F "file=@./abc.txt; filename=abc.txt;" http://127.0.0.1:8090/service/foo
        ```

- Download a mp4 with http range.
    1. Create a controller with name `foo`, type `controller` and url `/service/foo`.
        ```typescript
        // http://127.0.0.1:8090/editor.html?name=foo
        export default function (req: ServiceRequest) {
            const buf = $native("file").read("a.mp4")

            const range = req.getHeader().Range
            if (!range) {
                return new ServiceResponse(200, {
                    "Accept-Ranges": "bytes",
                    "Content-Length": buf.length + "",
                    "Content-Type": "video/mp4"
                })
            }

            const ranges = range.substring(6).split("-"),
                slice = 1024 * 1024 * 2, // The slice size is 2 MB
                start = Number(ranges[0]),
                end = Math.min(Number(ranges[1]) || (start + slice - 1), buf.length - 1)
            return new ServiceResponse(206, {
                "Content-Range": `bytes ${start}-${end}/${buf.length}`,
                "Content-Length": end - start + 1 + "",
                "Content-Type": "video/mp4"
            }, new Uint8Array(buf.slice(start, end + 1))) // slice the mp4 file from [start, end) as a Uint8Array
        }
        ```
    2. You can preview at `http://127.0.0.1:8090/service/foo` in browser.
