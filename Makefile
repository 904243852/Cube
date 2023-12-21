run: # 运行
	@go run . -n 8

watch: # 监听当前目录下的相关文件变动，实时编译、运行
	@gowatch -o ./cube

build: clean # 构建
	@go build .

buildx: clean # 构建（删除符号、调试信息）、压缩（upx）
	@go build -ldflags "-s -w" .
ifeq ($(shell uname), Linux)
	@upx -9 -q -o cubemin cube
else
	@upx -9 -q -o cubemin.exe cube.exe
endif

clean:
	@rm -rf cube cubemin *.log *.exe

tidy: # 安装依赖、删除 go.mod、go.sum 中的无用依赖
	@go mod tidy

crt:
	@ls | grep -P 'ca\.(key|crt)' > /dev/null && echo 'The ca.key or ca.crt already existed, skip.' || openssl req -new -days 3650 -x509 -nodes -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=Sunke Root CA" -keyout ca.key -out ca.crt
	@bash -c 'openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=localhost" -out server.csr && openssl x509 -sha256 -req -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt'

ccrt:
	@openssl req -newkey rsa:2048 -nodes -keyout client.key -subj "/C=CN/ST=BJ/L=BJ/O=/CN=" -out client.csr && openssl x509 -sha256 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

kill:
	@ps -ef | grep -P "/cube|/gowatch" | grep -v "grep" | awk '{print $$2}' | xargs kill -9

update: # 更新依赖
	@go get -u .

wrk:
	@wrk -t8 -c256 -R 20000 -d5s http://127.0.0.1:8090/service/greeting

fmt:
	@find ./ -name "*.go" | xargs -I {} go fmt {}

buildz: # 构建一个不依赖于 cdn 的版本，依赖的 js、css 等库文件将下载至本地 web/libs 目录下（Makefile 中命令需要转义字符 `$` -> `$$`、`'` -> `'\''`）
	@bash -c 'grep -hor "https://cdn.bootcdn.net/ajax/libs/[^\"'\'''\'']*" ./web | grep -v "monaco-editor" | while read uri; do name=$${uri#https://cdn.bootcdn.net/ajax/}; mkdir -p "web/$$(dirname $${name})"; curl -s "https://cdn.bootcdn.net/ajax/$$name" -o "web/$${name}"; done'
	@bash -c 'version=`grep -horP "monaco-editor/[\d\.]+" ./web | uniq | cut -d "/" -f 2`; curl -sOL "https://registry.npm.taobao.org/monaco-editor/-/monaco-editor-$$version.tgz"; mkdir -p "./web/libs/monaco-editor/$$version/"; tar -zxf monaco-editor-$$version.tgz -C "./web/libs/monaco-editor/$$version/" --strip-components 1 "package/min"; rm monaco-editor-$$version.tgz'
	@sed -i 's#https://cdn.bootcdn.net/ajax/libs#/libs#g' web/*.html
	@go build -ldflags "-s -w" .
	@sed -i 's#/libs#https://cdn.bootcdn.net/ajax/libs#g' web/*.html
