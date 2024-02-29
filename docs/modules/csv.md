# CSV

```typescript
//#name=node_modules/csv&type=module
export class CSV {
    public static toJsonArray(content: string): object[] {
        let options = {
            separator: content.match(/(\t|,|;)/)?.pop() || ",",
            linebreak: content.match(/(\n|\r\n)/)?.pop() || "\n"
        };

        content += options.linebreak; // 结尾补充一个换行符，用以兼容 "id,name\n1,zhang san" 场景

        let current = 0,
            tokens: string[] = [];

        while (current < content.length) {
            if (content.substr(current, options.linebreak.length) === options.linebreak) { // 如果是换行符
                tokens.push(options.linebreak);
                current += options.linebreak.length;
                continue;
            }

            if (content[current] === options.separator && content[current + 1] === options.separator) { // 如果当前字符和下一个字符都是分隔符，需要补充一个空值
                tokens.push(<string><unknown>null);
            }

            if (content[current] !== options.separator && content.substr(current, options.linebreak.length) !== options.linebreak) { // 如果当前字符既不是换行符，也不是分隔符
                let value = "";

                let escaped = content[current] === "\"",
                    quotes = [];

                while (
                    content[current] && (
                        quotes.length != 0 // 引号栈中的所有引号需要全部闭合
                        || content[current] !== options.separator && content.substr(current, options.linebreak.length) !== options.linebreak
                    )
                ) {
                    if (escaped && content[current] === "\"") {
                        if (quotes.length == 0 || quotes[quotes.length - 1] !== content[current]) {
                            quotes.push(content[current]);
                        } else {
                            quotes.pop();
                        }
                    }
                    value += content[current];
                    current += 1;
                }

                if (value.length > 1 && value[0] === value[value.length - 1] && (value[0] === "\"")) {
                    value = value.substr(1, value.length - 2); // 首尾半角双引号需要去除
                }

                value = value.replace(/""/g, "\""); // 还原已转义的半角双引号 `""` 为 `"`

                tokens.push(value);

                continue;
            }

            current++;
        }

        let rows: string[][] = [],
            row_tokens: string[] = [];

        for (let token of tokens) {
            if (token !== options.linebreak) {
                row_tokens.push(token);
            } else if (row_tokens.length != 0) {
                rows.push(row_tokens);
                row_tokens = [];
            }
        }

        let headers = rows.shift(); // 取第一列为列名

        return rows.map(ts => {
            let columns = Math.max(headers.length, ts.length),
                obj = {};
            for (let i = 0; i < columns; i++) {
                let value = i < ts.length ? ts[i] : null;
                if (headers[i]) {
                    obj[headers[i]] = value;
                } else {
                    obj["unknown" + (i - headers.length)] = value; // 如果列名不存在，属性名取 unknown0、unknown1...
                }
            }
            return obj;
        });
    }

    public static toCSV(objs: object[], options?: { separator?: "," | "\t" | ";"; linebreak?: "\n" | "\r\n"; }): string {
        options = {
            separator: ",", // 默认以逗号分隔
            linebreak: "\n",
            ...options
        };

        let headers: { [name: string]: number; } = {},
            column = 0,
            header_tokens = [];

        for (let obj of objs) {
            for (let field in obj) {
                if (headers[field] == null) {
                    headers[field] = column++;
                    if (field.indexOf("\"") != -1) {
                        field = field.replace("\"", "\"\"");
                    }
                    if (field.indexOf(options.separator) != -1) {
                        field = "\"" + field + "\"";
                    }
                    header_tokens.push(field);
                }
            }
        }

        let content = header_tokens.join(options.separator) + "\n"; // 写入列名，即第一行

        for (let obj of objs) {
            let row_tokens = new Array(header_tokens.length);

            for (let field in obj) {
                let token: string = obj[field] + "";
                if (token.indexOf("\"") != -1) {
                    token = token.replace("\"", "\"\"");
                }
                if (token.indexOf(options.separator) != -1) {
                    token = "\"" + token + "\"";
                }
                row_tokens[headers[field]] = token;
            }

            content += row_tokens.join(options.separator) + "\n";
        }

        return content;
    }
    
    public static toSimpleCSV(objs: object[], headers: string[]): string {
        return headers.join(",") + "\n" + objs.map(o => {
            const tokens = [];
            for (const h of headers) {
                let v = (o[h] != null && o[h] != undefined) ? o[h] + "" : "",
                    n = false; // 是否需要引号
                if (v.indexOf("\"") != -1) {
                    v = v.replaceAll("\"", "\"\"");
                    n = true;
                }
                if (v.indexOf("\n") != -1) {
                    n = true;
                }
                if (v.indexOf(",") != -1) {
                    n = true;
                }
                if (n) {
                    v = "\"" + v + "\"";
                }
                tokens.push(v);
            }
            return tokens.join(",");
        }).join("\n");
    }
}
```

### Usage

```typescript
import { CSV } from "csv"

export default function (ctx) {
    return CSV.toJsonArray(`id,name,age\n1,zhangsan,19\n2,"li,si",20`)
}
```
