# Cube

## 启动

```bash
make run
```

## API

### 删除脚本

```bash
curl -XDELETE http://127.0.0.1:8090/script?name=node_modules/db
```
<!--
## 开发

### 注册模块加载器

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
-->
## 脚本示例

### node_modules 模块

1. 新建模块脚本如 `node_modules/users`
    ```typescript
    export const user = {
        name: "zhangsan"
    };
    ```
2. 调用模块
    ```typescript
    import { user } from "users";
    
    export default function (req) {
        return `hello, ${user?.name ?? "world";
    };
    ```

### 自定义模块

1. 新建模块脚本如 `users`
    ```typescript
    export const user = {
        name: "zhangsan"
    };
    ```
2. 调用模块
    ```typescript
    import { user } from "./users";
    
    export default function (req) {
        return `hello, ${user?.name ?? "world";
    };
    ```

### 自定义拓展方法模块

1. 新建模块脚本如 `node_modules/date`
    ```typescript
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
2. 调用模块
    ```typescript
    import "date";

    export default function (req) {
        return new Date().toString("yyyy-MM-dd HH:mm:ss.S");
    };
    ```

### 调用 Native 方法

#### 接口响应

```typescript
export default function (req: HttpRequest, res: HttpResponse) {
    res.setData(new Uint8Array([104, 101, 108, 108, 111])) // 自定义写入值，将会覆盖后文中的 return 值
    return // 这里如果 return 任意值都是无效的，实际返回值以 res.setData 为准
}
```

#### 控制台打印

```typescript
// console
console.error("hello, world")
```

#### 异常

```typescript
// error
throw new Error("hello, world")
throw {
    code: "error code", // 指定错误码
    message: "error message"
}
```

#### 其它

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

// email
$native("email")("smtp.163.com", 465, username, password).send(["zhangsan@abc.com"], "greeting", "hello, world")

// crypto
const crypto = $native("crypto")
crypto.md5("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // e4d7f1b4ed2e42d15898f4b27b019da4
crypto.sha256("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // 09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b

// http
$native("http").request("GET", "https://www.baidu.com") // {"data":"PGh0bWw...","headers":{"Content-Length":["227"],"Content-Type":["text/html"]...]},"status":200}}

// image
const image = $native("image")
const
    img0 = image.new(100, 200), // 创建一张空图片，宽 100，高 200
    img1 = image.parse(request("GET", "https://www.baidu.com/img/flexible/logo/plus_logo_web_2.png").data) // 读取一张图片
image.toBytes(img0) // 将图片转换成二进制字节数组
```
<!-- 
## Curl

```bash
curl -XPOST http://127.0.0.1:8090/service/greeting -H 'Content-Type: application/x-www-form-urlencoded' -d 'name=zhangsan&age=26&name=lisi'

curl -XPOST http://127.0.0.1:8090/service/greeting -H 'Content-Type: application/json' -d '{"name":"zhangsan","age":26}'
``` -->
