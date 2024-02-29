# XSS

```typescript
//#node_modules/xss&type=module
export class XSS {
    public static ESCAPTION = {
        "<": "&lt;",
        ">": "&gt;"
    } as { [char: string]: string; };

    private tags: { [name: string]: string[]; };

    constructor() {
        this.tags = {
            a: ["href"],
            br: [],
            canvas: [],
            div: [],
            h1: [],
            img: ["src"],
            p: [],
            video: [],
            section: []
        };
    }

    public escape(input: string): string {
        input = input || "";

        let current = 0,
            output = "";

        while (current < input.length) {
            if (input[current] === "<") { // 标签开始
                let start = current++,
                    end = start,
                    quotation = "",
                    token = "",
                    tokens = ["<"];

                while (current < input.length) {
                    if (quotation === input[current]) { // 引号结束
                        tokens.push(token);
                        token = "";
                        quotation = "";
                        current++;
                        continue;
                    }
                    if (quotation.length === 0 && input[current] === "\"" || input[current] === "'") { // 引号开始
                        quotation = input[current++];
                        continue;
                    }
                    if (quotation.length > 0) { // 属性值
                        if (input[current] === "\\" && input[current + 1] === "\"") { // 转义的双引号
                            token += "\"";
                            current += 2;
                        } else {
                            token += XSS.ESCAPTION[input[current]] || input[current];
                            current++;
                        }
                        continue;
                    }

                    if (input[current] === "<") { // 异常开始标签
                        end = Math.max(current - 1, start);
                        break;
                    }
                    if (/[a-zA-Z0-9\-]/.test(input[current])) { // 标签名或属性名
                        token += input[current++];
                        continue;
                    }

                    if (token) {
                        tokens.push(token);
                        token = "";
                    }

                    if (input[current] === ">") { // 标签结束
                        tokens.push(">");
                        end = current++;
                        break;
                    }
                    if (input[current] !== " ") { // 空格字符
                        tokens.push(input[current]);
                    }
                    current++;
                }

                if (tokens[tokens.length - 1] === ">") { // 有效标签
                    let isEndTag = tokens[1] === "/",
                        tagName = (isEndTag ? tokens[2] : tokens[1]).toLowerCase();

                    this.onTag(tagName);

                    if (tagName in this.tags) {
                        if (isEndTag) {
                            output += `</${tagName}>`;
                            continue;
                        }

                        let t = [tagName];
                        for (let i = 2; i < tokens.length - 1;) {
                            if (tokens[i] === "/" && tokens[i + 1] === ">") {
                                t.push("/");
                                break;
                            }
                            if (tokens[i] === "=") { // 非法属性名
                                console.warn(`The attribute name "${tokens[i]}" is invalid.`);
                                i++;
                                continue;
                            }

                            let attributeName = tokens[i].toLowerCase(),
                                attributeValue: string = "";

                            if (tokens[i + 1] === "=") {
                                attributeValue = tokens[i + 2] || "";

                                if (attributeValue.length > 1 && attributeValue[0] === attributeValue[attributeValue.length - 1] && (attributeValue[0] === "'" || attributeValue[0] === "\"")) { // 去除首尾单双引号
                                    attributeValue = attributeValue.substring(1, attributeValue.length - 1);
                                }

                                i += 3;
                            } else {
                                i++;
                            }

                            this.onTagAttribute(tagName, attributeName, attributeValue);

                            if (this.tags[tagName].indexOf(attributeName) !== -1) {
                                t.push(attributeValue !== null ? `${attributeName}="${attributeValue}"` : attributeName);
                            }
                        }

                        output += `<${t.join(" ")}>`;

                        continue;
                    }
                }

                output += input.substring(start, end + 1).split("").map(c => XSS.ESCAPTION[c] || c).join("");

                continue;
            }

            output += XSS.ESCAPTION[input[current]] || input[current];
            current++;
        }

        return output;
    }

    public onTag(tagName: string) {
        console.debug(tagName);
    }

    public onTagAttribute(tagName: string, attributeName: string, attributeValue?: string) {
        console.debug(tagName, attributeName, attributeValue);
    }
}

export let xss = new XSS();
```

### Usage

```typescript
import { XSS } from "xss"

console.log(xss.escape(`<img src="http://127.0.0.1/a.png" a="" b=0 c= d = />`))
```
