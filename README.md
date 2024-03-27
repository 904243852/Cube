# Cube

A simple web server that can be developed online using typescript/javascript.

## Getting started

1. Clone the repository.

2. Build from the source code.
    ```bash
    make build
    ```

3. Start the server.
    ```bash
    ./cube -n 8
    ```
    Or you can start directly from the source code:
    ```bash
    make run
    ```
    For more startup parameters, please refer to:
    ```bash
    ./cube --help
    ```

4. Open `http://127.0.0.1:8090/` in browser.

### Run with SSL/TLS

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

### Run with HTTP/3

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

- [A custom module extends Date.](docs/modules/date.md)

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
    cryptoc.createHmac("sha1").sum("hello, world", "123456").toString("hex") // "9a231f1dd39a4ff6ea778a5640d1498794f8a9f8"
    // rsa
    // privateKey and publicKey mentioned is PKCS#1 format
    const rsa = cryptoc.createRsa(),
        { privateKey, publicKey } = rsa.generateKey();
    rsa.decrypt(
        rsa.encrypt("hello, world", publicKey),
        privateKey,
    ).toString() // "hello, world"
    rsa.verify(
        "hello, world",
        rsa.sign("hello, world", privateKey, "sha256", "pss"),
        publicKey,
        "sha256",
        "pss",
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
    header // { "Content-Length": "227", "Content-Type": "text/html", ... }
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
                hash = $native("crypto").md5(file.data).toString("hex")
            console.info(hash)
        }
        ```
    3. You can preview at `http://127.0.0.1:8090/resource/foo.html`. You can also run it using curl:
        ```bash
        # Upload a file
        curl -F "file=@./abc.txt; filename=abc.txt;" http://127.0.0.1:8090/service/foo
        ```

### For more examples, please refer to the [document](docs/summary.md)
