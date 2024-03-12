# Create a webdav server

1. Create a controller with url `/service/webdav/{path}` and method `Any`.
    ```typescript
    //?name=webdav&type=controller&url=webdav/{path}
    export default ctx => (app => app.run(ctx))(new class {
        private config = {
            username: "",
            password: "",
        }

        public run(ctx: ServiceContext) {
            if (this.config.password && ctx.getHeader().Authorization !== `Basic ${Buffer.from(this.config.username + ":" + this.config.password).toString("base64")}`) {
                return new ServiceResponse(401, null)
            }

            const { path } = ctx.getPathVariables()
            switch (<string>ctx.getMethod()) {
                case "PROPFIND":
                    return this.propfind(path)
                case "GET":
                    return this.get(path)
                case "PUT":
                    return this.put(path, ctx.getBody().toString())
                default:
                    throw new Error("not implemented")
            }
        }

        private propfind(path: string) {
            const resources = (path => {
                if (path === "") {
                    return $native("db").query("select type, last_modified_date from source group by type order by type").map(e => {
                        return {
                            type: "folder",
                            href: `/dav/root/${e.type}/`,
                            name: e.type,
                            date: new Date(e.last_modified_date.toString().substring(0, 19)).toUTCString(),
                            length: 0,
                        }
                    })
                }
                const type = path.substring(0, path.length - 1)
                return $native("db").query("select name, content, last_modified_date from source where type = ?", path.substring(0, path.length - 1)).map(e => {
                    return {
                        type: "file",
                        href: `/dav/root/${type}/${e.name.replace("/", "@")}`,
                        name: e.name,
                        date: new Date(e.last_modified_date.toString().substring(0, 19)).toUTCString(),
                        length: e.content.length,
                    }
                })
            })(path)
            return new ServiceResponse(207, {
                "Content-Type": "text/xml; charset=utf-8"
            }, `<?xml version="1.0" encoding="UTF-8"?>
                    <D:multistatus xmlns:D="DAV:">
                        <D:response>
                            <D:href>/dav/root/</D:href>
                            <D:propstat>
                                <D:prop>
                                    <D:displayname>root</D:displayname>
                                    <D:resourcetype>
                                        <D:collection xmlns:D="DAV:" />
                                    </D:resourcetype>
                                </D:prop>
                                <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                        </D:response>
                    ${resources.map(e => `
                        <D:response>
                            <D:href>${e.href}</D:href>
                            <D:propstat>
                                <D:prop>
                                    <D:displayname>${e.name}</D:displayname>
                                    <D:getlastmodified>${e.date}</D:getlastmodified>
                        ${e.type === "folder" ? `
                                    <D:resourcetype><D:collection/></D:resourcetype>
                        ` : `
                                    <D:resourcetype></D:resourcetype>
                                    <D:getcontentlength>${e.length}</D:getcontentlength>
                                    <D:getcontenttype>text/plain; charset=utf-8</D:getcontenttype>
                        `}
                                </D:prop>
                                <D:status>HTTP/1.1 200 OK</D:status>
                            </D:propstat>
                        </D:response>
                    `).join("")}
                    </D:multistatus>`.replace(/>[\s\n\r]*</g, "><")
            )
        }

        private get(path: string) {
            const [type, name] = path.split("/")
            return $native("db").query("select content from source where type = ? and name = ?", type, name.replace("@", "/")).pop()?.content || ""
        }

        private put(path: string, content: string) {
            const [type, name] = path.split("/")
            $native("db").exec("update source set content = ? where type = ? and name = ?", content, type, name.replace("@", "/"))
            return new ServiceResponse(201, {}, "Created")
        }
    })
    ```

2. Connect with url `http://127.0.0.1:8090/service/webdav`.
