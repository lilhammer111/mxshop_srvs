package main

import (
	"flag"
	"fmt"
	capi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"log"
	"mxshop_srvs/goods_srv/global"
	"mxshop_srvs/goods_srv/handler"
	"mxshop_srvs/goods_srv/initialize"
	"mxshop_srvs/goods_srv/proto"
	util "mxshop_srvs/goods_srv/utils"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 要带杠 -i 127.0.0.1 -p 50051
	IP := flag.String("i", "0.0.0.0", "ip地址")
	// fix default value as 50051 for testing
	Port := flag.Int("p", 50051, "端口号")
	flag.Parse()

	// 初始化配置
	initialize.Logger()

	initialize.Config()

	zap.S().Infof("server config is %+v", global.ServerConfig)

	initialize.DB()

	if *Port == 0 {
		*Port, _ = util.GetFreePort()
	}

	//fmt.Println("ip:", *IP)
	zap.S().Info("ip: ", *IP)
	//fmt.Println("port: ", *Port)
	zap.S().Info("port: ", *Port)

	server := grpc.NewServer()
	proto.RegisterGoodsServer(server, &handler.GoodsServer{})
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *IP, *Port))
	if err != nil {
		log.Fatalln("failed to listen: " + err.Error())
	}

	// 注册服务健康检查
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())

	//服务注册
	cfg := capi.DefaultConfig()
	cfg.Address = fmt.Sprintf("%s:%d", global.ServerConfig.ConsulInfo.Host,
		global.ServerConfig.ConsulInfo.Port)

	client, err := capi.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	//生成对应的检查对象
	check := &capi.AgentServiceCheck{
		GRPC:                           fmt.Sprintf("192.168.1.5:%d", *Port),
		Timeout:                        "5s",
		Interval:                       "5s",
		DeregisterCriticalServiceAfter: "15s",
	}

	//生成注册对象
	registration := new(capi.AgentServiceRegistration)
	registration.Name = global.ServerConfig.Name
	uuidStr, _ := uuid.GenerateUUID()
	serviceID := fmt.Sprintf("%s", uuidStr)
	registration.ID = serviceID
	//registration.ID = global.ServerConfig.Name
	registration.Port = *Port
	registration.Tags = []string{"imooc", "bobby", "user", "srv"}
	registration.Address = "192.168.1.5"
	registration.Check = check
	//1. 如何启动两个服务
	//2. 即使我能够通过终端启动两个服务，但是注册到consul中的时候也会被覆盖
	err = client.Agent().ServiceRegister(registration)
	if err != nil {
		panic(err)
	}

	go func() {
		err = server.Serve(lis)
		if err != nil {
			log.Fatalln("failed to start grpc: " + err.Error())
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	if err = client.Agent().ServiceDeregister(serviceID); err != nil {
		zap.S().Info("注销失败")
	}
	zap.S().Info("注销成功")
}
