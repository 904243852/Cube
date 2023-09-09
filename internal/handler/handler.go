package handler

import (
	"cube/internal/builtin"
	"cube/internal/config"
	"cube/internal/util"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"io/fs"
	"net/http"
)

func InitHandle(web *embed.FS) {
	// 运行态
	http.HandleFunc("/service/", HandleService)
	http.HandleFunc("/resource/", HandleResource)

	// 开发态
	http.HandleFunc("/source", authenticate(HandleSource))

	fileList, _ := fs.Sub(web, "web")
	http.Handle("/", http.FileServer(http.FS(fileList)))
}

func toSuccess(w http.ResponseWriter, data interface{}) {
	switch v := data.(type) {
	case string:
		fmt.Fprintf(w, "%s", v)
	case []uint8: // byte 即 uint8，通过 goja.Runtime.Set(...) 方法写入的类型为 []byte 的变量或方法返回值（见 goja.Runtime.ToValue(...) 方法实现：变量因匹配 reflect.Slice 类型从而被包装成 goja.objectGoSliceReflect 对象），在调用 goja.Value.Export() 后将会被转换成为 []uint8 类型（见 goja.Object.Export() 方法实现：变量调用 goja.objectGoReflect.origValue.Interface() 反射方法从而被还原成了原始 []uint8 类型）
		w.Write(v)
	case builtin.Buffer:
		w.Write(([]byte)(v))
	case *builtin.Buffer:
		w.Write(([]byte)(*v))
	case *builtin.ServiceResponse: // 自定义响应
		builtin.ResponseWithServiceResponse(w, v)
	default: // map[string]interface[]
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false) // 见 https://pkg.go.dev/encoding/json#Marshal，字符串值编码为强制为有效 UTF-8 的 JSON 字符串，用 Unicode 替换符文替换无效字节。为了使 JSON 能够安全地嵌入 HTML 标记中，字符串使用 HTMLEscape 编码，它将替换 `<`、`>`、`&`、`U+2028` 和 `U+2029`，并转义到 `\u003c`、`\u003e`、`\u0026`、`\u2028` 和 `\u2029`。在使用编码器时，可以通过调用 SetEscapeHTML(false) 禁用此替换。
		enc.Encode(map[string]interface{}{
			"code":    "0",
			"message": "success",
			"data":    v, // 注：这里的 data 如果为 []byte 类型或包含 []byte 类型的属性，在通过 json 序列化后将会被自动转码为 base64 字符串
		})
	}
}

func toError(w http.ResponseWriter, err error) {
	code, message := "1", err.Error() // 错误信息默认包含了异常信息和调用栈
	if e, ok := err.(*goja.Exception); ok {
		if o, ok := e.Value().Export().(map[string]interface{}); ok {
			if m, ok := util.ExportMapValue(o, "message", "string"); ok {
				message = m.(string) // 获取 throw 对象中的 message 和 code 属性，作为失败响应的错误信息和错误码
			}
			if c, ok := util.ExportMapValue(o, "code", "string"); ok {
				code = c.(string)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest) // 在同一次请求响应过程中，只能调用一次 WriteHeader，否则会抛出异常 http: superfluous response.WriteHeader call from ...
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

func authenticate(next func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// 如果未配置用户名密码，直接执行
		if config.IdeAuthorization == "" {
			next(w, r)
			return
		}

		// 校验用户名密码
		a := &util.DigestAuth{}
		if a.VerifyWithMd5(r.Header.Get("Authorization"), r.Method, config.IdeAuthorization) {
			next(w, r)
			return
		}

		// 用户名密码校验不通过
		w.Header().Set("WWW-Authenticate", "Digest nonce=\""+a.Random(16)+"\", opaque=\""+a.Random(16)+"\", qop=\"auth\"")
		w.WriteHeader(http.StatusUnauthorized)
	}
}
