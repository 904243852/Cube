# BPM

```typescript
//?name=node_modules/bpm&type=module
//#region types

enum ActivityType {
    StartEvent = "startEvent", EndEvent = "endEvent",
    UserTask = "userTask", ScriptTask = "scriptTask",
    ParallelGateway = "parallelGateway", ExclusiveGateway = "exclusiveGateway", // InclusiveGateway, EventGateway,
}

abstract class Activity {
    /**
     * 活动标识
     */
    id: string;
    /**
     * 活动类型
     */
    type: ActivityType;

    constructor(id: string, type: ActivityType) {
        this.id = id;
        this.type = type;
    }
}

abstract class Event extends Activity {
}
export class StartEvent extends Event {
    /**
     * 下一步活动
     */
    next: string;

    constructor(input: { next: string }) {
        super("start", ActivityType.StartEvent);
        this.next = input.next;
    }
}
export class EndEvent extends Event {
    constructor(input: { id: string }) {
        super(input.id, ActivityType.EndEvent);
    }
}

abstract class Task extends Activity {
    /**
     * 活动名称
     */
    name: string;
    /**
     * 下一步活动
     */
    next: string;
}
export class UserTask extends Task {
    /**
     * 受让人
     */
    assignee: string;

    constructor(input: { id: string, name: string, assignee: string, next: string }) {
        super(input.id, ActivityType.UserTask);
        this.name = input.name;
        this.next = input.next;
        this.assignee = input.assignee;
    }
}
export class ScriptTask extends Task {
    /**
     * 脚本方法
     */
    func: Function;

    constructor(input: { id: string, name: string, func: Function, next: string }) {
        super(input.id, ActivityType.ScriptTask);
        this.name = input.name;
        this.next = input.next;
        this.func = input.func;
    }
}

abstract class Gateway extends Activity {
}
export class ParallelGateway extends Gateway {
    /**
     * 下一步活动集合
     */
    next: string[];

    constructor(input: { id: string, next: string[] }) {
        super(input.id, ActivityType.ParallelGateway);
        this.next = input.next;
    }
}
export class ExclusiveGateway extends Gateway {
    /**
     * 下一步活动路由
     */
    next: { [conditionExpression: string]: string; };

    constructor(input: { id: string, next: { [conditionExpression: string]: string } }) {
        super(input.id, ActivityType.ExclusiveGateway);
        this.next = input.next;
    }
}

type Process = {
    id: string;
    name: string;
    activities: Activity[];
}

//#endregion

export class BusinessProcessManagement {
    private db = $native("db");

    private process: Process;

    public constructor(process: Process) {
        this.db.exec(`
            create table if not exists process_instance (
                id integer primary key autoincrement,
                process_id text not null,
                status text not null default ('Processing'),
                variables text
            );
            create table if not exists process_activity_instance (
                id integer primary key autoincrement,
                process_instance_id integer not null,
                activity_id text not null,
                assignee text,
                --creator text,
                start_time integer not null,
                end_time integer
            );
        `);
        this.process = this.verifyProcess(process);
    }

    public start(variables: object): string {
        // 保存流程实例
        const processInstanceId = this.db.query("insert into process_instance (process_id, variables) values (?, ?) returning id",
            this.process.id,
            JSON.stringify(variables),
        ).pop()?.id;

        // 发起流程活动
        this.createActivityInstance(processInstanceId,
            "start",
            variables,
        );

        return processInstanceId;
    }

    public suspend(processInstanceId: string) {
        this.db.exec("update process_instance set status = 'Suspended' where id = ? and status = 'Processing'", processInstanceId);
    }

    public resume(processInstanceId: string) {
        this.db.exec("update process_instance set status = 'Processing' where id = ? and status = 'Suspended'", processInstanceId);
    }

    public terminate(processInstanceId: string) {
        this.db.exec("update process_instance set status = 'Terminated' where id = ? and status = 'Processing'", processInstanceId);
    }

    public complete(taskId: string, variable?: object) {
        const processActivityInstance = this.db.query("select activity_id, process_instance_id from process_activity_instance where id = ?", taskId).pop() as { activity_id: string, process_instance_id: string };
        if (processActivityInstance == null) {
            throw new Error("task not found");
        }

        const processInstance = this.db.query("select id, variables from process_instance where id = ?", processActivityInstance.process_instance_id).pop() as { id: string, variables: string };

        // 保存活动实例
        this.db.exec("update process_activity_instance set end_time = ? where id = ?", new Date().getTime(), taskId);

        // 查找并执行下一步活动
        const currentActivity = <Task>this.process.activities.filter(i => i.id === processActivityInstance.activity_id).pop();

        this.createActivityInstance(processInstance.id,
            currentActivity.next,
            {
                ...JSON.parse(processInstance.variables),
                ...variable,
            },
        );
    }

    public delegate(taskId: string, assignee: string) {
        this.db.exec("update process_activity_instance set assignee = ? where id = ? and end_time is null", taskId, assignee);
    }

    public queryTasks(assignee: string) {
        return this.db.query("select * from process_activity_instance where assignee = ? and end_time is null", assignee) as {
            id: string,
            activity_id: string,
            process_instance_id: string,
            assignee: string,
            start_time: number,
            end_time: number,
        }[];
    }

    public queryProcessActivityInstance(processInstanceId: string) {
        return this.db.query("select * from process_activity_instance where process_instance_id = ? order by start_time desc", processInstanceId) as {
            id: string,
            activity_id: string,
            process_instance_id: string,
            assignee: string,
            start_time: number,
            end_time: number,
        }[];
    }

    private createActivityInstance(processInstanceId: string, activityId: string, variables: object) {
        const activity = this.process.activities.filter(a => a.id === activityId).pop();
        if (activity == null) {
            throw new Error("activity does not exist: " + activityId);
        }

        switch (activity.type) {
            case ActivityType.StartEvent: // 开始事件
                this.createActivityInstance(processInstanceId,
                    (<StartEvent>activity).next,
                    variables);
                break;
            case ActivityType.EndEvent: // 结束事件
                this.db.exec("update process_instance set status = 'Completed' where id = ?", processInstanceId);
                break;
            case ActivityType.UserTask: // 用户任务
                // 保存活动实例
                this.db.exec("insert into process_activity_instance (process_instance_id, activity_id, assignee, start_time) values (?, ?, ?, ?)",
                    processInstanceId,
                    activity.id,
                    this.resolve(variables, (<UserTask>activity).assignee),
                    new Date().getTime(),
                );
                break;
            case ActivityType.ExclusiveGateway: // 排他网关
                const exclusiveGateway = <ExclusiveGateway>activity;
                Object.keys(exclusiveGateway.next)
                    .filter(e => this.resolve(variables, e) === "true") // 计算排他网关上的条件表达式
                    .forEach(e => this.createActivityInstance(processInstanceId,
                        exclusiveGateway.next[e],
                        variables,
                    ));
                break;
            case ActivityType.ParallelGateway: // 平行网关
                // 如果上一步有并行执行的活动，需要等待这些活动同时执行结束后才能触发
                const lastActivityIds = [
                    "",
                    ...this.process.activities
                        .filter(a => this.getNextsOfActivity(a).indexOf(activity.id) !== -1)
                        .map(a => a.id),
                ];
                const incompletedTasksCount = Number(this.db.query(`select count(1) as incompletedTasksCount from process_activity_instance where process_instance_id = ? and activity_id in (${lastActivityIds.map(() => "?").join(",")}) and end_time is null`, processInstanceId, ...lastActivityIds).pop()?.incompletedTasksCount);
                if (incompletedTasksCount === 0) {
                    (<ParallelGateway>activity).next
                        .forEach(next => {
                            this.createActivityInstance(processInstanceId,
                                next,
                                variables,
                            )
                        });
                }
                break;
            default:
                throw new Error(`activity type ${activity.type} not implemented`);
        }
    }

    private resolve(variables: object, expression: string = ""): string {
        return eval("`" + Object.keys(variables).reduce((p, c) => { p = p.replace(c, "variables." + c); return p; }, expression) + "`");
    }

    private getNextsOfActivity(activity: Activity): string[] {
        if (!("next" in activity)) {
            return [];
        }
        if (typeof (activity["next"]) === "string") {
            return [activity["next"]];
        }
        if (activity["next"] instanceof Array) {
            return activity["next"];
        }
        return Object.keys(<Object>activity["next"]).map(n => (<Object>activity["next"])[n]);
    }

    private verifyProcess(process: Process) {
        const activityIds = process.activities.map(a => a.id);

        // 校验重复 id
        const duplicateActivityIds = <string[]>[];
        activityIds.sort((a, b) => {
            if (a == b && duplicateActivityIds.indexOf(a) === -1) {
                duplicateActivityIds.push(a);
            }
            return 0;
        });
        if (duplicateActivityIds.length) {
            throw new Error("duplicate activity id: " + duplicateActivityIds.join(", "));
        }

        let startEventCount = 0,
            endEventCount = 0;
        for (const activity of process.activities) {
            // 校验 next
            const invalidNexts = this.getNextsOfActivity(activity).filter(n => activityIds.indexOf(n) === -1);
            if (invalidNexts.length) {
                throw new Error(`next of activity '${activity.id}' does not exist: '${invalidNexts.pop()}'`);
            }

            if (activity.type === ActivityType.StartEvent) {
                startEventCount += 1;
            }
            if (activity.type === ActivityType.EndEvent) {
                endEventCount += 1;
            }
        }

        // 校验 startEvent、endEvent
        if (startEventCount === 0) {
            throw new Error("startEvent is required");
        }
        if (endEventCount === 0) {
            throw new Error("endEvent is required");
        }
        if (startEventCount > 1) {
            throw new Error("more than one startEvent is not allowed");
        }

        return process;
    }
}
```

### Usage

```typescript
import { BusinessProcessManagement, StartEvent, UserTask, ExclusiveGateway, ParallelGateway, EndEvent, } from "bpm"

const bpm = new BusinessProcessManagement({
    id: "expense",
    name: "报销流程",
    activities: [
        new StartEvent({ next: "t1" }),
        new UserTask({ id: "t1", name: "填写报销单", assignee: "${form.username}", next: "t2" }),
        new UserTask({ id: "t2", name: "部门经理审批", assignee: "${manager}", next: "g1" }),
        new ExclusiveGateway({
            id: "g1",
            next: {
                "${form.amount <= 500}": "t3",
                "${form.amount > 500}": "t4",
            }
        }),
        new UserTask({ id: "t3", name: "人事审批", assignee: "${cho}", next: "g2" }),
        new UserTask({ id: "t4", name: "总经理审批", assignee: "${ceo}", next: "t3" }),
        new ParallelGateway({
            id: "g2",
            next: ["t5", "t6"],
        }),
        new UserTask({ id: "t5", name: "打印报销申请单", assignee: "${cho}", next: "g3" }),
        new UserTask({ id: "t6", name: "粘贴发票", assignee: "${form.username}", next: "g3" }),
        new ParallelGateway({
            id: "g3",
            next: ["t7"],
        }),
        new UserTask({ id: "t7", name: "财务打款", assignee: "${cfo}", next: "end" }),
        new EndEvent({ id: "end" }),
    ],
})

export default function (ctx: ServiceContext): ServiceResponse {
    // 发起流程
    const processInstanceId = bpm.start({
        form: {
            username: "zhangsan",
            amount: 500,
        },
        manager: "lisi",
        ceo: "wangwu",
        cho: "zhaoliu",
        cfo: "sunqi",
    })

    // zhangsan 填写报销单
    let task = bpm.queryTasks("zhangsan").pop()
    bpm.complete(task.id)

    // lisi 部门经理审批
    task = bpm.queryTasks("lisi").pop()
    bpm.complete(task.id)

    // zhaoliu 人事审批
    task = bpm.queryTasks("zhaoliu").pop()
    bpm.complete(task.id)

    // zhangsan 粘贴发票
    task = bpm.queryTasks("zhangsan").pop()
    bpm.complete(task.id)

    // zhaoliu 打印报销申请单
    task = bpm.queryTasks("zhaoliu").pop()
    bpm.complete(task.id)

    // sunqi 财务打款
    task = bpm.queryTasks("sunqi").pop()
    bpm.complete(task.id)

    const processActivityInstances = bpm.queryProcessActivityInstance(processInstanceId)

    $native("db").exec(`
        drop table process_instance;
        drop table process_activity_instance;
    `)

    return new ServiceResponse(200, { "Content-Type": "text/html; charset=utf-8" }, processActivityInstances.map(i => {
        return `${new Date(i.start_time).toString()} - ${new Date(i.end_time).toString()}: ${i.assignee} ${process.activities.filter(a => a.id === i.activity_id).pop()["name"]}`
    }).join("<br/>"))
}
```
