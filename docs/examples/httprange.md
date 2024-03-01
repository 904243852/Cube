# Download a mp4 using HTTP Range

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
