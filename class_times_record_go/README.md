# Course Record Go (迁移 PoC)

本目录为 Java 后端 (class_times_record_back) 迁移至 Go 语言的 PoC 验证项目。

## 迁移目标
- 性能/资源优化：Java 微服务单实例 300-500MB → Go 30-80MB
- 微服务架构保留：go-zero 框架 + Nacos 服务发现/配置中心

## PoC 验证项
1. **国密互通** (poc/crypto/)：SM2 解密 + SM3 加盐哈希，验证与 Java BouncyCastle 互通
2. **go-zero + Nacos** (poc/nacos/)：最小 go-zero 服务注册到现有 Nacos，验证与 Java 服务互通

## 运行
```bash
cd class_times_record_go
go test ./poc/crypto/... -v
go run ./poc/nacos/main.go
```
