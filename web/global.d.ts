type GenericByteArray = string | Uint8Array | Array<number> | Buffer

//#region service

interface ServiceContext {
    "Native Service Context"; /* it is not allowed to create it by yourself */
    getHeader(): { [name: string]: string; };
    getURL(): { path: string; params: { [name: string]: string[]; }; };
    getBody(): Buffer;
    getMethod(): "GET" | "POST" | "PUT" | "DELETE";
    getForm(): { [name: string]: string[]; };
    getPathVariables(): { [name: string]: string; };
    getFile(name: string): { name: string; size: number; data: Buffer; };
    getCerts(): any[];
    getCookie(name: string): { value: string; };
    upgradeToWebSocket(): ServiceWebSocket;
    getReader(): { readByte(): number; read(count: number): Buffer; };
    getPusher(): { push(target: string, options: any): void; };
    write(data: GenericByteArray): number;
    flush(): void;
    resetTimeout(timeout: number): void;
}

interface ServiceWebSocket {
    read(): { messageType: number; data: Buffer; };
    send(data: GenericByteArray);
    close();
}

//#endregion

//#region builtin

declare interface Buffer extends Array<number> {
    toString(encoding?: "utf8" | "hex" | "base64" | "base64url"): string;
    toJson(): any;
}
declare interface BufferConstructor {
    from(input: GenericByteArray, encoding?: "utf8" | "hex" | "base64" | "base64url"): Buffer;
}
declare var Buffer: BufferConstructor;

interface Console {
    log(...data: any[]): void;
    debug(...data: any[]): void;
    info(...data: any[]): void;
    warn(...data: any[]): void;
    error(...data: any[]): void;
}
declare var console: Console;

interface Date {
    toString(layout?: string): string
}
interface DateConstructor {
    toDate(value: string, layout: string): Date
}

declare class Decimal {
    constructor(value: string);
    add(value: Decimal): Decimal;
    sub(value: Decimal): Decimal;
    mul(value: Decimal): Decimal;
    div(value: Decimal): Decimal;
    pow(value: Decimal): Decimal;
    mod(value: Decimal): Decimal;
    compare(value: Decimal): -1 | 0 | 1;
    abs(): Decimal;
    string(): string;
    stringFixed(places: number): string;
}

interface IntervalId { "Native Interval Id"; }
declare function setInterval(handler: Function, timeout?: number, ...arguments: any[]): IntervalId;
declare function clearInterval(id: IntervalId): void;
interface TimeoutId { "Native Interval Id"; }
declare function setTimeout(handler: Function, timeout?: number, ...arguments: any[]): TimeoutId;
declare function clearTimeout(id: TimeoutId): void;

declare function fetch(url: string, options?: { method?: "GET" | "POST" | "PUT" | "DELETE"; header?: { [name: string]: string }; body?: string; }): Promise<{ status: number; headers: { [name: string]: string }; buffer(): Buffer; json(): any; text(): string; }>;

declare class ServiceResponse {
    constructor(status: number, header: { [name: string]: string | number; }, data?: GenericByteArray);
    setStatus(status: number): void;
    setHeader(name: string, value: string): void;
    setData(data: GenericByteArray): void;
    setCookie(name: string, value: string): void;
}

//#endregion

//#region native module

type BlockingQueue = {
    put(input: any, timeout: number): void;
    poll(timeout: number): any;
    drain(size: number, timeout: number): any[];
}
declare function $native(name: "bqueue"): (size: number) => BlockingQueue;

declare function $native(name: "cache"): {
    set(key: any, value: any, timeout: number): void;
    get(key: any): any;
    has(key: any): boolean;
    expire(key: any, timeout: number): void;
}

type HashAlgorithm = "md5" | "sha1" | "sha256" | "sha512"
declare function $native(name: "crypto"): {
    createCipher(algorithm: "aes-ecb", key: GenericByteArray, options: { padding: "none" | "pkcs5" | "pkcs7"; }): {
        encrypt(input: GenericByteArray): Buffer;
        decrypt(input: GenericByteArray): Buffer;
    };
    createHash(algorithm: HashAlgorithm): {
        sum(input: GenericByteArray): Buffer;
    };
    createHmac(algorithm: HashAlgorithm): {
        sum(input: GenericByteArray, key: GenericByteArray): Buffer;
    };
    createRsa(): {
        generateKey(): { privateKey: Buffer; publicKey: Buffer; };
        encrypt(input: GenericByteArray, publicKey: GenericByteArray, padding: "pkcs1" | "oaep" = "pkcs1"): Buffer;
        decrypt(input: GenericByteArray, privateKey: GenericByteArray, padding: "pkcs1" | "oaep" = "pkcs1"): Buffer;
        sign(input: GenericByteArray, key: GenericByteArray, algorithm: HashAlgorithm, padding: "pkcs1" | "pss" = "pkcs1"): Buffer;
        verify(input: GenericByteArray, sign: GenericByteArray, key: GenericByteArray, algorithm: HashAlgorithm, padding: "pkcs1" | "pss" = "pkcs1"): boolean;
    };
}

type DatabaseTransaction = {
    query(stmt: string, ...params: any[]): any[];
    exec(stmt: string, ...params: any[]): number;
    commit(): void;
    rollback(): void;
}
declare function $native(name: "db"): {
    /**
     * begin a transaction
     *
     * @param func function during this transaction
     * @param isolation transaction isolation level: 0 = Default, 1 = Read Uncommitted, 2 = Read Committed, 3 = Write Committed, 4 = Repeatable Read, 5 = Snapshot, 6 = Serializable, 7 = Linearizable
     */
    transaction(func: (tx: DatabaseTransaction) => void, isolation: number = 0): void;
} & Pick<DatabaseTransaction, "query" | "exec">

declare function $native(name: "email"): (host: string, port: number, username: string, password: string) => {
    send(receivers: string[], subject: string, content: string, attachments: { Name: string; ContentType: string; Base64: string; }[]): void;
}

declare function $native(name: "event"): {
    emit(topic: string, data: any): void;
    createSubscriber(...topics: string[]): {
        next(): any;
    };
    on(topic: string, func: (data: any) => void): {
        cancel(): void;
    };
}

declare function $native(name: "file"): {
    read(name: string): Buffer;
    readRange(name: string, offset: number, length: number): Buffer;
    write(name: string, content: GenericByteArray): void;
    writeRange(name: string, offset: number, content: GenericByteArray): void;
    stat(name: string): {
        name(): string;
        size(): number;
        isDir(): boolean;
        mode(): string;
        modTime(): string;
    };
    list(name: string): string[];
}

type HttpOptions = Partial<{
    caCert: string;
    insecureSkipVerify: boolean;
    isHttp3: boolean;
    proxy: string;
}> | {
    cert: string;
    key: string;
}
type FormData = {
    "Native Form Data"
}
declare function $native(name: "http"): (options?: HttpOptions) => {
    request(method: string, url: string, header?: { [name: string]: string; }, body?: GenericByteArray | FormData): { status: number; header: { [name: string]: string; }; data: Buffer; };
    toFormData(data: { [name: string]: string | { filename: string; data: GenericByteArray; }; }): FormData;
}

type Image = {
    width(): number;
    height(): number;
    get(x: number, y: number): number;
    set(x: number, y: number, p: number): void;
    /** set rotate for next drawings */
    setDrawRotate(degrees: number): void;
    /** set font face for next drawings */
    setDrawFontFace(fontSize?: number, ttf?: GenericByteArray): void;
    /** set RGBA color for next drawings */
    setDrawColor(color: string | [red: number, green: number, blue: number, alpha?: number]): void;
    getStringWidthAndHeight(s: string): { width: number; height: number; };
    drawImage(image: Image, x: number, y: number): void;
    drawString(s: string, x: number, y: number, ax?: number, ay?: number, width?: number, lineSpacing?: number): void;
    resize(width: number, height?: number): Image;
    toJPG(quality?: number): Buffer;
    toPNG(): Buffer;
}
declare function $native(name: "image"): {
    create(width: number, height: number): Image;
    parse(input: GenericByteArray): Image;
}

declare function $native(name: "lock"): (name: string) => {
    lock(timeout: number): void;
    unlock(): void;
}

declare function $native(name: "pipe"): (name: string) => BlockingQueue;

type TCPSocketConnection = {
    read(size?: number): Buffer;
    readLine(): Buffer;
    write(data: GenericByteArray): number;
    close(): void;
}
type UDPSocketConnection = {
    read(size?: number): Buffer;
    write(data: GenericByteArray, host?: string, port?: number): number;
    close(): void;
}
declare function $native(name: "socket"): {
    (protocol: "tcp"): {
        dial(host: string, port: number): TCPSocketConnection;
        listen(port: number): {
            accept(): TCPSocketConnection;
        };
    };
    (protocol: "udp"): {
        dial(host: string, port: number): UDPSocketConnection;
        listen(port: number): UDPSocketConnection;
        listenMulticast(host: string, port: number): UDPSocketConnection;
    };
}

declare function $native(name: "process"): {
    exec(command: string, ...params: string[]): Buffer;
    pexec(command: string, ...params: string[]): Promise<Buffer>;
};

declare function $native(name: "template"): (name: string, input: { [name: string]: any; }) => string;

declare function $native(name: "ulid"): () => string;

type XmlNode = {
    find(expr: string): XmlNode[];
    findOne(expr: string): XmlNode;
    innerText(): string;
    toString(): string;
}
declare function $native(name: "xml"): (content: string) => XmlNode;

type ZipEntry = {
    name: string;
    compressedSize64: number;
    uncompressedSize64: number;
    comment: string;
    getData(): Buffer;
}
declare function $native(name: "zip"): {
    write(data: { [name: string]: string | Buffer; }): Buffer;
    read(data: GenericByteArray): {
        getEntries(): ZipEntry[];
        getData(name: string): Buffer;
    };
}

//#endregion
