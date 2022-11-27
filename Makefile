run: # 运行
	@go run .

watch: # 监听当前目录下的相关文件变动，实时编译、运行
	@gowatch -o ./cube

build: clean # 构建
	@go build .

clean:
	@rm -rf cube

tidy: # 安装依赖、删除 go.mod、go.sum 中的无用依赖
	@go mod tidy

crt:
	@ls | grep server.crt > /dev/null && echo 'server.crt already existed.' || openssl req -new -days 3650 -x509 -nodes -subj "/CN=127.0.0.1" -keyout server.key -out server.crt

kill:
	@ps -ef | grep -P "/cube|/gowatch" | grep -v "grep" | awk '{print $$2}' | xargs kill -9
