# A custom module extends Number

```typescript
//?name=node_modules/number&type=module
declare global {
    interface Number {
        toIPv4(): string;
    }
    interface NumberConstructor {
        fromIPv4(ip: string): number;
    }
}
Number.prototype.toIPv4 = function () {
    return this.toString(2).padStart(32, '0').match(/(\d{8})/g).map(e => parseInt(e, 2)).join('.')
}
Number.fromIPv4 = function (ip) {
    return parseInt(ip.split(".").map(e => (parseInt(e) & 0xff).toString(2).padStart(8, '0')).join(""), 2)
}

export default { Number }
```

### Usage

```typescript
import "number"

Number.fromIPv4("192.168.0.1") // 3232235521

(3232235521).toIPv4() // "192.168.0.1"
```
