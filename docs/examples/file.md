# File reader, such as jpg/png(resize), mp4(HTTP-Range), zip

1. Create a controller.
    ```typescript
    //?name=FileReader&type=controller&method=GET&url=file/{name}
    const filec = $native("file"),
        imagec = $native("image"),
        zipc = $native("zip")

    export default (app => app.run.bind(app))(new class FileReader {
        private static readonly FileTypes = {
            "FFD8FF": "jpeg", "89504E47": "png", "47494638": "gif",
            "504B0304": "zip",
            "0000????66747970": "mp4",
        }

        private static readonly HttpRangeSliceSize = 1024 * 1024

        public run(ctx: ServiceContext) {
            return this.read(ctx.getPathVariables().name.split("!/"), ctx, undefined)
        }

        private read([name, ...subnames]: string[], ctx: ServiceContext, zipFiles: ZipEntry[]) {
            if (name === "" || name[name.length - 1] === "/") {
                return this.toDir(name, zipFiles)
            }
    
            // 根据文件的前 8 个字节来判断文件的类型
            const headers = this.getHeaderBytes(name, 8, zipFiles),
                magic = headers.map(i => i.toString(16).padStart(2, '0')).join("").toUpperCase(),
                fileType = FileReader.FileTypes[
                    Object.keys(FileReader.FileTypes)
                        .filter(n => new RegExp("^" + n.padEnd(headers.length * 2, '?').replace(/\?+/g, s => `[A-F0-9]{${s.length}}`) + "$").test(magic))
                        .pop()
                ]

            if (!!~["jpeg", "png"].indexOf(fileType)) {
                return this.toImage(name, ctx, zipFiles) // 读取并缩放图片
            }

            if (fileType === "zip" && subnames.length) {
                return this.read(subnames, ctx, zipc.read(filec.read(name)).getEntries()) // 读取 zip 子文件
            }

            if (!subnames.length) { // zip 的子文件无需范围请求
                const fileSize = filec.stat(name).size()
                if (fileSize > FileReader.HttpRangeSliceSize) { // 如果文件不大，也无需范围请求
                    return this.toFileWithHttpRange(name, ctx, fileSize, { mp4: "video/mp4" }[fileType]) // 范围请求
                }
            }

            return this.toFile(name, zipFiles)
        }

        private getHeaderBytes(name: string, size: number, zipEntries?: ZipEntry[]) {
            if (zipEntries === undefined) {
                return filec.readRange(name, 0, size)
            }
            return zipEntries.filter(i => i.name === name).pop()?.getData()?.slice(0, size)
        }

        private toDir(name: string, zipFiles: ZipEntry[]) {
            if (zipFiles === undefined) {
                return filec.list(name)
            }
            return zipFiles.map(i => i.name)
        }

        private toImage(name: string, ctx: ServiceContext, zipFiles: ZipEntry[]) {
            const width = Number(ctx.getURL().params.width?.pop() || 1280)
            return imagec.parse(zipFiles === undefined ? filec.read(name) : zipFiles.filter(i => i.name === name).pop()?.getData())
                .resize(width)
                .toJPG()
        }

        private toFile(name: string, zipFiles: ZipEntry[]) {
            return zipFiles === undefined ? filec.read(name) : zipFiles.filter(i => i.name === name).pop()?.getData()
        }

        private toFileWithHttpRange(name: string, ctx: ServiceContext, fileSize: number, contentType: string) {
            const range = ctx.getHeader().Range
            if (!range) {
                return new ServiceResponse(200, {
                    "Accept-Ranges": "bytes",
                    "Content-Length": fileSize + "",
                    ...(contentType && {
                        "Content-Type": contentType,
                    }),
                })
            }

            const ranges = range.substring(6).split("-"),
                start = Number(ranges[0]),
                end = Math.min(Number(ranges[1]) || (start + FileReader.HttpRangeSliceSize - 1), fileSize - 1)

            return new ServiceResponse(206, {
                "Content-Range": `bytes ${start}-${end}/${fileSize}`,
                "Content-Length": end - start + 1 + "",
                ...(contentType && {
                    "Content-Type": contentType,
                }),
            }, filec.readRange(name, start, end - start + 1))
        }
    })
    ```

2. Create a html resource.
    ```html
    //?name=FileReader4Image&type=resource&lang=html&url=image
    <!DOCTYPE html>
    <html>
    
    <head>
        <meta charset="UTF-8">
        <title></title>
        <style>
            body {
                margin: 0;
            }
            #app {
                position: relative;
                width: 100%;
            }
            .item {
                position: absolute;
                text-align: center;
            }
        </style>
    </head>
    
    <body>
        <div id="app"></div>
        <script>
            ({
                path: window.location.hash.substring(1),
                files: [], // 图片队列
                options: null, // 配置：边框大小，图片最大宽度
                columns: null, // 每列图片的高度
                observer: null, // 观察器
                render: function() {
                    const that = this;
                    const name = that.files.pop();
                    if (!name) {
                        return;
                    }
                    const e = document.createElement("img");
                    e.src = `/service/file/${that.path}/${name}?width=${that.options.width}`;
                    e.setAttribute("class", "item");
                    e.onload = function() {
                        const minHeight = Math.min(...that.columns), // 找到对应列最小高度的值
                            minHeightIndex = that.columns.indexOf(minHeight); // 找到对应列最小高度的下标
                        e.style.transform = `translate(${minHeightIndex * 100}%, ${minHeight}px)`; // 根据下标进行变换，变换宽度为偏移多少个下标，上下为该下标所有高度
                        document.getElementById("app").appendChild(e);
                        that.columns[minHeightIndex] += e.offsetHeight; // 对应下标增加高度
                        that.observer.observe(e); // 观察该元素用作懒加载
                    };
                    e.onclick = function() {
                        window.open(`/service/file/${that.path}/${name}`);
                    };
                },
                mount: function(options) {
                    const that = this;
                    // 保存配置
                    that.options = {
                        width: 140, // 默认每张图片最大宽度 140 px
                        ...options,
                    };
                    // 创建列缓存，每行展示 ${columns.length} 张图片
                    that.columns = [...new Array(Math.floor(window.innerWidth / that.options.width))].map(_ => 0);
                    // 创建观察器
                    that.observer = new IntersectionObserver(function(entries) {
                        const entry = entries[0];
                        if(entry.isIntersecting) { // 如果已进入视图，停止监听，并且生成新的元素
                            this.unobserve(entry.target); // 这里用 this 以指向观察器自己
                            that.render();
                        }
                    });
                    fetch(`/service/file/${that.path}`).then(r => r.json()).then(r => {
                        that.files = r.data.filter(i => /\.(jpe?g|png)$/.test(i)).reverse();
                        that.render();
                    });
                }
            }).mount({
                width: Math.floor(window.innerWidth / 2), // 每行展示 2 张图片
            });
        </script>
    </body>
    
    </html>
    ```

3. You can preview at [`/service/file/`](/service/file/) and [`/resource/image#/`](/resource/image#/) in browser.
    ```
    # read a.jpg with resized(width = 720)
    /service/file/a.jpg?width=720

    # read a.mp4 (size > 1 mb) with http range
    /service/file/a.mp4

    # read b.jpg in a.zip
    /service/file/a.zip!/b.jpg

    # read all images in a.zip
    /resource/image#a.zip!/
    ```
