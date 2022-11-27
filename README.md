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
    make run
    ```

4. Open `http://127.0.0.1:8090/` in browser and code what you want.

## Shortcut key of Editor Online

- Save as a script: `Ctrl` + `S`

- Delete this script: `Ctrl` + `0`

## Example of script

- A example of a custom module.
    ```typescript
    // http://127.0.0.1:8090/?name=user
    export const user = {
        name: "zhangsan"
    };
    ```
    ```typescript
    // http://127.0.0.1:8090/?name=foo
    import { user } from "./user";

    export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
        return `hello, ${user?.name ?? "world"}`;
    };
    ```

- A example of a custom module that extends Date.
    ```typescript
    // http://127.0.0.1:8090/?name=node_modules/date
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
    // http://127.0.0.1:8090/?name=foo
    import "date";

    export default function (req) {
        return new Date().toString("yyyy-MM-dd HH:mm:ss.S");
    };
    ```

- A simple example of return a custom response.
    ```typescript
    export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
        // return new Uint8Array([104, 101, 108, 108, 111]);
        return new ServiceResponse(500, {}, new Uint8Array([104, 101, 108, 108, 111]));
    }
    ```

- A example of return a view with asynchronous vues.
    1. Create a template with name `index.tpl`.
        ```html
        <!-- http://127.0.0.1:8090/?name=index.tpl -->
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
                        component: httpVueLoader(`view${to.path === "/" ? "/index" : to.path}.vue`) // 远程加载组件
                    });
                    next(to.path); // 重新进入 beforeEach 方法
                });
                new Vue({ router }).$mount("#container");
            </script>
        </body>
        </html>
        ```
    2. Create a vue SFC resource with name `greeting.vue`.
        ```html
        <!-- http://127.0.0.1:8090/?name=greeting.vue -->
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
    3. Create a typescript service with name `index` or `index.ts`.
        ```typescript
        // http://127.0.0.1:8090/?name=index
        export default function (req: ServiceRequest): ServiceResponse | Uint8Array | any {
            return $native("template")("index", {
                title: "this is title"
            });
        }
        ```
    4. You can preview at http://127.0.0.1:8090/service/index#/greeting

- A simple example of websocket server.
    ```typescript
    export default function (req: ServiceRequest) {
        const ws = req.upgradeToWebSocket() // upgrade http and get a websocket
        console.info(ws.read()) // read a message
        ws.send("hello, world") // write a message
        ws.close() // close the connection
    }
    ```

- A simple example of using console.
    ```typescript
    // ...
    console.error("this is a error message")
    ```

- A simple example of using error.
    ```typescript
    // ...
    throw new Error("error message")

    // ...
    throw {
        code: "error code",
        message: "error message"
    }
    ```

- Examples of using native module.
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

    // email
    $native("email")("smtp.163.com", 465, username, password).send(["zhangsan@abc.com"], "greeting", "hello, world")

    // crypto
    const crypto = $native("crypto")
    crypto.md5("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // e4d7f1b4ed2e42d15898f4b27b019da4
    crypto.sha256("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // 09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b

    // http
    const { status, header, data } = $native("http")({}).request("GET", "https://www.baidu.com")
    status // 200
    header // {"Content-Length":["227"],"Content-Type":["text/html"]...]}
    data.toString() // "<html>..."

    // image
    const image = $native("image")
    const
        img0 = image.new(100, 200), // create a picture with width 100 and height 200
        img1 = image.parse(request("GET", "https://www.baidu.com/img/flexible/logo/plus_logo_web_2.png").data) // read a picture from network
    image.toBytes(img0) // convert this picture to a byte array

    // template
    const content = $native("template")("greeting", { // read template greeting.tpl and render with input
        name: "this is name"
    })
    ```

<!--

## Other

- Register a module loader
    ```go
    registry := require.NewRegistryWithLoader(func(path string) ([]byte, error) { // 创建自定义 require loader（脚本每次修改后，registry 需要重新生成，防止 module 被缓存，从而导致 module 修改后不生效）
        // 从数据库中查找模块
        rows, err := Db.Query("select jscontent from script where name = ?", path)
        if err != nil {
            panic(err.Error())
            return nil, err
        }
        defer rows.Close()
        if rows.Next() == false {
            // 读取 node_modules 目录下模块或文件
            if strings.HasPrefix(path, "node_modules/") {
                return require.DefaultSourceLoader(path)
            }
            return nil, errors.New("The module was not found: " + path)
        }
        script := Script{}
        err = rows.Scan(&script.JsContent)
        return []byte(script.JsContent), err
    })
    ```

- Commands of example service using curl
    ```bash
    curl -XPOST http://127.0.0.1:8090/service/greeting -H 'Content-Type: application/x-www-form-urlencoded' -d 'name=zhangsan&age=26&name=lisi'

    curl -XPOST http://127.0.0.1:8090/service/greeting -H 'Content-Type: application/json' -d '{"name":"zhangsan","age":26}'
    ```

- Commands of source using curl
    ```bash
    curl -XDELETE "http://127.0.0.1:8090/source?name=greeting&type=tpl"
    ```

-->
