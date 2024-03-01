# Return a view with asynchronous vues

1. Create a template with lang `html`.
    ```html
    <!-- http://127.0.0.1:8090/editor.html?name=index&type=template&lang=html -->
    <!DOCTYPE html>
    <html>
    <head>
        <meta charset="utf-8" />
        <title>{{ .title }}</title>
        <style>
            * {
                margin: 0;
                padding: 0;
            }
            html, body {
                width: 100%;
                height: 100%;
            }
            html {
                overflow: hidden;
            }
        </style>
    </head>
    <body>
        <script src="https://cdn.bootcdn.net/ajax/libs/vue/2.7.14/vue.min.js"></script>
        <script src="https://cdn.bootcdn.net/ajax/libs/vue-router/3.6.5/vue-router.min.js"></script>
        <script src="https://unpkg.com/http-vue-loader"></script>
        <router-view id="container"></router-view>
        <script>
            const router = new VueRouter({
                mode: "hash"
            })
            router.beforeEach((to, from, next) => {
                if (to.matched.length) { // 当前路由已匹配上
                    next() // 直接渲染当前路由
                    return
                }
                router.addRoute({ // 动态添加路由
                    path: to.path,
                    component: httpVueLoader(`../resource${to.path === "/" ? "/index" : to.path}.vue`), // 远程加载组件
                })
                next(to.path) // 重新进入 beforeEach 方法
            })
            new Vue({ router }).$mount("#container")
        </script>
    </body>
    </html>
    ```

2. Create a resource with lang `vue` and url `/resource/greeting.vue`.
    ```html
    <!-- http://127.0.0.1:8090/editor.html?name=greeting&type=reaource&lang=vue -->
    <template>
        <p>hello, {{ name }}</p>
    </template>

    <script>
        module.exports = {
            data: function() {
                return {
                    name: "world",
                }
            }
        }
    </script>

    <style scoped>
        p {
            color: #000;
        }
    </style>
    ```

3. Create a controller with url `/service/`.
    ```typescript
    // http://127.0.0.1:8090/editor.html?name=index
    export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
        return $native("template")("index", {
            title: "this is title",
        })
    }
    ```

4. You can preview at `http://127.0.0.1:8090/service/#/greeting`
