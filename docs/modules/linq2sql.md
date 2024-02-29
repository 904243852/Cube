# Linq to SQL

```typescript
//#name=node_modules/linq2sql&type=module
const db = $native("db")

type SQLBuilder = {
    table: string
    wheres: string
    params: any[]
}

class Column<V> {
    private builder: SQLBuilder

    private name: string

    constructor(name: string, builder: SQLBuilder) {
        this.name = name
        this.builder = builder
    }

    private add(op: string, ...v: V[]): WherePredicate {
        this.builder.wheres += " and " + op
        this.builder.params.push(...v)
        return
    }

    public eq(v: V): WherePredicate {
        return this.add(this.name + " = ?", v)
    }

    public ne(v: V): WherePredicate {
        return this.add(this.name + " <> ?", v)
    }

    public gt(v: V): WherePredicate {
        return this.add(this.name + " > ?", v)
    }

    public ge(v: V): WherePredicate {
        return this.add(this.name + " >= ?", v)
    }

    public lt(v: V): WherePredicate {
        return this.add(this.name + " < ?", v)
    }

    public le(v: V): WherePredicate {
        return this.add(this.name + " <= ?", v)
    }

    public like(v: V): WherePredicate {
        return this.add(this.name + " like ?", v)
    }

    public unlike(v: V): WherePredicate {
        return this.add(this.name + " not like ?", v)
    }

    public in(v: V[]): WherePredicate {
        return this.add(this.name + " in (" + v.map(i => "?").join(", ") + ")", ...v)
    }

    public notin(v: V[]): WherePredicate {
        return this.add(this.name + " not in (" + v.map(i => "?").join(", ") + ")", ...v)
    }

    public isnull(): WherePredicate {
        return this.add(this.name + " is null")
    }

    public notnull(): WherePredicate {
        return this.add(this.name + " is not null")
    }
}

type Table<T> = {
    [alias in keyof T]: Column<T[alias]>
}

type WherePredicate = {
}

export class Dataset<T> {
    private builder: SQLBuilder

    private static readonly _ = new Object()

    private constructor(c: new () => T) {
        this.builder = {
            table: <string>c["name"],
            wheres: "1 = 1",
            params: [],
        }
    }

    public static from<T>(c: new () => T): Dataset<T> {
        return new Dataset<T>(c)
    }

    public where(e: (t: Table<T>) => WherePredicate): Omit<Dataset<T>, "insert"> {
        const builder = this.builder

        e.apply(this, [new Proxy(Dataset._, {
            get(_, property) {
                return new Column(property, builder)
            },
        })])

        return this
    }

    public select<E>(e: (t: T) => E): E[] {
        const columns = []

        e.apply(this, [new Proxy(Dataset._, {
            get(_, property) {
                columns.push(property)
                return null
            },
        })])

        return this.invoke("select", `select ${columns.join(", ")} from ${this.builder.table} where ${this.builder.wheres}`, this.builder.params).map(i => e.apply(this, [i]))
    }

    public count(): number {
        return Number(this.invoke("count", `select count(1) count from ${this.builder.table} where ${this.builder.wheres}`, this.builder.params).pop()["count"])
    }

    public update(t: Partial<T>): void {
        const columns = Object.keys(t)
        this.invoke("update", `update ${this.builder.table} set ${columns.map(c => c + " = ?").join(", ")} where ${this.builder.wheres}`, [...columns.map(c => t[c]), ...this.builder.params])
    }

    public delete(): void {
        this.invoke("delete", `delete from ${this.builder.table} where ${this.builder.wheres}`, this.builder.params)
    }

    public insert(t: T) {
        const columns = Object.keys(t)
        return this.invoke("insert", `insert into ${this.builder.table}(${columns.join(", ")}) values(${columns.map(_ => "?").join(", ")})`, columns.map(c => t[c]))
    }

    private invoke(op: string, stmt: string, params: object[]): object[] {
        if (op == "select" || op == "count") {
            return db.query(stmt, ...params)
        }
        db.exec(stmt, ...params)
        if (op == "insert") {
            return [db.query("select LAST_INSERT_ROWID() ROWID").pop().ROWID]
        }
        return []
    }
}
```

### Usage

```typescript
import { Dataset } from "linq2sql"

$native("db").exec(`
    create table if not exists user (
        id integer primary key autoincrement,
        name text
    );
    insert or replace into user(id, name)
    values
        (1, 'zhangsan'),
        (2, 'lisi'),
        (3, 'wangwu');
`)

class user {
    id: number
    name: string
}

export default function (ctx) {
    const input: Partial<user> = {
        name: "zhang",
    }
    
    Dataset.from(user)
        .where(u => u.id.eq(1))
        .update({ name: "zhangsan@" + new Date().getTime() })

    return Dataset.from(user)
        .where(u => input.name && u.name.like(input.name + "%") || u.name.notnull())
        .select(u => { return { Id: u.id, Name: u.name, } })
}
```
