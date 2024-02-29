# Server-Side Rendering

```typescript
//#name=node_modules/ssr&type=module
interface App {
    created?: () => object
    methods?: { [name: string]: Function }
}

export class ServerPage {
    private template: string

    private app: App

    constructor(template: string, app: App) {
        this.template = template
        this.app = app
    }

    public get() {
        return this.render(
            this.template,
            this.app?.created && this.app.created.apply(this) || {},
        )
    }

    public post(name: string, ...params: any[]) {
        if (name in this.app.methods) {
            return {
                output: this.app.methods[name].apply(this.app.methods, params),
            }
        }
        throw new Error("no such method")
    }

    private render(template: string, data: object): string {
        const pattern = /<%([\s\S]*?)%>/g // 匹配模板中 "<%...%>" 片段代码

        const codes = [`with(data) { let __tpl__ = "";`] // 需要动态执行的函数代码集合

        let match: RegExpExecArray = null, // 当前匹配的结果
            index = 0 // 当前匹配到的索引
        while (match = pattern.exec(template)) {
            // 保存当前匹配项之前的普通文本/占位
            codes.push(`__tpl__ += "${template.substring(index, match.index).replace(/\r?\n/g, "\\n").replace(/"/g, '\\"')}";`)

            // 保存当前匹配项
            if (match[1].substr(0, 1) == "=") { // 占位符
                codes.push(`__tpl__ += ${match[1].substr(1)};`)
            } else { // 逻辑代码
                codes.push(match[1])
            }

            // 更新当前匹配索引
            index = match.index + match[0].length
        }
        // 保存文本尾部
        codes.push(`__tpl__ += "${template.substring(index).replace(/\r?\n/g, "\\n").replace(/"/g, '\\"')}";`)
        codes.push("return __tpl__; }")

        return new Function("data", codes.join("\n"))(data) // 渲染模板中 "<%...%>" 片段代码
            .replace(/{{(.+?)}}/g, function ($0: string, $1: string) { // 渲染模板中 "{{ ... }}" 片段代码
                return data[$1.trim()] || $0
            })
            + `<script>
                (() => {
                    const rmi = function(name) {
                        return {
                            value: function (...params) {
                                return fetch(window.location.href.replace(/[#?].*/, ""), {
                                    method: "POST",
                                    body: JSON.stringify({
                                        name,
                                        params,
                                    }),
                                    credentials: "include",
                                }).then(r => r.json()).then(r => r.data.output)
                            }
                        }
                    };
                    ${Object.keys(this.app?.methods || {}).map(name => `Object.defineProperty(window, "${name}", rmi("${name}"));`).join(" ")}
                })()
            </script>`.replace(/\s*\n\s*/g, " ")
    }
}
```

### Usage

```typescript
import { ServerPage } from "ssr"

const p = new ServerPage(`<html>
<head>
	<title></title>
    <meta charset="UTF-8">
</head>
<body>
    <button onclick="greeting('world').then(i => alert(i))">hello, {{ name }}</button>
    <ul>
<% for(const user of users) {
    if(user.age < 18) { %>
        <li>我的名字是<%=user.name%>，我的年龄是<%=user.age%></li>
    <% } else { %>
        <li>my name is <%=user.name%>,my age is a sercet.</li>
    <% } %>
<% } %>
    </ul>
</body>
</html>`, {
    created() {
        return {
            name: "zhangsan",
            users: [{ name: "zhangsan", age: 12 }, { name: "lisi", age: 13 }, { name: "wangwu", age: 18 }],
        }
    },
    methods: {
        greeting(name) {
            return "hello, " + name
        },
    },
})

export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
    if (ctx.getMethod() === "GET") {
        return new ServiceResponse(
            200,
            {
                "Content-Type": "text/html"
            },
            p.get(),
        )
    }

    const { name, params, } = ctx.getJsonBody()
    return p.post(name, ...params)
}
```
