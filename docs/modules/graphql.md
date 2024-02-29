# GraphQL

```typescript
//#name=node_modules/graphql&type=module
//#region record

export type Record = {
    [name: string]: any;
}

//#endregion

//#region schema

export type Schema = {
    [name: string]: TableSchema;
}

type SubSchema = {
    [name: string]: SubTableSchema | ColumnSchema;
}

type TableSchema = {
    table: string;
    properties: SubSchema;
}

type SubTableSchema = TableSchema & {
    joined: {
        column: string;
        parent: string;
        isCollection?: false;
    };
}

type ColumnSchema = {
    column: string;
    isPrimaryKey?: true;
}

//#endregion

//#region query

export type QueryDSL = {
    [name: string]: TableQueryDSL | Record | null;
}

type TableQueryDSL = {
    properties: QueryDSL | null;
    conditions?: ConditionQueryDSL[];
    options?: {
        skip?: number;
        limit?: number;
    };
}

type ConditionQueryDSL = {
    field: string;
    operator: "eq" | "gt" | "ge" | "lt" | "le" | "ne";
    value: object;
} | {
    field: string;
    operator: "in";
    value: object[];
}

type ExtendedQueryDSL = {
    [name: string]: {
        foreignKeys?: string[];
        conditions?: {
            field: string;
            operator: "in";
            value: object[];
        }[];
    }
}

//#endregion

//#region mutation

export type MutationDSL = {
    [name: string]: Record | MutationDSL;
}

type MutationRequestData = {
    /**
     * 请求对象
     */
    record: Record;
    /**
     * 数据库对象
     */
    dataset?: Record;
}

type MutationRequests = {
    [name: string]: {
        /**
         * 对象集合
         */
        data: MutationRequestData[];
        /**
         * 前置子请求
         */
        subrequests0?: MutationRequests;
        /**
         * 后置子请求
         */
        subrequests1?: MutationRequests;
    }
};

//#endregion

export abstract class GraphQL {
    private schema: Schema;

    constructor(schema: Schema) {
        this.schema = schema;
    }

    //#region query

    public query(dsl: QueryDSL) {
        return this.doQuery(dsl, <SubSchema>this.schema, null);
    }

    private doQuery(dsl: QueryDSL, schema: SubSchema, exdsl?: ExtendedQueryDSL) {
        return Object.keys(dsl).reduce((result: Record, name: string) => {
            // 查询表模型
            const model = schema[name];
            if (!model) {
                throw new Error(`can not find ${name} in schema`);
            }
            if (!("table" in model)) {
                throw new Error(`${name} is not a table schema`);
            }

            // 初始化请求
            const request = this.toQueryRequest(dsl, name, model, exdsl);

            // 构建查询列集合、子请求
            const { selects, subrequests } = this.toSqlSelectClauses(request.properties, request.foreignKeys, model);

            // 构建查询条件集合
            const { wheres, params } = this.toSqlWhereClauses(<ConditionQueryDSL[]>request.conditions);

            // 查询
            const datasets = this.onSelect(`select ${Object.keys(selects).join(", ")} from ${model.table} where ${["1 = 1"].concat(wheres).join(" and ")} limit ${request.options.skip},${request.options.limit}`, { params }),
                records = this.toQueryRecords(datasets, model, { selects, subrequests });

            result[name] = schema == this.schema
                ? records.map(r => r.data) // 隐藏根元素的 keys 记录
                : records;
            return result;
        }, {});
    }

    private toQueryRequest(dsl: QueryDSL, name: string, model: SubTableSchema, exdsl: ExtendedQueryDSL) {
        const tdsl = this.toTableQueryDSL(dsl, name, model);

        return {
            properties: tdsl.properties || model.properties,
            options: {
                skip: 0,
                limit: 5000,
                ...tdsl.options,
            },
            conditions: (tdsl.conditions || [])
                .map(({ field, operator, value }) => {
                    // 校验 field
                    if (!field) {
                        throw new Error(`field of condition can not be null or empty`);
                    }
                    const column = model.properties[field];
                    if (!column) {
                        throw new Error(`field ${field} does not exist`);
                    }
                    if (!("column" in column)) {
                        throw new Error(`field ${field} is not a column`);
                    }
                    return {
                        field: column.column,
                        operator,
                        value,
                    };
                })
                .concat(exdsl?.[name]?.conditions || []),
            foreignKeys: exdsl?.[name]?.foreignKeys || [],
        };
    }

    private toTableQueryDSL(dsl: QueryDSL, name: string, model: SubTableSchema): TableQueryDSL {
        const v = dsl[name];

        // 如果请求的 properties 为 null，表示默认查询所有字段
        if (!v) {
            return {
                properties: model.properties,
            };
        }

        // 若 dsl 没有 properties 属性，则将 dsl 作为 record 查询
        if (!("properties" in v)) {
            return {
                properties: v,
                conditions: Object.keys(v)
                    .reduce((p, c) => {
                        // 如果 record 的属性值不为 null，则将该属性值作为查询条件
                        if (v[c] !== null && typeof v[c] !== "object") {
                            p.push({ field: c, operator: "eq", value: <object>v[c] });
                        }
                        return p;
                    }, [] as ConditionQueryDSL[]),
            };
        }

        return <TableQueryDSL>v;
    }

    private toSqlSelectClauses(properties: QueryDSL, foreignKeys: string[], model: SubTableSchema) {
        // 初始化子请求
        const subrequests = <QueryDSL>{};

        const selects = {} as { [name: string]: { alias?: string; display?: boolean; isForeignKey?: true; } };

        // 处理外键所在的列
        for (const n of foreignKeys) {
            selects[n] = { display: false, isForeignKey: true };
        }

        // 处理属性所在的列
        for (const n in properties) {
            const property = model.properties[n];

            if (!property) {
                throw new Error(`property ${n} does not exist`);
            }
            if ("table" in property) { // 如果当前属性是子模型
                const p = property.joined.parent;
                selects[p] = selects[p] || { display: false };
                subrequests[n] = properties[n];
            } else { // 否则当前属性是列
                const c = property.column;
                selects[c] = {
                    ...selects[c],
                    alias: n,
                    display: true,
                };
            }
        }

        return {
            selects,
            subrequests,
        };
    }

    private toSqlWhereClauses(conditions: ConditionQueryDSL[]) {
        const wheres: string[] = [],
            params: object[] = [];

        for (const condition of conditions) {
            if (condition.value == null) {
                throw new Error(`value of condition ${condition.field} can not be null`);
            }

            const operator: string = { eq: "=", gt: ">", ge: ">=", lt: "<", le: "<=", ne: "<>" }[<string>condition.operator] || condition.operator;
            switch (operator) {
                case "in":
                    const values = (<object[]>condition.value).filter(v => v != null); // 去除 null 值
                    if (values.length) {
                        wheres.push(`${condition.field} ${operator} (${values.map(_ => "?").join(", ")})`);
                        params.push(...values);
                    }
                    break;
                case "=":
                case ">":
                case ">=":
                case "<":
                case "<=":
                case "<>":
                    wheres.push(`${condition.field} ${operator} ?`);
                    params.push(condition.value);
                    break;
                default:
                    throw new Error(`operator ${condition.operator} is not supported`);
            }
        }

        return {
            wheres,
            params,
        };
    }

    private toQueryRecords(datasets: object[], model: SubTableSchema, { selects, subrequests }: ReturnType<typeof this.toSqlSelectClauses>) {
        if (!datasets.length) {
            return [];
        }

        const records = [];

        // 遍历子请求，构造外键和外键关联条件
        const subexrequests: ExtendedQueryDSL = {};
        for (const n in subrequests) {
            const { column: c, parent: p } = (<SubTableSchema>model.properties[n]).joined;
            subexrequests[n] = {
                foreignKeys: [c],
                conditions: [{
                    field: c,
                    operator: "in",
                    value: this.uniq(datasets.map(d => <object>d[p])),
                }]
            };
        }

        // 如果有子属性请求，查询子对象
        const subdatasets = this.doQuery(subrequests, model.properties, subexrequests) as { [name: string]: { data: object, keys?: Record; }[] };

        // 先遍历所有子对象，按照外键分类，便于后续回写子对象值
        const subdatasetsmap: { [name: string]: { [key: string]: object[] } } = {};
        for (const n in subdatasets) {
            const { column: c } = (model.properties[n] as SubTableSchema).joined;
            subdatasetsmap[n] = {};
            for (const subdataset of subdatasets[n]) {
                if (!subdataset.keys?.[c]) {
                    throw new Error(`property ${c} in sub hide dataset does not exist`);
                }
                subdatasetsmap[n][subdataset.keys[c]] = subdatasetsmap[n][subdataset.keys[c]] || [];
                subdatasetsmap[n][subdataset.keys[c]].push(subdataset.data);
            }
        }

        // 遍历对象，回写属性值
        for (const dataset of datasets) {
            const data = {} as Record, // 数据（需要随最终结果返回）
                keys = {} as Record; // 键值（不需要返回，用于外键关联）

            for (const n in selects) {
                if (selects[n].display) { // 如果需要作为结果返回，属性名称需要重命名
                    data[selects[n].alias || n] = <object>dataset[n];
                }
                if (selects[n].isForeignKey) { // 如果是外键，则将属性放入不可见属性集中
                    keys[n] = <object>dataset[n];
                }
            }

            for (const n in subdatasetsmap) {
                const submodel = model.properties[n] as SubTableSchema;

                // 根据外键过滤子对象集合，并赋值给父对象
                dataset[n] = subdatasetsmap[n][dataset[submodel.joined.parent]];

                if (submodel.joined.isCollection === false) { // 如果子模型非集合，则返回第一个子对象
                    dataset[n] = (<object[]>dataset[n])?.pop();
                }

                data[n] = <unknown>dataset[n]; // 子模型需要作为结果返回
            }

            records.push({ data, keys });
        }

        return records;
    }

    //#endregion

    //#region mutate

    public mutate(dsl: MutationDSL) {
        const requests: MutationRequests = {};

        // 构造请求
        this.toMutationRequests(dsl, <SubSchema>this.schema, requests);

        // 执行请求
        this.doMutateRequests(requests, <SubSchema>this.schema);

        return dsl;
    }

    private toMutationRequests(dsl: MutationDSL, schema: SubSchema, requests: MutationRequests) {
        for (const name in dsl) {
            // 查询表模型
            const model = schema[name];
            if (!model) {
                throw new Error(`${name} in schema does not exist`);
            }
            if (!("table" in model)) {
                throw new Error(`${name} is not a table schema`);
            }

            // 校验请求
            if (!dsl[name]) {
                throw new Error(`request ${name} can not be null`);
            }
            if (typeof (dsl[name]) !== "object") {
                throw new Error(`request ${name} must be a object or collection`);
            }

            // 查询当前模型中保存了子模型主键的属性名称
            const propertiesOfSubPrimaryKey = Object.keys(model.properties)
                .filter(n => {
                    const property = model.properties[n];
                    if ("column" in property) { // 如果是列属性，跳过不处理
                        return false;
                    }
                    if ((<ColumnSchema>property.properties[Object.keys(property.properties).filter(sn => (<ColumnSchema>property.properties[sn]).column === property.joined.column).pop() || ""])?.isPrimaryKey !== true) {
                        return false;
                    }
                    if (property.joined.isCollection !== false) { // 前置子请求必须为非集合元素
                        throw new Error(`subschema ${n} can not be a collection for the foreign key ${property.joined.parent} is a primary key of subschema`);
                    }
                    return true;
                });

            const records = dsl[name] instanceof Array ? <Record[]>dsl[name] : [dsl[name]],
                data: MutationRequestData[] = requests[name]?.data || [], // 对象集合
                subrequests0: MutationRequests = {}, // 前置子请求，需要优先入库以生成主键
                subrequests1: MutationRequests = {}; // 后置子请求，需要依赖当前请求的主键入库

            for (const record of records) {
                if (!record) {// 原始请求对象
                    throw new Error("request data can not be null");
                }

                const dataset = {}; // 数据库对象

                for (const subname in record) {
                    const property = model.properties[subname];
                    if (!property) {
                        throw new Error(`property ${subname} in schema ${name} does not exist`);
                    }
                    if (!record[subname]) {
                        continue;
                    }

                    // 如果该属性是列，当前的请求对象属性写入到数据库对象属性
                    if ("column" in property) {
                        dataset[property.column] = <unknown>record[subname];
                        continue;
                    }

                    if ((property.joined.isCollection !== false) !== (record[subname] instanceof Array)) { // isCollection 为 true 或 null 均表示集合类型
                        throw new Error(`subrequest ${subname} ${property.joined.isCollection ? "must" : "can not"} be a collection`);
                    }

                    const subdsl = {
                        [subname]: <MutationDSL>record[subname]
                    };

                    // 缓存前置子请求
                    if (!!~(propertiesOfSubPrimaryKey.indexOf(subname))) { // 如果该属性是子模型，并且子模型的主键关联父模型，则需要先保存子对象以生成子对象主键
                        this.toMutationRequests(subdsl, model.properties, subrequests0);
                        continue;
                    }

                    // 缓存后置子请求
                    this.toMutationRequests(subdsl, model.properties, subrequests1);
                }

                data.push({
                    dataset,
                    record,
                });
            }

            requests[name] = {
                data,
                subrequests0,
                subrequests1,
            };
        }
    }

    private doMutateRequests(requests: MutationRequests, schema: SubSchema) {
        for (const name in requests) {
            const model = <SubTableSchema>schema[name],
                { data, subrequests0, subrequests1 } = requests[name];

            // 保存前置子请求
            this.doMutateRequests(subrequests0, model.properties);
            for (const subname in subrequests0) {
                const submodel = <SubTableSchema>model.properties[subname],
                    primaryKey = Object.keys(submodel.properties).filter(i => (submodel.properties[i] as ColumnSchema).column === submodel.joined.column).pop() || "";
                // 遍历对象，将子对象以及子对象主键值回写至对象中
                for (const { record, dataset } of data) {
                    const subrecord = <Record>record[subname];
                    if (!subrecord) {
                        continue;
                    }
                    dataset[submodel.joined.parent] = <unknown>subrecord[primaryKey];
                }
            }

            // 保存请求并回写主键
            this.doMutationRequestData(data, name, model);

            // 遍历对象，将对象以及对象主键值回写至后置子请求中
            for (const subname in subrequests1) {
                const submodel = <SubTableSchema>model.properties[subname];
                for (const { record, dataset } of data) {
                    const s = subrequests1[subname].data
                        .filter(d => {
                            if (submodel.joined.isCollection === false) {
                                return d.record === record[subname];
                            } else {
                                return (<Record[]>record[subname]).some(i => d.record === i);
                            }
                        });
                    if (!s.length) {
                        continue;
                    }

                    if (!dataset[submodel.joined.parent]) {
                        throw new Error(`value of ${submodel.joined.parent} is required`);
                    }
                    s.forEach(o => {
                        o.dataset[submodel.joined.column] = <unknown>dataset[submodel.joined.parent]; // 将外键塞入子对象中
                    });
                }
            }

            // 保存后置子对象
            this.doMutateRequests(subrequests1, model.properties);
        }
    }

    private doMutationRequestData(data: MutationRequestData[], name: string, model: SubTableSchema) {
        if (!data.length) {
            return;
        }

        // 找出当前模型的主键
        const primaryKey = Object.keys(model.properties).filter(n => (model.properties[n] as ColumnSchema).isPrimaryKey === true).pop();
        if (!primaryKey) {
            throw new Error(`primary key of ${name} in schema does not exist`);
        }
        const primaryKeyColumn = <ColumnSchema>model.properties[primaryKey];

        // 将请求对象分类为需要新增的数据集和需要更新的数据集
        const data2insert: MutationRequestData[] = [],
            data2update: MutationRequestData[] = [];
        for (const d of data) {
            if (d.dataset?.[primaryKeyColumn.column]) {
                data2update.push(d);
            } else {
                data2insert.push(d);
            }
        }

        if (data2insert.length) {
            const ids = this.onInsert(model.table, data2insert.map(d => d.dataset));
            for (let i = 0; i < ids.length; i++) {
                // 回写 id 至原始请求对象和数据库对象中
                data2insert[i].record[primaryKey] = ids[i];
                if (data2insert[i].dataset) {
                    data2insert[i].dataset[primaryKeyColumn.column] = ids[i];
                }
            }
        }
        if (data2update.length) {
            this.onUpdate(model.table, data2update.map(d => d.dataset), primaryKeyColumn.column);
        }
    }

    //#endregion

    private uniq<T>(arr: T[]): T[] {
        return [...new Set(arr)];
    }

    protected abstract onSelect(stmt: string, options: { params: object[]; }): object[];

    protected abstract onInsert(table: string, datasets: object[]): string[];

    protected abstract onUpdate(table: string, datasets: object[], primaryKey: string): void;
}

export class MyGraphQL extends GraphQL {
    private db = $native("db");

    constructor(schema: Schema, initsql?: string) {
        super(schema);
        this.db.exec(initsql);
    }

    protected onSelect(stmt: string, options: { params: any[]; }): object[] {
        return this.db.query(stmt, ...options.params);
    }

    protected onInsert(table: string, datasets: Record[]): string[] {
        const that = this;
        return datasets.map(d => {
            that.db.exec(`insert into ${table} (${Object.keys(d).join(", ")}) values (${Object.keys(d).map(n => "?").join(", ")})`, ...Object.keys(d).map(i => d[i]));
            return that.db.query("select LAST_INSERT_ROWID() ROWID").pop().ROWID;
        });
    }

    protected onUpdate(table: string, datasets: Record[], primaryKey: string): void {
        datasets.forEach(d => {
            this.db.exec(`update ${table} set ${Object.keys(d).filter(n => n !== primaryKey).map(n => `${n} = ?`).join(", ")} where ${primaryKey} = ?`, ...Object.keys(d).filter(n => n !== primaryKey).map(i => d[i]).concat([d[primaryKey]]));
        });
    }
}
```

### Usage

```typescript
import { MyGraphQL } from "graphql"

const G = new MyGraphQL({
    offering: {
        table: "Offering",
        properties: {
            spu: {
                column: "Id",
                isPrimaryKey: true
            },
            name: {
                table: "I18n",
                properties: {
                    id: {
                        column: "Id",
                        isPrimaryKey: true
                    },
                    zh: {
                        column: "Zh"
                    },
                    en: {
                        column: "En"
                    }
                },
                joined: {
                    column: "Id",
                    parent: "Name",
                    isCollection: false
                }
            },
            description: {
                column: "Description"
            },
            product: {
                table: "Product",
                properties: {
                    sku: {
                        column: "Id",
                        isPrimaryKey: true
                    },
                    price: {
                        column: "Price"
                    },
                    stock: {
                        column: "Stock"
                    },
                    attribute: {
                        table: "ProductAttribute",
                        properties: {
                            id: {
                                column: "Id",
                                isPrimaryKey: true
                            },
                            code: {
                                column: "Code"
                            },
                            value: {
                                column: "Value"
                            },
                            type: {
                                column: "Type"
                            }
                        },
                        joined: {
                            column: "ProductId",
                            parent: "Id"
                        }
                    }
                },
                joined: {
                    column: "OfferingId",
                    parent: "Id"
                }
            }
        }
    }
}, `
    create table if not exists Offering (
        Id integer primary key autoincrement,
        Name text,
        Description text
    );
    create table if not exists Product (
        Id integer primary key autoincrement,
        Price text,
        Stock text,
        OfferingId text
    );
    create table if not exists ProductAttribute (
        Id integer primary key autoincrement,
        Code text,
        Value text,
        Type text,
        ProductId text
    );
    create table if not exists I18n (
        Id integer primary key autoincrement,
        Zh text,
        En text
    );
`);

export default function (ctx) {
    console.info("Mutate:", JSON.stringify(
        // 保存记录
        G.mutate({
            offering: {
                name: {
                    zh: "球",
                    en: "ball"
                },
                description: "this is a ball",
                product: [{
                    price: 2,
                    attribute: [{
                        code: "color",
                        value: "red"
                    }]
                }, {
                    price: 2.5,
                    attribute: [{
                        code: "color",
                        value: "green"
                    }]
                }]
            }
        }), null, "    "
    ));
    
    console.info("Query:", JSON.stringify(
        // 查询记录
        G.query({
            offering: {
                properties: null,
                conditions: [{ // 查询条件
                    field: "spu",
                    operator: "eq",
                    value: 1
                }],
                options: { // 分页查询
                    limit: 1,
                    skip: 0
                }
            }
        }), null, "    "
    ));
    console.info("Query:", JSON.stringify(
        // 查询记录
        G.query({
            offering: {
                name: null,
                description: "this is a ball", // 如果属性不为 null，则表示该属性作为查询条件
                product: null
            }
        }), null, "    "
    ));
    return null
}
```