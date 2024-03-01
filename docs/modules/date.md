# Date

```typescript
//?name=node_modules/date&type=module
declare global {
    interface Date {
        toString(layout?: string): string
    }
    interface DateConstructor {
        toDate(value: string, layout: string): Date
    }
}

const L = {
    "yyyy|yy": ["FullYear"],
    "M{1,2}": ["Month", 1],
    "d{1,2}": ["Date"],
    "H{1,2}": ["Hours"],
    "m{1,2}": ["Minutes"],
    "s{1,2}": ["Seconds"],
    "S{1,3}": ["Milliseconds", 0, -1],
}

const toString = Date.prototype.toString

Date.prototype.toString = function(layout?: string) {
    if (!layout) {
        return toString()
    }
    for (const l in L) {
        const m = layout.match(new RegExp(`(${l})`))
        if (m) {
            layout = layout.replace(m[1], (this[`get${L[l][0]}`]() + (L[l][1] || 0)).toString().padStart(m[1].length, "0").substr(Math.min(m[1].length * (L[l][2] || 1) * -1, 0), m[1].length))
        }
    }
    return layout
}

Date.toDate = function(value: string, layout: string): Date {
    const t = new Date(0)
    for (const l in L) {
        const r = new RegExp(`(${l})`).exec(layout)
        if (r && r.length) {
            t[`set${L[l][0]}`](Number(value.substr(r.index, r[0].length)) - (L[l][1] || 0))
        }
    }
    return t
}

export default { Date }
```

### Usage

```typescript
import "date"

export default function (ctx) {
    return new Date().toString("yyyy-MM-dd HH:mm:ss.S")
}
```
