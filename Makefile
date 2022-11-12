run: # 运行
	@go run cube.go

watch: # 监听当前目录下的相关文件变动，实时编译、运行
	@gowatch -o ./cube

build: clean # 构建
	@go build cube.go

clean:
	@rm -rf cube

tidy: # 删除无用依赖，此操作将会更新 go.mod、go.sum 文件
	@go mod tidy