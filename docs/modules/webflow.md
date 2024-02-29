# WebFlow

```typescript
//#node_modules/webflow&type=module
interface Transition {
    on: string;
    to: string;
}
interface StartState {
    /**
     * 开始类型
     */
    $type: "start";
    /**
     * 初始化方法
     */
    evaluate?: (data: any) => void;
    transition: { to: string; }
}
interface ViewState {
    /**
     * 视图类型
     */
    $type: "view";
    /**
     * view 地址
     */
    view: string;
    /**
     * 所有候选操作
     */
    transitions: Transition[];
}
interface ActionState {
    /**
     * 动作类型
     */
    $type: "action";
    /**
     * 初始化方法
     */
    evaluate: (data: any) => void;
    /**
     * 所有候选操作
     */
    transitions: Transition[];
}
interface SubflowState {
    /**
     * 子流程类型
     */
    $type: "subflow";
    /**
     * 所有候选操作
     */
    transitions: Transition[];
}
interface EndState {
    /**
     * 结束类型
     */
    $type: "end";
    /**
     * view 地址
     */
    view: string;
}

type States = {
    [id: string]: StartState | ViewState | ActionState | SubflowState | EndState;
};

export class WebFlow {
    private states: States;

    private startends: [string, string] = ["", ""];

    private session = {
        get() {
            return {
                id: sessionStorage.getItem("id"),
                data: JSON.parse(sessionStorage.getItem("data") || "")
            };
        },
        set(value: { id: string; data: object; }): void {
            sessionStorage.setItem("id", value?.id);
            sessionStorage.setItem("data", JSON.stringify(value?.data));
        }
    }

    constructor(states: States) {
        for (let id in states) {
            if (states[id].$type === "start") {
                if (this.startends[0]) {
                    throw new Error("Duplicate start state.");
                }
                this.startends[0] = id;
            }
            if (states[id].$type === "end") {
                if (this.startends[1]) {
                    throw new Error("Duplicate end state.");
                }
                this.startends[1] = id;
            }
        }
        if (!this.startends[0] || !this.startends[1]) {
            throw new Error("The start state or end state is required.");
        }

        this.states = states;
    }

    public start() {
        this.onNextState(this.startends[0], new Object());
    }

    public complete(event: string) {
        let { id, data } = this.session.get();
        if (!id) {
            throw new Error("The current session does not have any web flows.")
        }

        const state = this.states[id];
        if (!state) {
            throw new Error(`The state "${id}" is not found.`)
        }
        if (state.$type === "start" || state.$type === "end") {
            throw new Error("The operation is not allowed for the start state or end state.")
        }

        const transition = state.transitions.filter(t => t.on === event).pop();
        if (!transition) {
            throw new Error(`The event "${event}" is invalid for the current state "${id}", the expected value is "${state.transitions.map(t => t.on).join("\", \"")}".`)
        }

        this.onNextState(transition.to, data);
    }

    private onNextState(id: string, data: object): void {
        if (!id) {
            throw new Error("The state id is null or empty.")
        }

        const state = this.states[id];
        if (!state) {
            throw new Error(`The state "${id}" is not found.`)
        }

        switch (state.$type) {
            case "start":
                if (state.evaluate) {
                    state.evaluate.apply(this, [data]);
                }
                return this.onNextState(state.transition.to, data);
            case "view":
                this.redirect(state.view);
                break;
            case "action":
                if (state.evaluate) {
                    state.evaluate.apply(this, [data]);
                }
                break;
            case "end":
                this.session.set({ id: "", data: {} }); // 流程已结束，清理当前 state
                return this.redirect(state.view);
            default:
                throw new Error(`The state "${state.$type}" is not implemented.`);
        }

        // 保存当前 state id
        this.session.set({ id, data });
    }

    private redirect(view: string) {
        console.log(`redirect to view ${view}`);
        // response.setHeader("Location", view);
        // response.setStatusCode(302);
    }
}
```

### Usage

```typescript
import { WebFlow } from "webflow"

let webflow = new WebFlow({
    start: {
        $type: "start",
        evaluate: function (data) {
            data.start = new Date();
            console.log("this is start state evaluate", data);
        },
        transition: {
            to: "view_login"
        }
    },
    view_login: {
        $type: "view",
        view: "login.html",
        transitions: [
            { on: "submit", to: "action_login" }
        ]
    },
    action_login: {
        $type: "action",
        evaluate: function (data) {
            data.action_login = new Date();
            console.log("this is action state evaluate for login", data);
        },
        transitions: [
            { on: "success", to: "view_confirm" }, // 登录成功后，跳转至短信验证页面
            { on: "first", to: "view_resetpwd" } // 如果是第一次登录，跳转至重置密码页面
        ]
    },
    view_resetpwd: {
        $type: "view",
        view: "reset-pwd.html",
        transitions: [
            { on: "submit", to: "action_resetpwd" }
        ]
    },
    action_resetpwd: {
        $type: "action",
        evaluate: function (data) {
            data.action_resetpwd = new Date();
            console.log("this is action state evaluate for resetpwd", data);
        },
        transitions: [
            { on: "success", to: "view_login" } // 修改密码成功后，返回登录页重新登录
        ]
    },
    view_confirm: {
        $type: "view",
        view: "confirm.html",
        transitions: [
            { on: "submit", to: "action_confirm" }
        ]
    },
    action_confirm: {
        $type: "action",
        evaluate: function (data) {
            data.action_confirm = new Date();
            console.log("this is action state evaluate for confirm", data);
        },
        transitions: [
            { on: "success", to: "end" }
        ]
    },
    end: {
        $type: "end",
        view: "home.html"
    }
});

webflow.start(); // 发起流程，跳转至登录页
webflow.complete("submit"); // 提交账号密码表单
webflow.complete("first"); // 第一次登录，跳转至修改密码页面
webflow.complete("submit"); // 提交密码变更表单
webflow.complete("success"); // 密码变更成功，重新跳转回登录页
webflow.complete("submit"); // 提交账号密码表单
webflow.complete("success"); // 账号密码校验成功，跳转至短信确认页面
webflow.complete("submit"); // 提交短信验证码表单
webflow.complete("success"); // 短信验证码校验成功，跳转至首页
```
