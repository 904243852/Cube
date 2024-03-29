type GenericByteArray = string | Uint8Array | Array<number> | Buffer

interface ServiceContext {
    "Native Go Implementation"; /* it is not allowed to create it by yourself */
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

declare class ServiceResponse {
    constructor(status: number, header: { [name: string]: string | number; }, data?: GenericByteArray);
    setStatus(status: number): void;
    setHeader(name: string, value: string): void;
    setData(data: GenericByteArray): void;
    setCookie(name: string, value: string): void;
}

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

type BlockingQueue = {
    put(input: any, timeout: number): void;
    poll(timeout: number): any;
    drain(size: number, timeout: number): any[];
}
declare function $native(name: "bqueue"): (size: number) => BlockingQueue;

declare function $native(name: "cache"): {
    set(key: any, value: any, timeout: number): void;
    get(key: any): any;
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

type Decimal = {
    add(value: Decimal): Decimal;
    sub(value: Decimal): Decimal;
    mul(value: Decimal): Decimal;
    div(value: Decimal): Decimal;
}
declare function $native(name: "decimal"): (value: string) => Decimal;

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
declare function $native(name: "http"): (options?: HttpOptions) => {
    request(method: string, url: string, header?: { [name: string]: string; }, body?: GenericByteArray): { status: number; header: { [name: string]: string; }; data: Buffer; };
}

type Image = {
    width: number;
    height: number;
    get(x: number, y: number): number;
    set(x: number, y: number, p: number): void;
    toBytes(): Buffer;
    resize(width: number, height: number): Image;
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

declare function $native(name: "template"): (name: string, input: { [name: string]: any; }) => string;

declare function $native(name: "ulid"): () => string;

type XmlNode = {
    find(expr: string): XmlNode[];
    findOne(expr: string): XmlNode;
    innerText(): string;
    toString(): string;
}
declare function $native(name: "xml"): (content: string) => XmlNode;

declare function $native(name: "zip"): {
    write(data: { [name: string]: string | Buffer; }): Buffer;
    read(data: GenericByteArray): {
        getFiles(): {
            getName(): string;
            getData(): Buffer;
        }[];
    };
}
