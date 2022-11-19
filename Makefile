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

kill:
	@ps -ef | grep -P "/cube|/gowatch" | grep -v "grep" | awk '{print $$2}' | xargs kill -9
