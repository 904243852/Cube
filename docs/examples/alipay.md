# Pay online using Alipay

1. Create a controller with url `/service/alipay`.
    ```typescript
    export default ctx => (app => app.run(ctx))(new class Alipay {
        public static readonly URL = "https://openapi-sandbox.dl.alipaydev.com/gateway.do" // 支付宝网关地址

        public static readonly APP_ID = "账号的 APPID"

        public static readonly MERCHANT_PRIVATE_KEY = ["-----BEGIN RSA PRIVATE KEY-----", "PKCS#1 格式的商户私钥", "-----END RSA PRIVATE KEY-----"].join("\n")

        public static readonly ALIPAY_PUBLIC_KEY = ["-----BEGIN PUBLIC KEY-----", "PKCS#1 格式的支付宝公钥", "-----END PUBLIC KEY-----"].join("\n")

        public static readonly NOTIFY_URL = "支付回调在公网的接口地址，如 http://.../service/alipay"

        public run(ctx: ServiceContext) {
            switch (ctx.getMethod()) {
                case "GET":
                    return this.pay($native("ulid")(), 0.01, "壹分商品")
                case "POST":
                    const form = ctx.getForm(),
                        data = Object.keys(form).reduce((p, c) => {
                            p[c] = form[c][0]
                            return p
                        }, {})
                    return this.notify(data)
                default:
                    throw new Error("not implemented")
            }
        }

        /**
         * 支付
         */
        public pay(outTradeNo: string, totalAmount: number, subject: string) {
            const data = {
                app_id: Alipay.APP_ID,
                method: "alipay.trade.page.pay",
                charset: "utf-8",
                sign_type: "RSA2",
                sign: "",
                timestamp: new Date(+new Date() + 8 * 3600_000).toISOString().replace("T", " ").replace(/\.\d{3}Z$/, ""),
                version: "1.0",
                notify_url: Alipay.NOTIFY_URL,
                biz_content: JSON.stringify({
                    out_trade_no: outTradeNo, // 商户订单号。由商家自定义，64个字符以内，仅支持字母、数字、下划线且需保证在商户端不重复。
                    total_amount: totalAmount, // 订单总金额，单位为元，精确到小数点后两位，取值范围为 [0.01,100000000]。金额不能为0。
                    subject, // 订单标题。注意：不可使用特殊字符，如 /，=，& 等。
                    product_code: "FAST_INSTANT_TRADE_PAY", // 销售产品码，与支付宝签约的产品码名称。注：目前电脑支付场景下仅支持FAST_INSTANT_TRADE_PAY
                }),
            }

            // 签名
            data.sign = this.sign(data)

            return new ServiceResponse(200, {
                "Content-Type": "text/html; charset=utf-8"
            }, [
                `<form method="post" action="${Alipay.URL + "?" + Object.keys(data).map(n => n + "=" + encodeURIComponent(data[n])).join("&")}">`,
                `<input type="submit" style="display:none" />`,
                `</form>`,
                `<script>document.forms[0].submit();</script>`,
            ].join(""))
        }

        /**
         * 异步回调
         */
        public notify(data: { [name: string]: string; }) {
            console.log("notify from alipay", JSON.stringify(data))
            if (!this.verify(data)) {
                console.error("verify failed")
                throw new Error("verify failed")
            }
            return new ServiceResponse(200, {
                "Content-Type": "text/plain; charset=utf-8",
            }, "success")
        }

        private sign(data: { [name: string]: string; }): string {
            return Buffer.from(
                $native("crypto").createRsa().sign(
                    Object.keys(data).filter(n => n !== "sign" && data[n]).sort().map(n => n + "=" + data[n]).join("&"),
                    Alipay.MERCHANT_PRIVATE_KEY,
                    "sha256",
                )
            ).toString("base64")
        }

        private verify(data: { [name: string]: string; }): boolean {
            return $native("crypto").createRsa().verify(
                Object.keys(data).filter(n => !~["sign", "sign_type"].indexOf(n) && data[n]).sort().map(n => n + "=" + decodeURIComponent(data[n])).join("&"),
                Buffer.from(data.sign, "base64"),
                Alipay.ALIPAY_PUBLIC_KEY,
                "sha256",
            )
        }
    })
    ```

2. Start a test payment with url [`/service/alipay`](/service/alipay).
