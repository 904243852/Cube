# Create an rtmpd and convert the rtmp stream into an HTTP-FLV stream

1. Create a daemon with name `rtmpd` and start.
    ```typescript
    export default function () {
        const c = new RtmpConnection($native("socket")("tcp").listen(1935).accept())

        // 1. 握手阶段
        c.handshake()

        // 2. 建立连接阶段
        // 2.1 客户端发起连接请求
        c.readChunk()
        // 2.2 服务器设置客户端的应答窗口大小
        c.write(c.toChunk(
            2, // Chunk Stream ID: 2
            0x05, // Type ID: Window Acknowledgement Size
            [0x00, 0x4c, 0x4b, 0x40], // Window acknowledgement size: 5000000
        ))
        // 2.3 服务器设置客户端的发送带宽大小
        c.write(c.toChunk(
            2, // Chunk Stream ID: 2
            0x06, // Type ID: Set Peer Bandwidth
            [
                0x00, 0x4c, 0x4b, 0x40, // Window acknowledgement size: 5000000
                0x02, // Limit Type: Dynamic
            ],
        ))
        // 2.4 服务器设置客户端的接收块大小
        c.write(c.toChunk(
            2, // Chunk Stream ID: 2
            0x01, // Type ID: Set Chunk Size
            [
                // 0x00, 0x00, 0x10, 0x00, // Chunk size: 4096
                0x00, 0x00, 0xea, 0x60, // Chunk size: 60000
            ],
        ))
        // 2.5 服务器响应连接结果
        c.write(c.toChunk(3, 0x14, AMF0.encodes(
            "_result",
            1,
            {
                fmsVer: "FMS/3,0,1,123",
                capabilities: 31,
            },
            {
                levelc: "status",
                code: "NetConnection.Connect.Success",
                description: "Connection succeeded.",
                objectEncoding: 0,
            },
        )))
        // 2.6 客户端设置服务器的接收块大小
        const chunkSizeData = c.readChunk().data
        c.setMaxChunkSize((chunkSizeData[0] << 24) + (chunkSizeData[1] << 16) + (chunkSizeData[2] << 8) + chunkSizeData[3])

        // 3. 建立流阶段
        // 3.1 客户端通知服务器释放推流点
        c.readChunk()
        // 3.2 客户端通知服务器准备推流点
        c.readChunk()
        // 3.3 客户端发起创建流请求
        c.readChunk()
        // 3.4 服务器响应创建流结果
        c.write(c.toChunk(3, 0x14, AMF0.encodes(
            "_result",
            4, // Number
            null, // Null
            1, // Number
        )))

        // 4. 推流阶段
        // 4.1 客户端发起创建绑定推流点请求
        c.readChunk()
        // 4.2 服务器响应创建绑定推流点结果
        c.write(c.toChunk(5, 0x14, AMF0.encodes(
            "onStatus", // Command Name: onStatus
            0, // Transaction ID: 0
            null, // Command Object: Null
            { // Info Object
                level: "status",
                code: "NetStream.Publish.Start", // notify client to push stream
                description: "/ is now published.",
            },
        )))
        // 4.3 客户端向服务器设置媒体元数据
        const metaData = c.readChunk().data.slice(16)

        const cache = $native("cache")
        cache.set("FLV_HEADER_TAG", new Uint8Array([
            // FLV Header
            0x46, 0x4c, 0x56, // 'F'、'L'、'V'
            0x01, // 版本号为 1
            0x05, // 即 0b00000101，其中第 6 位表示是否存在音频，第 8 位表示是否存在视频，其余位均为 0
            0x00, 0x00, 0x00, 0x09, // FLV Header 的字节长度
            // Previous Tag Size 0
            0x00, 0x00, 0x00, 0x00,
        ]), -1)
        cache.set("FLV_META_TAG", FLVTag.encode(0x12, metaData, 0), -1) // Script Tag

        while (true) {
            // 4.4 客户端向服务器推送媒体数据
            const { header, data } = c.readChunk()
            if (header.messageTypeId === 0x14) {
                // 客户端通知服务器解绑释放推流点请求
                console.log(header.messageTypeId, String.fromCharCode(...data))
                c.close()
                $native("event").emit("HTTPFLV_STOP", 0)
                break
            }
            if (!data.length) {
                throw new Error("no data read")
            }

            const tag = FLVTag.encode(header.messageTypeId, data, c.clocks[header.chunkStreamId]) // messageTypeId: 8 = Audio, 9 = Video, 18 = Script

            // https://zhuanlan.zhihu.com/p/611128149
            if (header.messageTypeId === 0x08) { // Audio
                const soundFormat = (data[0] >> 4) & 0x0f
                if ((soundFormat == 10 || soundFormat == 13) && data[1] == 0) {
                    // AAC Sequence Header
                    cache.set("FLV_AACHEADER_TAG", FLVTag.encode(header.messageTypeId, data, 0), -1)
                }
            }
            if (header.messageTypeId === 0x09) { // Video
                const videoCodec = data[0] & 0x0f, // 第一个字节后 4 位表示编码 ID：7 表示 AVC
                    frameType = data[0] >> 4 & 0b0111, // 第一个字节的前 4 位表示帧类型：1 表示关键帧
                    isExHeader = (data[0] >> 4 & 0b1000) !== 0
                if (isExHeader) {
                    throw new Error("video tag ex header not implemented")
                }
                if ((videoCodec == 7 || videoCodec == 12 || videoCodec == 13) && frameType == 1 && data[1] == 0) {
                    // AVC Sequence Header
                    cache.set("FLV_AVCHEADER_TAG", FLVTag.encode(header.messageTypeId, data, 0), -1)
                }
            }

            $native("event").emit("HTTPFLV_TAG", tag)
        }
    }

    class RtmpConnection {
        private socket: {
            read(size?: number): NativeByteArray;
            write(data: string | Uint8Array | NativeByteArray): number;
            close(): void;
        }

        private maxChunkSize = 128

        public clocks = {}

        public headers = {} as { [csid: string]: { chunkStreamId: number, timestamp: number, messageLength: number, messageTypeId: number, messageStreamId: number } }

        constructor(socket) {
            this.socket = socket
        }

        public handshake() { // complex handshake is not supported
            // read c0 and c1
            const c0 = this.socket.read(1),
                c1 = this.socket.read(1536)

            // write s0, s1 and s2
            this.socket.write(c0)
            this.socket.write(c1)
            this.socket.write(c1)

            // read c2
            this.socket.read(1536)
        }

        public setMaxChunkSize(size: number) {
            if (!size) {
                throw new Error("unknown chunk size: " + size)
            }
            this.maxChunkSize = size
        }

        public read(size?: number) {
            if (size === 0) {
                return [] as NativeByteArray
            }

            return this.socket.read(size)
        }

        private readUintBE(size: number) {
            return parseInt(this.read(size).map(i => i.toString(2).padStart(8, "0")).join(""), 2)
        }

        public write(data: string | Uint8Array | NativeByteArray): number {
            return this.socket.write(data)
        }

        public close(): void {
            return this.socket.close()
        }

        public readChunk(size?: number) {
            // 1. Chunk Header
            let header = new Object() as typeof this.headers[0]
            // 1.1 Basic Header（1, 2 or 3 bytes）：Format + Chunk Stream ID
            const a = this.read(1).pop()
            if (!a) {
                throw new Error("failed to read format and chunk stream id in header")
            }
            const format = (a & 0b11000000) >> 6 // Format
            header.chunkStreamId = a & 0b111111 // Chunk Stream ID，即块流 ID，用于区分消息信道，取值范围 [3, 65599]，值 0, 1 和 2 被保留，3 ~ 8用于固定用途，9 ~ 65599 用于自定义
            // 0 表示 Basic Header 总共要占用 2 个字节
            // 1 表示 Basic Header 总共要占用 3 个字节
            // 2 代表该 chunk 是控制信息和一些命令信息
            // 3 代表该 chunk 是客户端发出的 AMF0 命令以及服务端对该命令的应答
            // 4 代表该 chunk 是客户端发出的音频数据，用于 publish
            // 5 代表该 chunk 是服务端发出的 AMF0 命令和数据
            // 6 代表该 chunk 是服务端发出的音频数据，用于 play；或客户端发出的视频数据，用于 publish
            // 7 代表该 chunk 是服务端发出的视频数据，用于 play
            // 8 代表该 chunk 是客户端发出的 AMF0 命令，专用来发送： getStreamLength, play, publish
            if (header.chunkStreamId === 0b000000) { // Chunk Stream ID use 6 + 8 bits
                const [b] = this.read(1)
                header.chunkStreamId = b + 64
            } else if (header.chunkStreamId === 0b000001) { // Chunk Stream ID use 6 + 8 + 8 bits
                const [b, c] = this.read(2)
                header.chunkStreamId = (c << 16) + b + 64 // 第三个字节 * 256 + 第二个字节 + 64
            }

            // 1.2 Message Header（0, 3, 7 or 11 bytes）
            switch (format) {
                case 0x00: // Type 0 (Full), 12 bytes
                    header.timestamp = this.readUintBE(3) // Timestamp，单位毫秒
                    header.messageLength = this.readUintBE(3) // Message Length
                    header.messageTypeId = this.readUintBE(1) // Message Type ID
                    header.messageStreamId = this.readUintBE(4) // Message Stream ID
                    break;
                case 0x01: // Type 1 (Relative Large), 8 bytes
                    header.timestamp = this.readUintBE(3) // Timestamp Delta
                    header.messageLength = this.readUintBE(3) // Message Length
                    header.messageTypeId = this.readUintBE(1) // Message Type ID
                    // Type 1 情况下省略了 Message Stream ID，这个 ID 与上一个 chunk message 相同
                    break;
                case 0x02: // Type 2 (Relative Timestamp Only), 4 bytes
                    header.timestamp = this.readUintBE(3) // Timestamp
                    // Type 2 情况下省略了 Message Length、Message Type ID 和 Message Stream ID，省略的这个与上一个 chunk message 相同
                    break;
                case 0x03: // Type 3 (Relative Single Byte), 1 bytes
                    // Type 3 情况下 Message Header 全部省略了，跟上一个 chunk message 完全相同
                    break;
                default:
                    throw new Error("unknown chunk format: " + format)
            }
            header = this.headers[header.chunkStreamId] = {
                ...this.headers[header.chunkStreamId],
                ...header,
            }

            // 1.3 Extended Timestamp（0 or 4 bytes）
            // 通常我们使用 Timestamp 来表示时间戳（包括绝对时间戳和 Timestamp delta），但当时间戳的值超过 16777215（即 0xffffff）时，Timestamp 会被置为 0xffffff，此时我们使用 Extended Timestamp 字段并由该字段表示时间戳
            if (header.timestamp === 0xffffff) {
                header.timestamp = this.readUintBE(4) // Extended Timestamp
            }

            if (size === undefined) { // 当前 chunk 没有被拆分多个
                if (format === 0x00) {
                    // 视频流（或音频流）的第一个时间戳为绝对时间戳，后续（type-1 和 type-2 chunk）的时间戳均为 timestamp delta，即当前时间戳与上一个时间戳的差值
                    this.clocks[header.chunkStreamId] = header.timestamp
                } else {
                    this.clocks[header.chunkStreamId] += header.timestamp
                }
            }

            if (header.messageTypeId === 0) {
                throw new Error("message type id can not be 0: " + header.messageTypeId)
            }

            const msgLength = size === undefined ? header.messageLength : size
            if (!msgLength) {
                throw new Error("unknown message length: " + msgLength)
            }

            // 2. Chunk Data
            const data = this.read(Math.min(this.maxChunkSize, msgLength)) // A message consists of one or more blocks, such as 1...n, and the chunk data length of the nth block is `messageLength - (n-1) * maximumChunkSize` (maximumChunkSize defaults to 128)
            while (data.length < msgLength) {
                data.push(...this.readChunk(msgLength - data.length).data)
            }

            return {
                header,
                data,
            }
        }

        public toChunk(chunkStreamId: number, typeId: number, body: Uint8Array | number[]): Uint8Array {
            return new Uint8Array([
                // Header
                chunkStreamId & 0b00111111, // Format and Chunk Stream ID
                0x00, 0x00, 0x00, // Timestamp
                body.length >> 16 & 0xff, body.length >> 8 & 0xff, body.length & 0xff, // Body size
                typeId & 0xff, // Type ID: AMF0 Command
                0x00, 0x00, 0x00, 0x00, // Stream id
                // Body
                ...body,
            ])
        }
    }

    class AMF0 {
        public static decodea(input: NativeByteArray, start: number): { value: number | boolean | string | object | null; end: number; } {
            if (input[start] === 0x00) { // Number
                return {
                    value: 0, // TODO
                    end: start + 1 + 8,
                }
            }
            if (input[start] === 0x01) {
                return {
                    value: input[start + 1],
                    end: start + 1 + 1,
                }
            }
            if (input[start] === 0x02) { // String
                const length = (input[start + 1] << 8) + input[start + 2]
                return {
                    value: String.fromCharCode(...input.slice(start + 3, start + 3 + length)),
                    end: start + 1 + 2 + length,
                }
            }
            if (input[start] === 0x08) { // ECMA Array
                const arrayLength = (input[start + 1] << 24) + (input[start + 2] << 16) + (input[start + 3] << 8) + input[start + 4],
                    output = []
                let i = start + 1 + 4
                for (let c = 0; c < arrayLength; c++) {
                    const propertyNameLength = (input[i] << 8) + input[i + 1]
                    const propertyName = String.fromCharCode(...input.slice(i + 2, i + 2 + propertyNameLength))
                    const { value: propertyValue, end: propertyValueEnd } = AMF0.decodea(input, i + 2 + propertyNameLength)
                    output.push({
                        [propertyName]: propertyValue,
                    })
                    i = propertyValueEnd
                }
                if (!(input[i] === 0x00 && input[i + 1] === 0x00 && input[i + 2] === 0x09)) {
                    throw new Error("end of object marker is expect")
                }
                return {
                    value: output,
                    end: i + 3,
                }
            }
            throw new Error("unknown amf0 type: " + input[start] + "@" + start)
        }

        public static decodes(input: NativeByteArray): (number | boolean | string | object | null)[] {
            const output = []
            for (let i = 0; i < input.length;) {
                const { value, end } = AMF0.decodea(input, i)
                output.push(value)
                i = end
            }
            return output
        }

        public static encodes(...values: (number | boolean | string | object | null)[]): Uint8Array {
            return new Uint8Array([].concat.apply([], values.map(v => {
                if (typeof v === "number") {
                    if (v === 0) {
                        return [
                            0x00, // Type is Number
                            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                        ]
                    }
                    const [a, b] = (v.toString(2) + ".").split("."),
                        ab = a + b,
                        s = v > 0 ? "0" : "1",
                        e = (1023 + (ab.indexOf("1") * -1 || (a.length - 1))).toString(2).padStart(11, "0"),
                        t = ab.substring(ab.indexOf("1") + 1).padEnd(52, "0")
                    return [
                        0x00, // Type is Number
                        ...(s + e + t).split(/([01]{8})/).filter(i => i).map(i => parseInt(i, 2)), // Number value
                    ]
                }
                if (typeof v === "boolean") {
                    return [
                        0x01, // Type is Boolean
                        v ? 0x01 : 0x00, // Boolean value
                    ]
                }
                if (typeof v === "string") {
                    return [
                        0x02, // Type is String
                        v.length >> 8 & 0xff, v.length & 0xff, // String length
                        ...v.split("").map(i => i.charCodeAt(0)), // String value
                    ]
                }
                if (typeof v === "object" && v !== null) {
                    return [
                        0x03, // Type is Object
                        ...Object.keys(v).reduce((p, c) => {
                            p.push(...[
                                // Property name
                                c.length >> 8 & 0xff, c.length & 0xff, // String length
                                ...c.split("").map(i => i.charCodeAt(0)), // String value
                                // Property value
                                ...AMF0.encodes(v[c]),
                            ])
                            return p
                        }, []),
                        0x00, 0x00, 0x09, // End of Object Marker
                    ]
                }
                if (v === null) {
                    return [
                        0x05 // Type is Null
                    ]
                }
                throw new Error("unknown amf0 value: " + v)
            })))
        }
    }

    class FLVTag {
        public static encode(dataType: number, data: number[], timestamp: number = 0) {
            if (dataType !== 8 && dataType !== 9 && dataType !== 18) {
                throw new Error("unknown flv tag data type: " + dataType)
            }
            const dataSize = data.length,
                previousTagSize = 11 + dataSize
            return new Uint8Array([
                // Tag Header
                dataType, // DataType
                (dataSize >> 16) & 0xff, (dataSize >> 8) & 0xff, dataSize & 0xff, // DataSize
                (timestamp >> 16) & 0xff, (timestamp >> 8) & 0xff, timestamp & 0xff, (timestamp >> 24) & 0xff, // Timestamp
                0x00, 0x00, 0x00, // Stream Id
                // Tag Data
                ...data,
                // Previous Tag Size
                (previousTagSize >> 24) & 0xff, (previousTagSize >> 16) & 0xff, (previousTagSize >> 8) & 0xff, previousTagSize & 0xff,
            ])
        }
    }
    ```

2. Create a controller with url `/service/httpflv`.
    ```typescript
    export default function (ctx: ServiceContext) {
        ctx.resetTimeout(30 * 60 * 1000)

        const cache = $native("cache")

        let playing = false, c = 0

        const a = $native("event").on("HTTPFLV_TAG", function (tag) {
            if (!playing) {
                [
                    cache.get("FLV_HEADER_TAG"),
                    cache.get("FLV_META_TAG"),
                    cache.get("FLV_AACHEADER_TAG"),
                    cache.get("FLV_AVCHEADER_TAG"),
                ].filter(i => i).forEach(i => {
                    ctx.write(i)
                })
                playing = true
            }

            ctx.write(tag)
            if (++c % 10 == 0) {
                ctx.flush()
            }
        })

        const b = $native("event").on("HTTPFLV_STOP", function () {
            a.cancel()
            b.cancel()
        })
    }
    ```

3. Create a resource with lang `html` and url `/resource/httpflv`.
    ```html
    <script src="https://cdn.bootcdn.net/ajax/libs/flv.js/1.6.2/flv.min.js"></script>
    <video id="videoElement" onclick="player.play()" style="width: 100%; height: 100%;"></video>
    <script>
        if (flvjs.isSupported()) {
            var player = flvjs.createPlayer({
                type: "flv",
                url: "/service/1_httpflv_srv",
                enableWorker: true,
                enableStashBuffer: true,
            })
            player.attachMediaElement(document.getElementById("videoElement"))
            player.load()
            // player.play() // play() failed here because the user didn't interect with the document first. see https://goo.gl/xX8pDD
        }
    </script>
    ```

4. Push a stream using ffmpeg.
    ```bash
    ffmpeg -re \
        -stream_loop -1 \
        -i "https://s.xlzys.com/play/9b64Eq9e/index.m3u8" \
        -vcodec libx264 -r 25 -b:v 800000 \
        -acodec aac -ac 2 -ar 44100 -ab 128k \
        -af "loudnorm" \
        -vf "scale=-1:720" \
        -f flv \
        -threads 5 -preset ultrafast \
        rtmp://127.0.0.1:1935
    ```

5. You can preview at `http://127.0.0.1:8090/resource/httpflv` and click the screen to play.
