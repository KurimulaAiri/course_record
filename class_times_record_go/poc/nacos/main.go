// Package main Nacos 服务注册互通验证 PoC
//
// 验证目标：
//  1. Go nacos-sdk-go 能连接现有 Nacos (nacos.kurimula-airi.top:8848)
//  2. 能注册 Go 服务到现有 namespace (course-record)
//  3. 能发现现有 Java 服务 (cr-gateway, cr-auth-service 等)
//  4. 验证 Java/Go 跨语言服务发现互通
//
// 运行：go run ./poc/nacos/main.go
package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// ============================================================
// Nacos 连接配置（与 Java 侧一致）
// ============================================================

// Nacos 服务器地址
const nacosServerHost = "nacos.kurimula-airi.top"
const nacosServerPort = 8848

// Nacos namespace（与 Java 侧 course-record 一致）
const nacosNamespace = "course-record"

// PoC 服务配置
const pocServiceName = "cr-go-poc"
const pocServiceIP = "127.0.0.1"
const pocServicePort = 18080

// ============================================================
// main 函数
// ============================================================

func main() {
	fmt.Println("========== go-zero + Nacos 互通 PoC ==========")
	fmt.Printf("Nacos: %s:%d, Namespace: %s\n", nacosServerHost, nacosServerPort, nacosNamespace)
	fmt.Println()

	// 1. 创建 Nacos naming 客户端
	namingClient, err := createNacosNamingClient()
	if err != nil {
		fmt.Printf("❌ 创建 Nacos 客户端失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Nacos 客户端连接成功")

	// 2. 查询现有 Java 服务列表
	fmt.Println("\n---------- 发现现有服务 ----------")
	discoverServices(namingClient)

	// 3. 注册 Go PoC 服务
	fmt.Println("\n---------- 注册 Go PoC 服务 ----------")
	registerService(namingClient)

	// 4. 查询刚注册的 Go 服务
	fmt.Println("\n---------- 查询 Go PoC 服务实例 ----------")
	queryRegisteredService(namingClient)

	// 5. 验证 Java 服务实例可发现
	fmt.Println("\n---------- 查询 Java 服务实例 ----------")
	queryJavaServices(namingClient)

	// 6. 注销服务
	fmt.Println("\n---------- 注销 Go PoC 服务 ----------")
	deregisterService(namingClient)

	fmt.Println("\n========== PoC 验证完成 ==========")
}

// createNacosNamingClient 创建 Nacos naming 客户端
// 使用与 Java 侧相同的 serverConfig 和 namespace
//
// 返回：naming_client.INamingClient
func createNacosNamingClient() (naming_client.INamingClient, error) {
	// Nacos 服务器配置
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: nacosServerHost,
			Port:   nacosServerPort,
			// Java 侧 nacos.kurimula-airi.top 使用 HTTPS 反代，此处先尝试 HTTP
			Scheme: "http",
			// 上下文路径（标准 Nacos 为 /nacos）
			ContextPath: "/nacos",
		},
	}

	// 客户端配置
	clientConfig := constant.ClientConfig{
		// namespace ID（与 Java 侧 course-record 一致）
		NamespaceId: nacosNamespace,
		// 超时设置
		TimeoutMs:            5000,
		NotLoadCacheAtStart:  true,
		// 心跳间隔（毫秒）
		BeatInterval:         1000,
		// 日志级别
		LogLevel:             "warn",
		// 日志目录
		LogDir:               ".temp/nacos-log",
		// 缓存目录
		CacheDir:             ".temp/nacos-cache",
	}

	// 创建 naming 客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建 naming 客户端失败: %w", err)
	}

	return namingClient, nil
}

// discoverServices 发现 Nacos 中所有已注册的服务
// 验证 Go 能看到 Java 注册的服务
func discoverServices(namingClient naming_client.INamingClient) {
	// 查询 namespace 下的服务列表
	services, err := namingClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		NameSpace: nacosNamespace,
		PageNo:    1,
		PageSize:  50,
	})
	if err != nil {
		fmt.Printf("❌ 查询服务列表失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 发现 %d 个服务:\n", services.Count)
	for _, svc := range services.Doms {
		fmt.Printf("   - %s\n", svc)
	}
}

// registerService 注册 Go PoC 服务到 Nacos
// 模拟 go-zero 微服务注册行为
func registerService(namingClient naming_client.INamingClient) {
	// 获取本机 IP（兜底用 pocServiceIP）
	ip := getLocalIP()
	if ip == "" {
		ip = pocServiceIP
	}

	success, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        uint64(pocServicePort),
		ServiceName: pocServiceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true, // 临时实例，心跳保活
		Metadata: map[string]string{
			"language":   "go",
			"framework":  "go-zero",
			"version":    "poc-1.0",
			"registered": time.Now().Format(time.RFC3339),
		},
		ClusterName: "DEFAULT",
		GroupName:   "DEFAULT_GROUP",
	})

	if err != nil {
		fmt.Printf("❌ 服务注册失败: %v\n", err)
		return
	}
	if success {
		fmt.Printf("✅ Go PoC 服务注册成功: %s:%d (IP=%s)\n", pocServiceName, pocServicePort, ip)
	} else {
		fmt.Printf("❌ 服务注册返回 false\n")
	}
}

// queryRegisteredService 查询刚注册的 Go PoC 服务实例
// 验证注册生效
func queryRegisteredService(namingClient naming_client.INamingClient) {
	instances, err := namingClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: pocServiceName,
		GroupName:   "DEFAULT_GROUP",
		HealthyOnly: true,
	})
	if err != nil {
		fmt.Printf("❌ 查询 Go PoC 服务实例失败: %v\n", err)
		return
	}

	if len(instances) == 0 {
		fmt.Println("⚠️  未找到 Go PoC 服务实例（可能心跳未生效）")
		return
	}

	for _, inst := range instances {
		fmt.Printf("✅ 找到实例: IP=%s, Port=%d, Healthy=%v, Meta=%v\n",
			inst.Ip, inst.Port, inst.Healthy, inst.Metadata)
	}
}

// queryJavaServices 查询现有 Java 服务的实例
// 验证 Go 能发现 Java 注册的微服务实例
func queryJavaServices(namingClient naming_client.INamingClient) {
	// Java 侧注册的服务名（Spring Cloud Alibaba 默认用 spring.application.name）
	javaServices := []string{
		"cr-gateway",
		"cr-auth-service",
		"cr-business-service",
		"cr-admin-service",
	}

	found := 0
	for _, svc := range javaServices {
		instances, err := namingClient.SelectInstances(vo.SelectInstancesParam{
			ServiceName: svc,
			GroupName:   "DEFAULT_GROUP",
			HealthyOnly: false, // 包含不健康的实例
		})
		if err != nil {
			fmt.Printf("   ⚠️  %s: 查询失败 - %v\n", svc, err)
			continue
		}
		if len(instances) == 0 {
			fmt.Printf("   - %s: 无实例（服务可能未启动）\n", svc)
			continue
		}
		for _, inst := range instances {
			fmt.Printf("✅ %s -> %s:%d (healthy=%v)\n", svc, inst.Ip, inst.Port, inst.Healthy)
			found++
		}
	}

	if found > 0 {
		fmt.Printf("\n✅ 成功发现 %d 个 Java 服务实例，跨语言服务发现互通验证通过\n", found)
	} else {
		fmt.Println("\n⚠️  未发现 Java 服务实例（服务可能未启动或网络不通）")
	}
}

// deregisterService 注销 Go PoC 服务
func deregisterService(namingClient naming_client.INamingClient) {
	ip := getLocalIP()
	if ip == "" {
		ip = pocServiceIP
	}

	success, err := namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        uint64(pocServicePort),
		ServiceName: pocServiceName,
		Cluster:     "DEFAULT",
		GroupName:   "DEFAULT_GROUP",
		Ephemeral:   true,
	})

	if err != nil {
		fmt.Printf("❌ 服务注销失败: %v\n", err)
		return
	}
	if success {
		fmt.Printf("✅ Go PoC 服务已注销: %s\n", pocServiceName)
	}
}

// getLocalIP 获取本机局域网 IP
// 用于 Nacos 服务注册时填写真实 IP
//
// 返回：IP 字符串，失败返回空字符串
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}
