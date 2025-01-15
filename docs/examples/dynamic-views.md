# Dynamic views

## Table

```typescript
//?name=DynamicTable&type=controller&url=dynamic/table/{name}
type Property = {
    name: string;
    label?: string;
    type: "string" | "integer" | "number" | "enum" | "boolean" | "date" | "datetime" | "code";
    mode: number;
}

enum PropertyMode {
    LIST = 0x1 << 0,
    DETAIL = 0x1 << 1,
    ADD = 0x1 << 2,
    UPDATE = 0x1 << 3,
    SEARCH = 0x1 << 4,
}

export abstract class DynamicTable {
    private static readonly TEMPLATE = `<!DOCTYPE html>
<html>

<head>
    <meta charset="UTF-8">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/element-plus/2.4.3/theme-chalk/index.min.css" />
    <script src="https://cdnjs.cloudflare.com/ajax/libs/vue/3.3.4/vue.global.prod.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/element-plus/2.4.3/index.full.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/element-plus-icons-vue/2.3.1/index.iife.min.js"></script>
    <title></title>
    <base target="_blank" /><!-- 网页中所有的超链接的目标地址都在新建窗口中打开 -->
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
                <el-button :icon="Plus" @click="onDialogNew">New</el-button>
                <div style="margin-left: auto; display: inline-flex;">
                    <el-input v-model="table.search.keyword" placeholder="Enter keyword here" clearable @change="onTableFetch(true)" :suffix-icon="Search">
                    </el-input>
                </div>
            </el-row>
            <el-row>
                <el-table v-loading="table.loading" :data="table.records" stripe show-overflow-tooltip @sort-change="onTableSortChange" table-layout="fixed">
                    <template v-for="property in schema.properties">
                        <el-table-column :label="property.label" v-if="property.mode & E.LIST">
                            <template #default="scope">
                                {{ scope.row[property.name] }}
                            </template>
                        </el-table-column>
                    </template>
                    <el-table-column label="Operation">
                        <template #default="scope">
                            <el-button link type="primary" @click="onTableRowEdit(scope.row)" :icon="Edit">
                            </el-button>
                            <el-button link type="danger" @click="onTableRowDelete(scope.row)" :icon="Delete">
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>
                <el-pagination @size-change="onTablePageSizeChange" @current-change="onTablePageCurrentChange" :current-page="table.pagination.index" :page-sizes="[10, 20, 50]" :page-size="table.pagination.size" layout="total, sizes, prev, pager, next, jumper" :total="table.pagination.count">
                </el-pagination>
            </el-row>
        </el-card>
        <el-dialog v-model="dialog.visible" :title="dialog.record.id ? 'Edit' : 'New'">
            <el-form ref="FormRef" :model="dialog.record" label-position="right" label-width="80px">
                <template v-for="property in schema.properties">
                    <el-form-item :label="property.label" :prop="property.name" v-if="property.mode & (E.DETAIL | (dialog.record.id ? E.UPDATE : E.ADD))">
                        <el-input-number v-model="dialog.record[property.name]" :disabled="!(property.mode & (dialog.record.id ? E.UPDATE : E.ADD))" v-if="property.type === 'integer'"></el-input-number>
                        <el-switch v-model="dialog.record[property.name]" :disabled="!(property.mode & (dialog.record.id ? E.UPDATE : E.ADD))" v-else-if="property.type === 'boolean'"></el-switch>
                        <el-date-picker v-model="dialog.record[property.name]" :type="property.type" :disabled="!(property.mode & (dialog.record.id ? E.UPDATE : E.ADD))" v-else-if="property.type === 'date' || property.type === 'datetime'"></el-date-picker>
                        <monaco-editor v-model="dialog.record[property.name]" :readonly="!(property.mode & (dialog.record.id ? E.UPDATE : E.ADD))" v-else-if="property.type === 'code'" language="json" height="200px" style="border: 1px solid #e5e7eb;"></monaco-editor>
                        <el-input v-model="dialog.record[property.name]" :disabled="!(property.mode & (dialog.record.id ? E.UPDATE : E.ADD))" v-else></el-input>
                    </el-form-item>
                </template>
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
                const { ref } = Vue
                const { Delete, Edit, Search, Plus } = ElementPlusIconsVue
                return {
                    Delete,
                    Edit,
                    Search,
                    Plus,
                    FormRef: ref(),
                }
            },
            data() {
                return {
                    E: ${JSON.stringify(PropertyMode)},
                    button: {
                        upload: {
                            loading: false,
                        },
                    },
                    table: {
                        records: [],
                        pagination: {
                            index: 1,
                            size: 10,
                            count: 0,
                        },
                        search: {
                            keyword: "",
                        },
                        sort: {
                            prop: "id",
                            order: "desc",
                        },
                        loading: false,
                    },
                    dialog: {
                        record: {},
                        visiable: false,
                        loading: false,
                    },
                    schema: {
                        properties: <%=properties%>,
                    },
                }
            },
            methods: {
                onTableFetch(reset) {
                    if (reset) {
                        this.table.pagination.index = 1
                    }
                    this.table.loading = true
                    fetch("?keyword=%25" + this.table.search.keyword + "%25" + "&from=" + (this.table.pagination.index - 1) * this.table.pagination.size + "&size=" + this.table.pagination.size + "&sort=" + this.table.sort.prop + " " + this.table.sort.order, {
                        method: "GET",
                    }).then(r => {
                        if (r.status != 200) {
                            throw new Error(r.statusText)
                        }
                        return r.json()
                    }).then(data => {
                        this.table.pagination.count = data.data.total
                        this.table.pagination.index = Math.min(this.table.pagination.index, Math.ceil(data.data.total / this.table.pagination.size))
                        this.table.records = data.data.records
                    }).catch(e => {
                        ElMessage.error(e.message)
                    }).finally(() => {
                        this.table.loading = false
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
                onTableSortChange({ prop, order }) {
                    if (!order) {
                        this.table.sort.prop = "id"
                        this.table.sort.order = "desc"
                    } else {
                        this.table.sort.prop = prop
                        this.table.sort.order = { ascending: "asc", descending: "desc" }[order]
                    }
                    this.onTableFetch()
                },
                onTableRowEdit(record) {
                    this.dialog.loading = true
                    fetch("?id=" + record.id, {
                        method: "GET",
                    }).then(r => {
                        if (r.status != 200) {
                            throw new Error(r.statusText)
                        }
                        return r.json()
                    }).then(data => {
                        this.dialog.record = { ...data.data, }
                        this.dialog.visible = true
                    }).catch(e => {
                        ElMessage.error(e.message)
                    }).finally(() => {
                        this.dialog.loading = false
                    })
                },
                onTableRowDelete(record) {
                    ElMessageBox.confirm("It will be deleted permanently. Continue ?", "Warning", {
                        confirmButtonText: "Confirm",
                        type: "warning",
                        beforeClose: (action, instance, done) => {
                            if (action === "confirm") {
                                instance.confirmButtonLoading = true
                                instance.confirmButtonText = "Delete..."
                                fetch("?id=" + record.id, {
                                    method: "DELETE",
                                }).then(r => r.json()).then(r => {
                                    if (r.code === "0") {
                                        ElMessage.success("Delete succeeded")
                                        this.onTableFetch()
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
                    FormRef.validate(valid => {
                        if (!valid) {
                            return false
                        }
                        fetch(this.dialog.record.id ? "?id=" + this.dialog.record.id : "", {
                            method: this.dialog.record.id ? "POST" : "PUT",
                            headers: {
                                "Content-Type": "application/json",
                            },
                            body: JSON.stringify(this.dialog.record),
                        }).then(r => r.json()).then(r => {
                            if (r.code === "0") {
                                ElMessage.success("Submit succeeded")
                                this.dialog.visible = false
                                this.onTableFetch()
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
        }).component("monaco-editor", {
            name: "MonacoEditor", // https://devpress.csdn.net/vue/66cd457d1338f221f9243439.html
            template: "<div ref='container' :style='{ width: this.width, height: this.height }'></div>",
            props: {
                modelValue: { type: String, default: "" },
                width: { type: String, default: "100%" },
                height: { type: String, default: "160px" },
                language: { type: String, default: "typescript" },
                readonly: { type: Boolean, default: false },
            },
            emits: ["update:modelValue"],
            data() {
                return {
                    editor: undefined,
                }
            },
            setup(props, { emit }) {
                const container = Vue.ref(),
                    loadjs = (() => {
                        const jsmap = new Map()
                        return (src) => {
                            if (jsmap.has(src)) {
                                return jsmap.get(src)
                            }
                            jsmap.set(src, new Promise((resolve, reject) => {
                                const s = document.createElement("script")
                                s.src = src
                                document.body.append(s)
                                s.addEventListener("load", () => resolve(true))
                                s.onerror = () => {
                                    document.body.removeChild(s)
                                    reject()
                                }
                            }))
                            return jsmap.get(src)
                        }
                    })()
                Vue.onMounted(() => {
                    loadjs("https://g.alicdn.com/code/lib/monaco-editor/0.34.1/min/vs/loader.js")
                        .then(() => {
                            window.require.config({ paths: { vs: "https://g.alicdn.com/code/lib/monaco-editor/0.34.1/min/vs" } })
                        })
                        .then(() => {
                            window.require(["vs/editor/editor.main"], () => {
                                this.editor = window.monaco.editor.create(container.value, {
                                    language: props.language,
                                    value: props.modelValue,
                                })
                                this.editor.onDidChangeModelContent(() => {
                                    emit("update:modelValue", this.editor.getValue())
                                })
                                this.editor.updateOptions({ readOnly: props.readonly ?? false })
                            })
                        })
                })
                Vue.watch(() => props.modelValue, newValue => {
                    if (this.editor) {
                        if (this.editor.getValue() !== newValue) {
                            this.editor.setValue(newValue)
                        }
                    }
                })
                Vue.watch(() => props.language, newValue => {
                    window.monaco.editor.setModelLanguage(this.editor.getModel(), newValue)
                })
                Vue.watch(() => props.readonly, newValue => {
                    this.editor.updateOptions({ readOnly: newValue })
                })
                return { container }
            },
        }).use(ElementPlus).mount("#app")
    </script>
</body>

</html>`

    protected name: string

    protected properties: Property[]

    constructor(name: string, properties: Property[]) {
        this.name = name
        this.properties = properties
    }

    public render() {
        return DynamicTable.TEMPLATE.replace("<%=properties%>", JSON.stringify(this.properties.map(p => {
            p.label = p.label ?? p.name
            return p
        })))
    }

    public abstract get(id: string, keyword: string, from: number, size: number): any;

    public abstract put(record: any): string

    public abstract post(id: string, record: any): void

    public abstract delete(id: string): void
}

export class MyDynamicTable extends DynamicTable {
    private dbc = $native("db")

    constructor(name: string, properties: Property[]) {
        super(name, properties)
        this.dbc.exec(`
            --drop table if exists ${this.name};
            create table if not exists ${this.name} (
                id integer primary key autoincrement,
                ${this.properties.map(f => {
            const type = { integer: "integer", number: "real", boolean: "boolean", date: "date", datetime: "datetime" }[f.type] || "text"
            return `${f.name} ${type}`
        }).join(",\n")}
            );
        `)
    }

    public get(id: string, keyword: string, from: number, size: number): any {
        if (id) {
            const columns = ["id", ...this.properties.filter(f => f.mode & PropertyMode.DETAIL).map(f => f.name)]
            return this.dbc.query(`select ${columns.join(", ")} from ${this.name} where id = ?`, id).pop()
        }
        const columns = ["id", ...this.properties.filter(f => f.mode & PropertyMode.LIST).map(f => f.name)]
        let wheres = "",
            params = []
        if (keyword) {
            keyword = keyword.replaceAll("%", "\%").replaceAll("_", "\_")
            const conditions = this.properties.filter(f => f.mode & PropertyMode.SEARCH).map(f => f.name + " like ?")
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
        const columns = this.properties.filter(f => f.mode & PropertyMode.ADD).map(f => f.name)
        this.dbc.exec(`insert into ${this.name}(${columns.join(", ")}) values(${columns.map(f => "?").join(", ")})`, ...columns.map(c => record[c]))
        return this.dbc.query("select LAST_INSERT_ROWID() rowID").pop().rowID
    }

    public post(id: string, record: any): void {
        const columns = this.properties.filter(f => f.mode & PropertyMode.UPDATE && f.name in record).map(f => f.name)
        this.dbc.exec(`update ${this.name} set ${columns.map(c => c + " = ?").join(", ")} where id = ?`, ...columns.map(c => record[c]), id)
    }

    public delete(id: string): void {
        this.dbc.exec(`delete from ${this.name} where id = ?`, id)
    }
}

export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
    const { name } = ctx.getPathVariables()
    if (!Schemas[name]) {
        return new ServiceResponse(404, undefined)
    }
    
    const dyntab = new MyDynamicTable(name, Schemas[name]),
        params = new Proxy(ctx.getURL().params, {
            get(target: any, property: string) {
                return target[property]?.[0]
            }
        })
    switch (ctx.getMethod()) {
        case "GET":
            if (params.id || params.keyword || params.from || params.size) {
                return dyntab.get(params.id, params.keyword, params.from, params.size)
            }
            return new ServiceResponse(200, { "Content-Type": "text/html" }, dyntab.render())
        case "PUT":
            return dyntab.put(ctx.getBody().toJson())
        case "POST":
            return dyntab.post(params.id, ctx.getBody().toJson())
        case "DELETE":
            return dyntab.delete(params.id)
        default:
            return new ServiceResponse(405, undefined)
    }
}

const Schemas: { [name: string]: Property[] } = {
    "user": [
        {
            name: "name",
            type: "string",
            mode: PropertyMode.LIST | PropertyMode.DETAIL | PropertyMode.ADD | PropertyMode.SEARCH,
        },
        {
            name: "age",
            type: "integer",
            mode: PropertyMode.DETAIL | PropertyMode.ADD | PropertyMode.UPDATE,
        },
        {
            name: "gender",
            type: "enum",
            mode: PropertyMode.DETAIL | PropertyMode.ADD,
        },
        {
            name: "active",
            type: "boolean",
            mode: PropertyMode.ADD | PropertyMode.DETAIL | PropertyMode.UPDATE | PropertyMode.LIST,
        },
        {
            name: "birthday",
            type: "date",
            mode: PropertyMode.ADD | PropertyMode.DETAIL,
        },
        {
            name: "remark",
            type: "code",
            mode: PropertyMode.ADD | PropertyMode.DETAIL,
        }
    ],
}
```

## Form

```typescript
//?name=DynamicForm&type=controller&url=dynamic/form
const TEMPLATE = `<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8" />
    <title></title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/element-plus/2.4.3/theme-chalk/index.min.css" />
    <script src="https://cdnjs.cloudflare.com/ajax/libs/vue/3.3.4/vue.global.prod.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/element-plus/2.4.3/index.full.min.js"></script>
</head>

<body style="margin: 40px;">
    <div id="app" v-cloak>
        <el-tabs tab-position="left">
            <el-tab-pane :label="func.name" v-for="(func) in funcs">
                <el-form label-width="120px" style="max-width: 600px;">
                    <el-form-item :label="param.name" v-for="(param) in func.params">
                        <el-date-picker v-model="param.value" type="date" v-if="param.type === 'date'"></el-date-picker>
                        <el-switch v-model="param.value" v-else-if="param.type === 'boolean'"></el-switch>
                        <el-input v-model="param.value" style="width: 240px" v-else></el-input>
                    </el-form-item>
                    <el-form-item>
                        <el-button type="primary" :loading="loading" @click="onSubmit(func)">Submit</el-button>
                    </el-form-item>
                </el-form>
            </el-tab-pane>
        </el-tabs>
    </div>
    <script>
        const { ElMessage, ElNotification, } = ElementPlus
        Vue.createApp({
            data() {
                return {
                    funcs: [
                        {
                            name: "greeting",
                            params: [{
                                "name": "name"
                            }]
                        },
                    ],
                    loading: false,
                }
            },
            methods: {
                onSubmit({ name, params }) {
                    this.loading = true
                    fetch("?name=" + encodeURIComponent(name), {
                        method: "POST",
                        headers: {
                            "Content-Type": "application/json",
                        },
                        body: JSON.stringify({
                            params: params.map(i => i.value),
                        }),
                    }).then(e => {
                        if (!e.ok) {
                            return e.json().then(i => new Error(i.message))
                        }
                        if (/^application\\/json/.test(e.headers.get("Content-Type"))) {
                            return e.json()
                        }
                        return e.text()
                    }).then(e => {
                        if (e instanceof Error) {
                            throw e
                        }
                        if (/^javascript:/.test(e)) {
                            return new Function(e.substring(11))()
                        }
                        ElNotification.success({
                            title: name,
                            message: JSON.stringify(e),
                            dangerouslyUseHTMLString: true,
                            duration: 0,
                        })
                    }).catch(e => {
                        ElMessage.error(e.message)
                    }).finally(() => {
                        this.loading = false
                    })
                },
            },
            mounted() {
                fetch("/service/dynamic/form?funcs").then(e => e.json()).then(e => {
                    this.funcs = e.data
                })
            },
        }).use(ElementPlus).mount("#app")
    </script>
</body>

</html>`

export default (app => app.run.bind(app))(new class {
    public run(ctx: ServiceContext) {
        switch (ctx.getMethod()) {
            case "GET":
                if ("funcs" in ctx.getURL().params) {
                    return this.toFuncs()
                }
                return new ServiceResponse(200, { "Content-Type": "text/html; charset=utf-8" }, TEMPLATE)
            case "POST":
                const name = ctx.getURL().params.name?.[0]
                if (!(name in MyFuncs)) {
                    break
                }
                if (typeof MyFuncs[name] === "function") {
                    return MyFuncs[name](...ctx.getBody().toJson())
                }
                return MyFuncs[name]
        }

        return new ServiceResponse(405, undefined)
    }

    private toFuncs() {
        return Object.keys(MyFuncs).map(name => {
            const params = MyFuncs[name].toString().match(/^function \(([^)]+)\)/)?.[1].split(", ").map(i => {
                const value = eval(MyFuncs[name].toString().match(new RegExp(`if \\\(${i} === void 0\\\) \\\{ ${i} = (.*); \\\}`))?.[1])
                switch (i.substring(0, 3)) {
                    case "_B_":
                        return {
                            name: i.substring(3),
                            type: "boolean",
                            value: value ?? false,
                        }
                    case "_N_":
                        return {
                            name: i.substring(3),
                            type: "number",
                            value: value ?? 0,
                        }
                    case "_D_":
                        return {
                            name: i.substring(3),
                            type: "date",
                            value: value ?? new Date(),
                        }
                    case "_D_":
                        return {
                            name: i.substring(3),
                            type: "datetime",
                            value: value ?? new Date(),
                        }
                    default:
                        return {
                            name: i.substring(0),
                            type: "string",
                            value: value ?? "",
                        }
                }
            }) || []
            return {
                name,
                params,
            }
        })
    }
})

const MyFuncs: { [name: string]: Function } = {
    修改账号密码(账号名, 密码) {
        return {
            账号名,
            密码,
        }
    },
    登录(账号名) {
        return "javascript:open(\"https://www.baidu.com/\")"
    },
}
```
