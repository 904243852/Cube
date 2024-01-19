# Makefile 中命令需要转义字符 `$` -> `$$`、`'` -> `'\''`

# 运行
run: # 从代码中运行
	@go run . -n 8

watch: # 监听当前目录下的相关文件变动，实时编译、运行
	@gowatch -o ./cube

kill:
	@ps -ef | grep -P "/cube|/gowatch" | grep -v "grep" | awk '{print $$2}' | xargs kill -9

# 编译
.ONESHELL:
build: clean # 默认使用 CDN 资源并且不使用 UPX 压缩，即 make build ENABLE_CDN=1 ENABLE_UPX=0
	@
	# 是否使用 CDN 资源
	if [ "$(ENABLE_CDN)" = "0" ]; then # 构建一个不依赖于 CDN 的版本，依赖的 js、css 等库文件将下载至本地 web/libs 目录下
		# 下载除 monaco-editor 外所有 css、js 资源文件
		grep -hor "https://cdn.bootcdn.net/ajax/libs/[^\"'\'''\'']*" ./web | grep -v "monaco-editor" | while read uri
		do
			name=$${uri#https://cdn.bootcdn.net/ajax/}
			mkdir -p "web/$$(dirname $${name})"
			curl -s "https://cdn.bootcdn.net/ajax/$$name" -o "web/$${name}"
		done
		# 下载 monaco-editor 资源文件
		export LANG=C.UTF-8
		export version=`grep -horP "monaco-editor/[\d\.]+" ./web | uniq | cut -d "/" -f 2`
		curl -sOL "https://registry.npm.taobao.org/monaco-editor/-/monaco-editor-$$version.tgz"
		mkdir -p "./web/libs/monaco-editor/$$version/"
		tar -zxf monaco-editor-$$version.tgz -C "./web/libs/monaco-editor/$$version/" --strip-components 1 "package/min"
		rm monaco-editor-$$version.tgz
		# 替换 html 中的 cdn 地址
		sed -i 's#https://cdn.bootcdn.net/ajax/libs#/libs#g' web/*.html
		# 编译（删除符号、调试信息）
		go build -ldflags "-s -w" .
		# 还原 html 中的 cdn 地址
		sed -i 's#/libs#https://cdn.bootcdn.net/ajax/libs#g' web/*.html
	else
		go build -ldflags "-s -w" .
	fi
	# 是否使用 UPX 压缩
	if [ "$(ENABLE_UPX)" = "1" ]; then
		if [ "$(shell uname)" = "Linux" ]; then
			upx -9 -q -o cubemin cube
		else
			upx -9 -q -o cubemin.exe cube.exe
		fi
	fi

clean:
	@rm -f cube cubemin cube.exe cubemin.exe

# 开发
tidy: # 安装依赖、删除 go.mod、go.sum 中的无用依赖
	@go mod tidy

update: # 更新依赖
	@go get -u .

wrk:
	@wrk -t8 -c256 -R 20000 -d5s http://127.0.0.1:8090/service/greeting

fmt:
	@find ./ -name "*.go" | xargs -I {} go fmt {}

# 证书
crt:
	@ls | grep -P 'ca\.(key|crt)' > /dev/null && echo 'The ca.key or ca.crt already existed, skip.' || openssl req -new -days 3650 -x509 -nodes -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=Sunke Root CA" -keyout ca.key -out ca.crt
	@bash -c 'openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=localhost" -out server.csr && openssl x509 -sha256 -req -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt'

ccrt:
	@openssl req -newkey rsa:2048 -nodes -keyout client.key -subj "/C=CN/ST=BJ/L=BJ/O=/CN=" -out client.csr && openssl x509 -sha256 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
