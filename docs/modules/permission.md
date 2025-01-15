# Permission

```typescript
//?name=node_modules/permission&type=module
export class Permission {
    private owns: string[]

    /**
     * 权限
     * 
     * @param owns 已有权限的集合
     */
    constructor(owns: string[]) {
        this.owns = owns
    }

    /**
     * 是否授权
     */
    public has(perm: string) {
        return this.owns.some(own => {
            const a = own.split(":"), // 已有权限，可包含通配符，例如 "user:*"
                b = perm.split(":") // 目标权限，例如 "user:add"
            for (let i = 0; i < a.length; i++) {
                if (a[i] === "*") {
                    continue
                }
                if (b[i] !== a[i]) {
                    return false
                }
            }
            return true
        })
    }
}

export const decorator = (input: Permission | string[]) => {
    const $ = input instanceof Permission ? input : new Permission(input)
    return function (perm: string) {
        return function (value: Function, context: any) {
            return function (this, ...args: any[]) {
                if (!$.has(perm)) {
                    throw new Error("Permisson denied")
                }
                return value.apply(this, args)
            }
        }
    }
}
```

### Usage

```typescript
import { Permission, decorator } from "./permission"

const owns = [
    "user:*",
    "user.role:guest",
]

const $ = new Permission(owns)
console.log($.has("user:add")) // true
console.log($.has("user.role:admin")) // false

const permission = decorator($)
class RoleController {
    @permission("role:add")
    add(name: string) {
        return name
    }
}
new RoleController().add("admin")
```
