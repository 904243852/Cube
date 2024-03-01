# Validator

```typescript
//?name=node_modules/validator&type=module
import "date";

type GeneralSchema = {
    /**
     * 字段名称
     */
    label?: string;
    /**
     * 错误消息
     */
    message?: string;
    /**
     * 是否必填，默认非必填
     */
    required?: true;
    /**
     * 自定义校验方法
     */
    validation?: (parameter: any) => boolean;
};
type CollectionSchema = {
    /**
     * 是否为集合
     */
    collection?: {
        /**
         * 是否互相不重复
         */
        unique?: true;
        /**
         * 最小长度
         */
        min?: number;
        /**
         * 最大长度
         */
        max?: number;
    };
};
type StringSchema = CollectionSchema & {
    /**
     * 字符串类型
     */
    type: "string";
    /**
     * 最小长度
     */
    min?: number;
    /**
     * 最大长度
     */
    max?: number;
    /**
     * 正则匹配
     */
    pattern?: RegExp;
};
type NumberSchema = CollectionSchema & {
    /**
     * 数值类型
     */
    type: "integer" | "number";
    /**
     * 最小值
     */
    min?: number;
    /**
     * 最大值
     */
    max?: number;
};
type DateSchema = CollectionSchema & {
    /**
     * 时间类型
     */
    type: "date";
    /**
     * 格式，默认为 yyyy-MM-dd HH:mm:ss
     */
    layout?: string;
    /**
     * 最小时间
     */
    min?: string;
    /**
     * 最大时间
     */
    max?: string;
};
type ObjectSchema = Omit<CollectionSchema, "unique"> & {
    /**
     * 对象类型
     */
    type: "object";
    /**
     * 子属性
     */
    properties?: {
        [name: string]: Schema;
    };
};

const simple = {
    phone: {
        type: "string",
        pattern: /^1([358][0-9]|4[579]|66|7[0135678]|9[89])[0-9]{8}$/
    }
};
type SimpleSchema = CollectionSchema & {
    type: keyof typeof simple;
};

export type Schema = GeneralSchema & (
    StringSchema | NumberSchema | DateSchema | ObjectSchema | SimpleSchema
);

class Validator {
    public validate(parameter: any, schema: Schema, strict = false, alias?: string): void {
        // 如果没有 schema，则无需做校验
        if (schema == null) {
            return this.assert(!strict, `The parameter ${alias || ""} is unexpected.`);
        }

        schema.label = schema.label || alias || ""; // 字段名称默认为空

        // 校验必填
        this.assert(!(schema.required === true && parameter == null), `The parameter ${schema.label} is required.`);

        // 使用自定义校验器校验
        this.assert(!(schema.validation != null && schema.validation(parameter) === false), `The parameter ${schema.label} is invalid.`);

        if ("collection" in schema && schema.collection) {
            // 校验是否是集合
            this.assert(parameter instanceof Array, `The parameter ${schema.label} should be a collection.`);
    
            // 校验集合最小长度
            this.assert(!(schema.collection.min != null && parameter.length < schema.collection.min), `The min length of parameter ${schema.label} is ${schema.collection.min}.`);
            // 校验集合最大长度
            this.assert(!(schema.collection.max != null && parameter.length > schema.collection.max), `The max length of parameter ${schema.label} is ${schema.collection.max}.`);
    
            // 校验集合元素是否互相不重复
            this.assert(!("unique" in schema.collection && schema.collection.unique === true && this.unique(parameter).length !== parameter.length), `The parameter ${schema.label} should be unique.`);
    
            for (let p of parameter) {
                this.onValidation(p, schema, strict);
            }
        }

        this.onValidation(parameter, schema, strict);
    }

    private onValidation(parameter: any, schema: Schema, strict = false): void {
        schema = {
            ...schema,
            ...simple[schema.type as keyof typeof simple]
        } as Schema;

        this.assert(typeof parameter === ({ integer: "number", date: "string", object: "object" }[schema.type as string] || schema.type), `The parameter ${schema.label} is an invalid ${schema.type} value.`);

        switch (schema.type) {
            case "string":
                this.onStringValidation(parameter, schema);
                break;
            case "integer":
                this.assert(/^\d+$/.test(parameter + ""), `The parameter ${schema.label} is an invalid ${schema.type} value.`);
            case "number":
                this.onNumberValidation(parameter, schema);
                break;
            case "date":
                this.onDateValidation(parameter, schema);
                break;
            case "object":
                for (let property in (strict ? parameter : schema.properties)) {
                    this.validate(parameter[property], schema.properties?.[property]!, strict, property);
                }
                break;
            default:
                throw new Error(`Unexpected parameter type.`);
        }
    }

    private onStringValidation(parameter: string, schema: GeneralSchema & StringSchema): void {
        // 使用正则表达式校验
        this.assert(!(schema.pattern != null && !schema.pattern.test(parameter + "")), `The parameter ${schema.label} is not match ${schema.pattern + ""}.`);

        // 校验长度
        this.assert(!(schema.min != null && parameter.length < schema.min), `The length of parameter ${schema.label} can not less than ${schema.max}.`);
        this.assert(!(schema.max != null && parameter.length > schema.max), `The length of parameter ${schema.label} can not greater than ${schema.max}.`);
    }

    private onNumberValidation(parameter: number, schema: GeneralSchema & NumberSchema): void {
        // 校验最小值
        this.assert(!(schema.min != null && parameter < schema.min), `The parameter ${schema.label} can not less than ${schema.min}.`);
        // 校验最大值
        this.assert(!(schema.max != null && parameter > schema.max), `The parameter ${schema.label} can not greater than ${schema.max}.`);
    }

    private onDateValidation(parameter: string, schema: GeneralSchema & DateSchema): void {
        let layout = schema.layout || "yyyy-MM-dd HH:mm:ss";

        let p: Date;

        try {
            p = Date.toDate(parameter, layout);
        } catch (e) {
            this.assert(false, `The parameter ${schema.label} is not a valid date.`);
            return;
        }

        this.assert(!(schema.min != null && p.getTime() <= Date.toDate(schema.min, layout).getTime()), `The parameter ${schema.label} can not before than ${schema.min}.`)
        this.assert(!(schema.max != null && p.getTime() >= Date.toDate(schema.max, layout).getTime()), `The parameter ${schema.label} can not later than ${schema.max}.`)
    }

    private assert(expression: boolean, errmsg: string) {
        if (expression) {
            return;
        }
        throw new Error(errmsg.replace(/\s{1,}/g, " "));
    }

    private unique<T>(arr: T[]): T[] {
        return [...new Set(arr)];
    }
}

export const validator = new Validator();

export function validate(...schemas: Schema[]) {
    return function (target: any, method: string, descriptor: PropertyDescriptor) {
        return {
            value: function (...input: any[]) {
                for (let i = 0; i < schemas.length; i++) {
                    if (schemas[i] != null) {
                        validator.validate(input[i], schemas[i]);
                    }
                }
                return (<Function>descriptor.value).apply(this, input);
            }
        };
    }
}
```

### Usage

```typescript
import { validator } from "validator"

validator.validate({
    id: 1,
    name: "zhangsan",
    msisdn: "18612345678",
}, {
    type: "object",
    properties: {
        id: {
            type: "number",
            min: 1,
        },
        name: {
            type: "string",
            max: 32,
        },
        msisdn: {
            type: "phone",
        }
    }
}, true)
```
