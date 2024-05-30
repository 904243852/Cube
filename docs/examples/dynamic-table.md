# Dynamic table

```typescript
enum E {
    List = 0x1 << 0,
    Detail = 0x1 << 1,
    Add = 0x1 << 2,
    Update = 0x1 << 3,
    Searchable = 0x1 << 4,
}

type Field = {
    name: string;
    label?: string;
    type: "string" | "number" | "enum" | "bool" | "date" | "datetime";
    mode: number;
}

type Options = {
    [name: string]: Function | object;
}

export abstract class DynamicElement {
    private static readonly Template = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <link rel="stylesheet" href="https://cdn.bootcdn.net/ajax/libs/element-plus/2.4.3/theme-chalk/index.min.css" />
    <script src="https://cdn.bootcdn.net/ajax/libs/vue/3.3.4/vue.global.prod.min.js"></script>
    <script src="https://cdn.bootcdn.net/ajax/libs/element-plus/2.4.3/index.full.min.js"></script>
    <title></title>
    <style>
        html, body {
            height: 100%;
            margin: 0;
            background-color: #f0f2f5;
        }
        .el-table {
            border-top: 1px solid #dcdfe6;
        }
        .el-pagination {
            flex: auto;
            margin-top: 13px;
        }
        .el-pagination .is-first {
            flex: auto;
        }
    </style>
</head>
<body>
    <div id="app" v-cloak style="height: 100%; padding: 32px; position: relative;">
        <el-card>
            <el-row style="padding-bottom: 10px;">
                <el-button @click="onDialogNew">New</el-button>
                <el-button-group style="padding-left: 5px;">
                    <el-button @click="onTableExport">Export</el-button>
                </el-button-group>
                <div style="margin-left: auto; display: inline-flex;">
                    <el-input v-model="table.search.keyword" placeholder="Enter keyword here" clearable @change="onTableFetch(true)">
                    </el-input>
                </div>
            </el-row>
            <el-row>
                <el-table v-loading="table.loading" :data="table.records" stripe table-layout="fixed">
<%
    for (const field of fields) {
        if (!(field.mode & E.List)) {
            continue
        }
%>
                    <el-table-column label="<%=field.label%>">
                        <template #default="scope">
                            {{ scope.row.<%=field.name%> }}
                        </template>
                    </el-table-column>
<%
    }
%>
                    <el-table-column label="Operation">
                        <template #default="scope">
                            <el-button link type="primary" @click="onTableRowEdit(scope.row.id)">Edit</el-button>
                            <el-button link type="danger" @click="onTableRowDelete(scope.row.id)">Delete</el-button>
                        </template>
                    </el-table-column>
                </el-table>
                <el-pagination @size-change="onTablePageSizeChange" @current-change="onTablePageCurrentChange" :current-page="table.pagination.index" :page-sizes="[5, 10, 20, 50]" :page-size="table.pagination.size" layout="total, sizes, prev, pager, next, jumper" :total="table.pagination.count">
                </el-pagination>
            </el-row>
        </el-card>
        <el-dialog v-model="dialog.visible">
            <el-form ref="FormRef" :model="dialog.record" label-position="right" label-width="80px">
<%
    for (const field of fields) {
        if (!(field.mode & E.Detail)) {
            continue
        }
%>
                <el-form-item label="<%=field.label%>" prop="<%=field.name%>">
                    <el-input v-model="dialog.record.<%=field.name%>" :disabled="!(dialog.record.id ? <%=field.mode & E.Update%> : <%=field.mode & E.Add%>)"></el-input>
                </el-form-item>
<%
    }
%>
                <el-form-item>
                    <el-button type="primary" :loading="dialog.loading" @click="onDialogSubmit(FormRef)">Submit</el-button>
                    <el-button @click="onDialogCancel(FormRef)">Cancel</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>
    </div>
    <script>
        const { ElMessage, ElMessageBox, } = ElementPlus
        Vue.createApp({
            setup() {
                return {
                    FormRef: Vue.ref(),
                }
            },
            data() {
                return {
                    table: {
                        records: [],
                        pagination: {
                            index: 1,
                            size: 5,
                            count: 0,
                        },
                        search: {
                            keyword: "",
                        },
                        loading: false,
                    },
                    dialog: {
                        record: {},
                        loading: false,
                    },
                }
            },
            methods: {
                onTableFetch(reset) {
                    const that = this
                    if (reset) {
                        that.table.pagination.index = 1
                    }
                    that.table.loading = true
                    fetch("?keyword=" + that.table.search.keyword + "&from=" + ((that.table.pagination.index - 1) * that.table.pagination.size) + "&size=" + that.table.pagination.size, {
                        method: "GET",
                    }).then(r => {
                        if (r.status != 200) {
                            throw new Error(r.statusText)
                        }
                        return r.json()
                    }).then(r => {
                        that.table.pagination.count = r.data.total
                        that.table.pagination.index = Math.min(that.table.pagination.index, Math.ceil(that.table.pagination.count / that.table.pagination.size))
                        that.table.records = r.data.records
                    }).catch(e => {
                        ElMessage.error(e.message)
                    }).finally(() => {
                        that.table.loading = false
                    })
                },
                onTablePageSizeChange(value) {
                    this.table.pagination.size = value
                    this.onTableFetch()
                },
                onTablePageCurrentChange(value) {
                    this.table.pagination.index = value
                    this.onTableFetch()
                },
                onTableRowEdit(id) {
                    const that = this
                    fetch("?id=" + id).then(r => r.json()).then(r => {
                        that.dialog.record = r.data
                        that.dialog.visible = true
                    })
                },
                onTableRowDelete(id) {
                    const that = this
                    ElMessageBox.confirm("This operation will permanently delete the record. Do you want to continue ?", "Tips", {
                        confirmButtonText: "Confirm",
                        cancelButtonText: "Cancel",
                        type: "warning",
                        beforeClose: (action, instance, done) => {
                            if (action === "confirm") {
                                instance.confirmButtonLoading = true
                                instance.confirmButtonText = "Delete..."
                                fetch("?id=" + id, {
                                    method: "DELETE",
                                }).then(r => r.json()).then(r => {
                                    if (r.code === "0") {
                                        ElMessage.success("Delete succeeded")
                                        that.onTableFetch()
                                    } else {
                                        ElMessage.error(r.message)
                                    }
                                    instance.confirmButtonLoading = false
                                })
                            }
                            done()
                        },
                    }).catch(() => { })
                },
                onDialogNew() {
                    this.dialog.record = {}
                    this.dialog.visible = true
                },
                onDialogSubmit(FormRef) {
                    const that = this
                    FormRef.validate(valid => {
                        if (!valid) {
                            return false
                        }
                        fetch(that.dialog.record.id ? "?id=" + that.dialog.record.id : "", {
                            method: that.dialog.record.id ? "POST" : "PUT",
                            body: JSON.stringify(that.dialog.record),
                        }).then(r => r.json()).then(r => {
                            if (r.code === "0") {
                                ElMessage.success("Submit succeeded")
                                that.dialog.visible = false
                                that.onTableFetch()
                            } else {
                                ElMessage.error(r.message)
                            }
                        })
                    })
                },
                onDialogCancel(FormRef) {
                    FormRef.resetFields()
                    this.dialog.visible = false
                },
            },
            mounted: function () {
                this.onTableFetch()
            },
        }).use(ElementPlus).mount("#app")
    </script>
</body>
</html>`

    protected name: string

    protected fields: Field[]

    protected options: Options

    constructor(name: string, fields: Field[], options?: Options) {
        this.name = name
        this.fields = fields
        this.options = options || {}
        this.init()
    }

    public render() {
        const pattern = /<%([\s\S]*?)%>/g // 匹配模板中 "<%...%>" 片段代码

        const codes = [`with(data) { let __tpl__ = "";`] // 需要动态执行的函数代码集合

        let match: RegExpExecArray = null, // 当前匹配的结果
            index = 0 // 当前匹配到的索引
        while (match = pattern.exec(DynamicElement.Template)) {
            // 保存当前匹配项之前的普通文本/占位
            codes.push(`__tpl__ += "${DynamicElement.Template.substring(index, match.index).replace(/\r?\n/g, "\\n").replace(/"/g, '\\"')}";`)

            // 保存当前匹配项
            if (match[1].substr(0, 1) == "=") { // 占位符
                codes.push(`__tpl__ += ${match[1].substr(1)};`)
            } else { // 逻辑代码
                codes.push(match[1])
            }

            // 更新当前匹配索引
            index = match.index + match[0].length
        }
        // 保存文本尾部
        codes.push(`__tpl__ += "${DynamicElement.Template.substring(index).replace(/\r?\n/g, "\\n").replace(/"/g, '\\"')}";`)
        codes.push("return __tpl__; }")

        const data = {
            E,
            fields: this.fields.map(e => {
                return {
                    label: e.name,
                    ...e,
                }
            }),
        }
        return new Function("data", codes.join("\n"))(data) // 渲染模板中 "<%...%>" 片段代码
            .replace(/{{(.+?)}}/g, function ($0: string, $1: string) { // 渲染模板中 "{{ ... }}" 片段代码
                return data[$1.trim()] || $0
            })
    }

    public invoke(name: string, ...params: any[]) {
        const option = this.options[name]
        if (!option) {
            throw new Error("no such option")
        }
        if (option instanceof Function) {
            return {
                output: option.apply(this.options, params),
            }
        }
        return {
            output: option,
        }
    }

    public abstract init(): void

    public abstract get(id: string, keyword: string, from: number, size: number): any;

    public abstract put(record: any): string

    public abstract post(id: string, record: any): void

    public abstract delete(id: string): void
}

const d = new class extends DynamicElement {
    private dbc = $native("db")

    public init() {
        this.dbc = $native("db")
        this.dbc.exec(`
            --drop table if exists ${this.name};
            create table if not exists ${this.name} (
                id integer primary key autoincrement,
                ${this.fields.map(f => {
            const type = { number: "integer" }[f.type] || "text"
            return `${f.name} ${type}`
        }).join(",\n")}
            );
        `)
    }

    public get(id: string, keyword: string, from: number, size: number): any {
        if (id) {
            const columns = ["id", ...this.fields.filter(f => f.mode & E.Detail).map(f => f.name)]
            return this.dbc.query(`select ${columns.join(", ")} from ${this.name} where id = ?`, id).pop()
        }
        const columns = ["id", ...this.fields.filter(f => f.mode & E.List).map(f => f.name)]
        let wheres = "",
            params = []
        if (keyword) {
            keyword = keyword.replaceAll("%", "\%").replaceAll("_", "\_")
            const conditions = this.fields.filter(f => f.mode & E.Searchable).map(f => f.name + " like ?")
            if (conditions.length) {
                wheres = " where " + conditions.join(" or ")
                params.push(...conditions.map(_ => "%" + keyword + "%"))
            }
        }
        return {
            total: this.dbc.query(`select count(1) total from ${this.name} ${wheres}`, ...params).pop().total,
            records: this.dbc.query(`select ${columns.join(", ")} from ${this.name} ${wheres} limit ${from || 0}, ${size || 10}`, ...params),
        }
    }

    public put(record: any): string {
        const columns = this.fields.filter(f => f.mode & E.Add).map(f => f.name)
        this.dbc.exec(`insert into ${this.name}(${columns.join(", ")}) values(${columns.map(f => "?").join(", ")})`, ...columns.map(c => record[c]))
        return this.dbc.query("select LAST_INSERT_ROWID() rowID").pop().rowID
    }

    public post(id: string, record: any): void {
        const columns = this.fields.filter(f => f.mode & E.Update && f.name in record).map(f => f.name)
        this.dbc.exec(`update ${this.name} set ${columns.map(c => c + " = ?").join(", ")} where id = ?`, ...columns.map(c => record[c]), id)
    }

    public delete(id: string): void {
        this.dbc.exec(`delete from ${this.name} where id = ?`, id)
    }
}(
    "user",
    [
        {
            name: "name",
            type: "string",
            mode: E.List | E.Detail | E.Add | E.Searchable,
        },
        {
            name: "age",
            type: "number",
            mode: E.Detail | E.Add | E.Update,
        },
        {
            name: "gender",
            type: "enum",
            mode: E.Detail | E.Add | E.List,
        },
        {
            name: "active",
            type: "bool",
            mode: E.Add | E.Detail | E.Update | E.List,
        },
        {
            name: "birthday",
            type: "date",
            mode: E.Add | E.Detail | E.List,
        },
        {
            name: "last_modify_time",
            type: "datetime",
            mode: E.Add | E.Detail,
        },
    ],
)

export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
    const params = new Proxy(ctx.getURL().params, {
        get(target: any, property: string) {
            return target[property]?.[0]
        }
    })
    switch (ctx.getMethod()) {
        case "GET":
            if (params.id || params.keyword || params.from || params.size) {
                return d.get(params.id, params.keyword, params.from, params.size)
            }
            return new ServiceResponse(
                200,
                {
                    "Content-Type": "text/html"
                },
                d.render(),
            )
        case "PUT":
            return d.put(ctx.getBody().toJson())
        case "POST":
            return d.post(params.id, ctx.getBody().toJson())
        case "DELETE":
            return d.delete(params.id)
        default:
            throw new Error("not implemented")
    }
}
```