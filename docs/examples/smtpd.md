# Create a smtpd using socket module

1. Create a daemon.
    ```typescript
    //?name=smtpd&type=daemon
    export default function (ctx: ServiceContext) {
        const tcpd = $native("socket")("tcp").listen(25)
        while(true) {
            const c = tcpd.accept()
            console.debug(toEmail(readData(c)))
        }
    }

    function readData(connection) {
        connection.write("220 My Mail Sever\n")

        let data = "",
            s = String.fromCharCode(...connection.readLine())
        while (s.length) {
            switch (s.substring(0, 4).replace(/[\r\n]*$/, "").toUpperCase()) {
                case "HELO":
                case "EHLO":
                case "MAIL":
                case "RCPT":
                case "RSET":
                case ".":
                    connection.write("250 OK\n")
                    break
                case "DATA":
                    connection.write("354 OK\n")
                    break
                case "QUIT":
                    connection.write("221 Bye\n")
                    connection.close()
                    return data
                default:
                    data += s
                    break
            }
            s = String.fromCharCode(...connection.readLine())
        }
        return null
    }

    function toEmail(data) {
        data = (data || "").replace(/\r\n/g, "\n")
        return {
            subject: data.match(/^Subject: (.*)$/m)?.[1],
            from: data.match(/^From: .*(<.*>)$/m)?.[1],
            to: data.match(/^To: .*(<.*>)$/m)?.[1],
            body: Buffer.from(data.match(/\n\n(.*)/m)?.[1] || "", "base64").toString(),
        }
    }
    ```

2. You can test it using telnet such as:
    ```bash
    telnet 127.0.0.1 25
    ```
    ```
    HELO abc.com
    MAIL FROM: <noone@abc.com>
    RCPT TO: <zhangsan@127.0.0.1>
    DATA
    To: zhangsan@127.0.0.1
    From: noone@abc.com
    Subject: greeting

    aGVsbG8=
    .
    QUIT
    ```
