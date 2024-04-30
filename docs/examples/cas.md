# Create a Single Sign On server based on CAS protocol

1. Create a template name `CASLogin`.
    ```html
    //?name=CASLogin&type=template&lang=html
    <!DOCTYPE html>
    <html>

    <head>
        <meta charset="utf-8">
        <title>Sign in</title>
        <style>
            body {
                padding: 0;
                margin: 0;
                font-family: SF Pro Display, Roboto, Noto, Arial, PingFang SC, Hiragino Sans GB, Microsoft YaHei, sans-serif;
                font-size: 14px;
            }
            .card {
                position: absolute;
                padding: 40px;
                width: 300px;
                top: 5.5em;
                right: 8em;
                background-color: #fff;
                box-shadow: 0 8px 16px 0 rgba(0, 0, 0, .1);
            }
            .card .title {
                text-align: center;
                font-size: 18px;
            }
            .card>form, div {
                margin-bottom: 20px;
            }
            .card .error {
                color: #f66f6a;
                margin-bottom: 10px;
            }
            .card .feild {
                border-bottom: 1px solid #DFE1E6 !important;
            }
            .card .feild:hover {
                border-color: #333 !important;
            }
            .card input:not([type=checkbox]) {
                height: 32px;
                border: none;
                outline: none;
                padding: 2px 8px;
                width: 100%;
                -webkit-box-sizing: border-box;
            }
            .card input[type=submit] {
                background-color: #F66F6A;
                border-radius: 2px;
                color: #fff;
                cursor: default;
            }
            .card .others {
                text-align: center;
                position: relative;
            }
            .card .others:before {
                content: '';
                border-top: 1px solid #ddd;
                width: 100%;
                position: absolute;
                top: 50%;
                display: block;
            }
            .card .others span {
                padding: 0 7px;
                background-color: #fff;
                position: relative;
            }
            .card .providers {
                text-align: center;
                margin-bottom: 0;
            }
            .card a {
                margin: 0 10px;
                color: #08c;
                cursor: pointer;
            }
        </style>
    </head>

    <body>
        <div class="card">
            <div class="title">Sign in</div>
            <form autocomplete="off" action="" method="post">
                {{ if .error }}
                <div class="error">{{ .error }}</div>
                {{ end }}
                <div class="feild">
                    <input type="text" name="username" placeholder="Username" />
                </div>
                <div class="feild">
                    <input type="password" name="password" placeholder="Password" />
                </div>
                <input type="submit" value="Submit" />
            </form>
            {{ if gt (len .delegates) 0 }}
            <div class="others">
                <span>Others</span>
            </div>
            <div class="providers">
                {{ range $key, $value := .delegates }}
                <a href="/service/cas/clientredirect?client_name={{ $key }}&service={{ $.service }}">{{ $key }}</a>
                {{ end }}
            </div>
            {{ end }}
        </div>
    </body>

    </html>
    ```

2. Create a controller with url `/service/cas/{path}` and method `Any`.
    ```typescript
    //?name=CASServer&type=controller&url=cas/{path}
    const MyHost = "http://127.0.0.1:8090/service/cas/"

    const Timeout = {
        ST: 300_000,
        TGT: 7200_000,
    }

    const services: string[] = [
        "http://127.0.0.1/",
    ]

    const delegates: { [name: string]: { login: string; validator: (ticket: string) => Authentication; }; } = {
        s3000: {
            login: "http://127.0.0.1:3000/login",
            validator(ticket) {
                const resp = $native("http")().request("post",
                    "http://127.0.0.1:3000/validate",
                    {
                        "Content-Type": "application/json",
                    },
                    JSON.stringify({
                        ticket,
                    }),
                )
                const { user } = resp.data.toJson()
                return {
                    user: user.name,
                    attributes: {
                        token: resp.header["access-token"],
                        userId: user.id,
                        userName: user.name,
                    },
                }
            },
        }
    }

    const cache = $native("cache")

    export default (app => app.run.bind(app))(new class {
        public run(ctx: ServiceContext) {
            const { path, params } = ctx.getURL(),
                input: Input = {
                    service: params.service?.[0],
                    ticket: params.ticket?.[0],
                    client: params.client_name?.[0],
                    delegatedclientid: params.delegatedclientid?.[0],
                    tgc: ctx.getCookie("TGC")?.value,
                }
            switch (path.substring(13)) {
                case "login":
                    return this.login(input, ctx.getMethod() === "POST" && ctx.getForm())
                case "clientredirect":
                    return this.clientredirect(input)
                case "validate":
                    return this.validate(input)
                case "logout":
                    return this.logout()
                default:
                    return this.toServiceResponse("error", 404, "not found")
            }
        }

        /**
         * service login
         * 
         * @param input input
         * @param forms forms user submitted, such as username, password, captcha
         */
        public login(input: Input, forms: { [name: string]: string[]; }) {
            if (!~services.indexOf(input.service)) {
                return this.toServiceResponse("error", 500, "service not allowed: " + input.service)
            }

            const response = new ServiceResponse(302, null)

            if (input.delegatedclientid) { // 第三方认证
                const d = cache.get(input.delegatedclientid)
                if (!d) {
                    return this.toServiceResponse("error", 500, "client invalid")
                }
                if (d.service != input.service) {
                    return this.toServiceResponse("error", 500, "service invalid")
                }
                const delegate = delegates[d.client]
                if (!delegate) {
                    return this.toServiceResponse("error", 500, "client not allowed")
                }
                if (!input.ticket) {
                    return this.toServiceResponse("error", 500, "ticket required")
                }
                const a = delegate.validator(input.ticket)
                if (!a) {
                    return this.toServiceResponse("error", 500, "delegate validate failed")
                }
                input.tgc = this.uid("TGT")
                cache.set(input.tgc, a, Timeout.TGT)
                response.setCookie("TGC", input.tgc)
            } else if (forms) { // 用户名、密码认证
                const a = this.authenticate(forms)
                if (!a) {
                    return this.toServiceResponse("login", input.service, "incorrect username or password")
                }
                input.tgc = this.uid("TGT")
                cache.set(input.tgc, a, Timeout.TGT)
                response.setCookie("TGC", input.tgc)
            } else { // cookie 认证
                const a = cache.get(input.tgc)
                if (!a) {
                    return this.toServiceResponse("login", input.service)
                }
            }

            // 签发 service ticket
            const st = this.uid("ST")
            cache.set(st, {
                service: input.service,
                tgc: input.tgc,
            }, Timeout.ST)
            response.setHeader("Location", input.service + "?ticket=" + st)
            return response
        }

        /**
         * client redirect
         */
        public clientredirect(input: Input) {
            const delegate = delegates[input.client]
            if (!delegate) {
                return this.toServiceResponse("error", 500, "client not allowed")
            }

            const id = this.uid("TST")
            cache.set(id, {
                service: input.service,
                client: input.client,
            }, Timeout.TGT)
            return new ServiceResponse(302, {
                Location: `${delegate.login}?client_name=${input.client}&service=${encodeURIComponent(`${MyHost}login?service=${encodeURIComponent(input.service)}&delegatedclientid=${id}`)}`,
            })
        }

        /**
         * service validate
         */
        public validate(input: Input) {
            const e = cache.get(input.ticket)
            if (!e) {
                return this.toServiceResponse("xml", "INVALID_TICKET", "invalid ticket")
            }
            if (e.service !== input.service) {
                return this.toServiceResponse("xml", "INVALID_SERVICE", "invalid service")
            }
            return this.toServiceResponse("xml", cache.get(e.tgc))
        }

        /**
         * service logout
         */
        public logout() {
            const resp = new ServiceResponse(200, null)
            resp.setCookie("TGC", "")
            return resp
        }

        protected authenticate(forms: { [name: string]: string[]; }): Authentication {
            // here we just use the mock user for example
            if (forms.username?.[0] === "zhangsan") {
                return {
                    user: "zhangsan",
                    attributes: {
                        grender: true,
                    },
                }
            }
            return null
        }

        private toServiceResponse(template: "login", service: string, error?: string);
        private toServiceResponse(template: "error", status: number, error: string);
        private toServiceResponse(template: "xml", auth: Authentication);
        private toServiceResponse(template: "xml", code: string, message: string);
        private toServiceResponse(template: string, ...params: any[]) {
            if (template === "login") {
                const [service, error] = params as [string, string]
                return $native("template")("CASLogin", {
                    service: encodeURIComponent(service),
                    error,
                    delegates,
                })
            }
            if (template === "error") {
                const [status, error] = params as [number, string]
                return new ServiceResponse(status, null, error)
            }
            if (template === "xml") {
                if (typeof params[0] === "object") {
                    const [a] = params as [Authentication]
                    return [
                        "<cas:serviceResponse xmlns:cas='http://www.yale.edu/tp/cas'>",
                        "    <cas:authenticationSuccess>",
                        `        <cas:user>${a.user}</cas:user>`,
                        "        <cas:attributes>",
                        ...Object.keys(a.attributes).map(name => `<cas:${name}>${a.attributes[name]}</cas:${name}>`),
                        "        </cas:attributes>",
                        "    </cas:authenticationSuccess>",
                        "</cas:serviceResponse>",
                    ].map(i => i.trim()).join("")
                }
                if (typeof params[0] === "string") {
                    const [code, message] = params as [string, string]
                    return [
                        "<cas:serviceResponse xmlns:cas='http://www.yale.edu/tp/cas'>",
                        `    <cas:authenticationFailure code="${code}">${message}</cas:authenticationFailure>`,
                        "</cas:serviceResponse>",
                    ].map(i => i.trim()).join("")
                }
            }
            throw new Error("not implemented")
        }

        private uid(type: string): string {
            return type + "-" + [...new Array(16)].map(_ => {
                return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"[Math.random() * 62 | 0]
            }).join("")
        }
    })

    type Authentication = {
        user: string;
        attributes?: {
            [name: string]: string | boolean | number;
        };
    }

    type Input = {
        /** service */
        service: string;
        /** service ticket */
        ticket?: string;
        /** client name */
        client?: string;
        /** ticket-granting cookie, id of ticket-granting ticket */
        tgc?: string;
        /** delegated client id */
        delegatedclientid?: string;
    }
    ```

3. Test login with url [`/service/cas/login?service=http://127.0.0.1/`](/service/cas/login?service=http://127.0.0.1/).

4. Validate with url [`/service/cas/validate?service=http://127.0.0.1/&ticket=${your ticket}`](/service/cas/validate?service=http://127.0.0.1/&ticket=).
