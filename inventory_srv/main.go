package main

import (
	"flag"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/hashicorp/go-uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"log"
	"mxshop_srvs/inventory_srv/global"
	"mxshop_srvs/inventory_srv/handler"
	"mxshop_srvs/inventory_srv/initialize"
	"mxshop_srvs/inventory_srv/proto"
	util "mxshop_srvs/inventory_srv/utils"
	"mxshop_srvs/inventory_srv/utils/register/consul"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 要带杠 -i 127.0.0.1 -p 50051
	IP := flag.String("i", "0.0.0.0", "ip地址")
	// fix default value as 50051 for testing
	Port := flag.Int("p", 50053, "端口号")
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
	proto.RegisterInventoryServer(server, &handler.InventoryServer{})
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *IP, *Port))
	if err != nil {
		log.Fatalln("failed to listen: " + err.Error())
	}

	// 注册服务健康检查
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
	//服务注册
	registryClient := consul.NewRegistryClient(global.ServerConfig.ConsulInfo.Host, global.ServerConfig.ConsulInfo.Port)
	serviceID, _ := uuid.GenerateUUID()
	err = registryClient.Register(global.ServerConfig.Host, *Port, global.ServerConfig.Name, global.ServerConfig.Tags, serviceID)
	if err != nil {
		zap.S().Panic("fail to register inventory-srv service", err.Error())
	}

	// start service
	go func() {
		err = server.Serve(lis)
		if err != nil {
			log.Fatalln("failed to start grpc: " + err.Error())
		}
	}()

	// Listen to the inventory return topic
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{"127.0.0.1:9876"}),
		consumer.WithGroupName("mxshop-inventory"))

	if err = c.Subscribe("order_reback", consumer.MessageSelector{}, handler.AutoReback); err != nil {
		fmt.Println("读取消息失败")
	}
	_ = c.Start()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	_ = c.Shutdown()

	err = registryClient.DeRegister(serviceID)
	if err != nil {
		zap.S().Panic("fail to deregister", err.Error())
	} else {
		zap.S().Info("succeed to deregister")
	}
}
