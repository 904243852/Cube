run: # 运行
	@go run . -n 8

watch: # 监听当前目录下的相关文件变动，实时编译、运行
	@gowatch -o ./cube

build: clean # 构建
	@go build .

clean:
	@rm -rf cube *.log

tidy: # 安装依赖、删除 go.mod、go.sum 中的无用依赖
	@go mod tidy

crt:
	@ls | grep -P 'ca\.(key|crt)' > /dev/null && echo 'The ca.key or ca.crt already existed, skip.' || openssl req -new -days 3650 -x509 -nodes -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=Sunke Root CA" -keyout ca.key -out ca.crt
	@bash -c 'openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=localhost" -out server.csr && openssl x509 -sha256 -req -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt'

ccrt:
	@openssl req -newkey rsa:2048 -nodes -keyout client.key -subj "/C=CN/ST=BJ/L=BJ/O=/CN=" -out client.csr && openssl x509 -sha256 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

kill:
	@ps -ef | grep -P "/cube|/gowatch" | grep -v "grep" | awk '{print $$2}' | xargs kill -9
