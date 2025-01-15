# Play a video online using HTTP-FLV

1. Create a flv file under `files/` using ffmpeg:
    ```bash
    ffmpeg \
        -i a.mp4 \
        -vcodec libx264 -r 25 -b:v 800000 \
        -acodec aac -ac 2 -ar 44100 -ab 128k \
        -af "loudnorm" \
        -vf "scale=-1:720" \
        -y a.flv
    ```
    > We need encode with libx264. Otherwise, using flv.js to pull the stream may cause an error: "DemuxException: type = CodecUnsupported, info = Flv: Unsupported codec in video frame: 2"

2. Create a controller with url `/service/foo`.
    ```typescript
    export default function (ctx: ServiceContext) {
        const buf = $native("file").read("a.flv")

        // send a chunk: flv header(9 bytes) + previousTagSize0(4 bytes)
        ctx.write(new Uint8Array(buf.slice(0, 9 + 4)))
        ctx.flush()

        let i = 9 + 4
        while (i < buf.length) {
            const dataSize = (buf[i + 1] << 16) + (buf[i + 2] << 8) + buf[i + 3],
                tagSize = 11 + dataSize,
                previousTagSize = (buf[i + tagSize] << 24) + (buf[i + tagSize + 1] << 16) + (buf[i + tagSize + 2] << 8) + buf[i + tagSize + 3]
            if (tagSize != previousTagSize) {
                throw new Error("Invalid previous tag size: " + tagSize + ", expected: " + previousTagSize)
            }

            // send a chunk: flv tag(each video tag is a frame of the video, total 11 + dataSize bytes) + previousTagSize(4 bytes)
            ctx.write(new Uint8Array(buf.slice(i, i + tagSize + 4)))
            ctx.flush()

            i = i + tagSize + 4
        }
    }
    ```

3. Create a resource with lang `html` and url `/resource/foo.html`.
    ```html
    <script src="https://cdnjs.cloudflare.com/ajax/libs/flv.js/1.6.2/flv.min.js"></script>
    <video id="videoElement"></video>
    <script>
        if (flvjs.isSupported()) {
            const flvPlayer = flvjs.createPlayer({
                type: "flv",
                url: "/service/foo",
                enableWorker: true, // https://github.com/bilibili/flv.js/issues/322
            })
            flvPlayer.attachMediaElement(document.getElementById("videoElement"))
            flvPlayer.load()
            flvPlayer.play()
        }
    </script>
    ```

4. You can preview at `http://127.0.0.1:8090/resource/foo.html`.
