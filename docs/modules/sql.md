# SQL

```typescript
//#name=node_modules/sql&type=module
export function sql(stmt: string): any {
    return function () {
        return function (this: any, ...params: any[]) {
            return $native("db").query(stmt, ...params)
        }
    }
}

export function sqlc(tpl: string): any {
    return function () {
        return function (this: any, ...args: any[]) {
            // 校验入参必须为一个 object 对象
            if (args.length !== 1 || typeof args[0] !== "object" || !args[0]) {
                throw new Error("args must be a object")
            }
            const condition = args[0] as object

            // 删除换行符
            tpl = tpl.replace(/\n/g, "")

            // 解析 if 标签
            const pattern = /<if\s+test=['"]?(\w+)['"]?>(.*)<\/if>/ig
            let result = null
            while ((result = pattern.exec(tpl)) !== null) {
                if (condition[result[1]]) {
                    tpl = tpl.replace(result[0], result[2])
                } else {
                    tpl = tpl.replace(result[0], "")
                }
            }

            // 解析变量
            const params = [],
                stmt = tpl.replace(/#{(\w+)}#/g, function ($0, $1) {
                    const p = condition[$1]
                    if (Array.isArray(p)) { // 如果是数组类型，转换成 (?, ?, ?...) 子句
                        if (!p.length) {
                            return "(null)" // "where ... in (null)" 避免数组为空的时候出现 sql 语法错误，同时该语句也不会查询出值为 null 的数据
                        }
                        params.push(...p)
                        return "(" + p.map(i => "?").join(",") + ")"
                    }
                    params.push(p)
                    return "?"
                })

            // 执行语句
            return $native("db").query(stmt, ...params)
        }
    }
}
```

### Usage

```typescript
import { sql, sqlc, } from "sql"

class SourceMapper {
    @sql("select * from source where rowid = ?")
    public queryById(id: number): any[] { return }

    @sqlc(`
        select rowid,
            name,
            type
        from source
        where 1 = 1
        <if test="name">
            and name like #{name}#
        </if>
    `)
    public queryByCondition(condition: object): any[] { return }

    @sqlc("select rowid, name, type from source where rowid in #{ids}#")
    public queryByIds(condition: object): any[] { return }
}

console.log(new SourceMapper().queryById(1))

console.log(new SourceMapper().queryByCondition({
    name: "greet%",
}))

console.log(new SourceMapper().queryByIds({
    ids: [1, 3],
}))
```