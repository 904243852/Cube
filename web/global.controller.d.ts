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

declare class ServiceResponse {
    constructor(status: number, header: { [name: string]: string | number; }, data?: GenericByteArray);
    setStatus(status: number): void;
    setHeader(name: string, value: string): void;
    setData(data: GenericByteArray): void;
    setCookie(name: string, value: string): void;
}

//#endregion
