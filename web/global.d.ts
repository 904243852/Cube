declare class ServiceContext {
    getHeader(): { [name: string]: string; };
    getURL(): { path: string; params: { [name: string]: string[]; }; };
    getBody(): Uint8Array;
    getJsonBody(): any;
    getMethod(): "GET" | "POST" | "PUT" | "DELETE";
    getForm(): { [name: string]: string[]; };
    getPathVariables(): { [name: string]: string; };
    getFile(name: string): { name: string; size: number; data: Uint8Array; };
    getCerts(): { raw: Uint8Array; }[];
    upgradeToWebSocket(): ServiceWebSocket;
    getReader(): { readByte(): number; read(count: number): Uint8Array; };
    getPusher(): { push(target: string, options: any): void; };
    write(data: Uint8Array | string): number;
    flush(): void;
};
declare class ServiceResponse {
    constructor(status: number, header: { [name: string]: string | number; }, data?: Uint8Array | string);
    setStatus(status: number): void;
    setHeader(header: { [name: string]: string; }): void;
    setData(data: Uint8Array | string): void;
};
declare class ServiceWebSocket {
    read(): { messageType: number; data: Uint8Array; };
    send(data: Uint8Array | string);
    close();
};
declare function $native(name: "base64"): {
    encode(input: string | Uint8Array): string;
    decode(input: string): Uint8Array;
};
type BlockingQueue = {
    put(input: any, timeout: number): void;
    poll(timeout: number): any;
    drain(size: number, timeout: number): any[];
};
declare function $native(name: "bqueue"): (size: number) => BlockingQueue;
declare function $native(name: "cache"): {
    set(key: any, value: any, timeout: number): void;
    get(key: any): any;
};
declare function $native(name: "crypto"): {
    createHash(algorithm: "md5" | "sha1" | "sha256" | "sha512"): {
        sum(input: string | Uint8Array): Uint8Array;
    };
    createHmac(algorithm: "md5" | "sha1" | "sha256" | "sha512"): {
        sum(input: string | Uint8Array, key: string | Uint8Array): Uint8Array;
    };
    createRsa(): {
        generateKey(): { privateKey: Uint8Array; publicKey: Uint8Array; };
        encrypt(input: string | Uint8Array, key: string | Uint8Array): Uint8Array;
        decrypt(input: string | Uint8Array, key: string | Uint8Array): Uint8Array;
        sign(input: string | Uint8Array, key: string | Uint8Array, algorithm: "md5" | "sha1" | "sha256" | "sha512"): Uint8Array;
        signPss(input: string | Uint8Array, key: string | Uint8Array, algorithm: "md5" | "sha1" | "sha256" | "sha512"): Uint8Array;
        verify(input: string | Uint8Array, sign: string | Uint8Array, key: string | Uint8Array, algorithm: "md5" | "sha1" | "sha256" | "sha512"): boolean;
        verifyPss(input: string | Uint8Array, sign: string | Uint8Array, key: string | Uint8Array, algorithm: "md5" | "sha1" | "sha256" | "sha512"): boolean;
    };
};
declare function $native(name: "db"): {
    beginTx(): {
        query(stmt: string, ...params: any[]): any[];
        exec(stmt: string, ...params: any[]): number;
        commit(): void;
        rollback(): void;
    };
    query(stmt: string, ...params: any[]): any[];
    exec(stmt: string, ...params: any[]): number;
};
type Decimal = {
    add(value: Decimal): Decimal;
    sub(value: Decimal): Decimal;
    mul(value: Decimal): Decimal;
    div(value: Decimal): Decimal;
};
declare function $native(name: "decimal"): (value: string) => Decimal;
declare function $native(name: "email"): (host: string, port: number, username: string, password: string) => {
    send(receivers: string[], subject: string, content: string, attachments: { Name: string; ContentType: string; Base64: string; }[]): void;
};
declare function $native(name: "event"): {
    emit(topic: string, data: any): void;
    on(topic: string, func: (data: any) => void): void;
};
declare function $native(name: "file"): {
    read(name: string): Uint8Array;
    readRange(name: string, offset: number, length: number): Uint8Array;
    write(name: string, content: string | Uint8Array): void;
    writeRange(name: string, offset: number, content: string | Uint8Array): void;
    stat(name: string): {
        name(): string;
        size(): number;
        isDir(): boolean;
        mode(): string;
        modTime(): string;
    };
    list(name: string): string[];
};
declare function $native(name: "http"): (options?: { caCert?: string; cert?: string; key?: string; insecureSkipVerify?: boolean; isHttp3?: boolean; proxy?: string; }) => {
    request(method: string, url: string, header?: { [name: string]: string; }, body?: string | Uint8Array): { status: number; header: { [name: string]: string; }; data: { toBytes(): Uint8Array; toString(): string; toJson(): any; }; };
};
type Image = {
    width: number;
    height: number;
    get(x: number, y: number): number;
    set(x: number, y: number, p: number): void;
    toBytes(): Uint8Array;
    resize(width: number, height: number): Image;
};
declare function $native(name: "image"): {
    new(width: number, height: number): Image;
    parse(input: Uint8Array): Image;
};
declare function $native(name: "lock"): (name: string) => {
    lock(timeout: number): void;
    unlock(): void;
};
declare function $native(name: "pipe"): (name: string) => BlockingQueue;
type SocketConnection = {
    readLine(): Uint8Array;
    write(data: string | Uint8Array): number;
    close(): void;
};
declare function $native(name: "socket"): {
    listen(protocol: "tcp" | "udp", port: number): {
        accept(): SocketConnection;
    };
    dial(protocol: "tcp" | "udp", host: string, port: number): SocketConnection;
};
declare function $native(name: "template"): (name: string, input: { [name: string]: any; }) => string;
declare function $native(name: "ulid"): () => string;
declare function $native(name: "zip"): {
    write(data: { [name: string]: string | Uint8Array; }): Uint8Array;
    read(data: Uint8Array): {
        getFiles(): {
            getName(): string;
            getData(): Uint8Array;
        }[];
    };
};