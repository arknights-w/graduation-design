# AIRFONE

airfone 传话筒

Dynamic Naming and Configuration Service [ 动态命名和配置服务 ]

这是一个服务注册/发现中心，采用 DDD 架构进行代码编写，目的是为了让服务中心中的组件具有高延展性，可插拔性。

## 下载 Kratos
```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```
## 创建一个服务
```
# 创建一个模板项目
kratos new server

cd server
# 添加 proto 模板
kratos proto add api/server/server.proto
# 生成 proto 代码
kratos proto client api/server/server.proto
# 通过proto文件生成服务的源代码
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```
## 通过Makefile生成其他辅助文件
```
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
# Generate all files
make all
```
## 依赖注入 (wire)
```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Docker
```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/your/configs>:/data/conf <your-docker-image-name>
```

