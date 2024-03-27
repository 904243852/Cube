/**
 * Service context
 */
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
    write(data: Uint8Array | Buffer | string): number;
    flush(): void;
    resetTimeout(timeout: number): void;
}

/**
 * Service websocket
 */
interface ServiceWebSocket {
    read(): { messageType: number; data: Buffer; };
    send(data: Uint8Array | Buffer | string);
    close();
}

declare class ServiceResponse {
    constructor(status: number, header: { [name: string]: string | number; }, data?: Uint8Array | Buffer | string);
    setStatus(status: number): void;
    setHeader(name: string, value: string): void;
    setData(data: Uint8Array | Buffer | string): void;
    setCookie(name: string, value: string): void;
}

/**
 * Buffer
 */
declare interface Buffer extends Array<number> {
    toString(encoding?: "utf8" | "hex" | "base64" | "base64url"): string;
    toJson(): any;
}
declare interface BufferConstructor {
    from(input: string | Uint8Array | Array<number>, encoding?: "utf8" | "hex" | "base64" | "base64url"): Buffer;
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

declare function $native(name: "crypto"): {
    createCipher(algorithm: "aes-ecb", key: string | Uint8Array | Buffer, options: { padding: "none" | "pkcs5" | "pkcs7"; }): {
        encrypt(input: string | Uint8Array | Buffer): Buffer;
        decrypt(input: string | Uint8Array | Buffer): Buffer;
    };
    createHash(algorithm: "md5" | "sha1" | "sha256" | "sha512"): {
        sum(input: string | Uint8Array | Buffer): Buffer;
    };
    createHmac(algorithm: "md5" | "sha1" | "sha256" | "sha512"): {
        sum(input: string | Uint8Array | Buffer, key: string | Uint8Array | Buffer): Buffer;
    };
    createRsa(): {
        generateKey(): { privateKey: Buffer; publicKey: Buffer; };
        encrypt(input: string | Uint8Array | Buffer, publicKey: string | Uint8Array | Buffer, padding: "pkcs1" | "oaep" = "pkcs1"): Buffer;
        decrypt(input: string | Uint8Array | Buffer, privateKey: string | Uint8Array | Buffer, padding: "pkcs1" | "oaep" = "pkcs1"): Buffer;
        sign(input: string | Uint8Array | Buffer, key: string | Uint8Array | Buffer, algorithm: "md5" | "sha1" | "sha256" | "sha512", padding: "pkcs1" | "pss" = "pkcs1"): Buffer;
        verify(input: string | Uint8Array | Buffer, sign: string | Uint8Array | Buffer, key: string | Uint8Array | Buffer, algorithm: "md5" | "sha1" | "sha256" | "sha512", padding: "pkcs1" | "pss" = "pkcs1"): boolean;
    };
}

declare function $native(name: "db"): {
    /**
     * begin a transaction
     *
     * @param isolation transaction isolation level: 0 = Default, 1 = Read Uncommitted, 2 = Read Committed, 3 = Write Committed, 4 = Repeatable Read, 5 = Snapshot, 6 = Serializable, 7 = Linearizable
     * @return transaction
     */
    beginTx(isolation: number = 0): {
        query(stmt: string, ...params: any[]): any[];
        exec(stmt: string, ...params: any[]): number;
        commit(): void;
        rollback(): void;
    };
    query(stmt: string, ...params: any[]): any[];
    exec(stmt: string, ...params: any[]): number;
}

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
    write(name: string, content: string | Uint8Array | Buffer): void;
    writeRange(name: string, offset: number, content: string | Uint8Array | Buffer): void;
    stat(name: string): {
        name(): string;
        size(): number;
        isDir(): boolean;
        mode(): string;
        modTime(): string;
    };
    list(name: string): string[];
}

declare function $native(name: "http"): (options?: { caCert?: string; cert?: string; key?: string; insecureSkipVerify?: boolean; isHttp3?: boolean; proxy?: string; }) => {
    request(method: string, url: string, header?: { [name: string]: string; }, body?: string | Uint8Array | Buffer): { status: number; header: { [name: string]: string; }; data: Buffer; };
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
    parse(input: Uint8Array | Buffer): Image;
}

declare function $native(name: "lock"): (name: string) => {
    lock(timeout: number): void;
    unlock(): void;
}

declare function $native(name: "pipe"): (name: string) => BlockingQueue;

type TCPSocketConnection = {
    read(size?: number): Buffer;
    readLine(): Buffer;
    write(data: string | Uint8Array | Buffer): number;
    close(): void;
}
type UDPSocketConnection = {
    read(size?: number): Buffer;
    write(data: string | Uint8Array | Buffer, host?: string, port?: number): number;
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
    read(data: Uint8Array | Buffer): {
        getFiles(): {
            getName(): string;
            getData(): Buffer;
        }[];
    };
}
