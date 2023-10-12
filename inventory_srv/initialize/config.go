package initialize

import (
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"mxshop_srvs/inventory_srv/global"
)

func GetEnvInfo(env string) bool {
	viper.AutomaticEnv()
	return viper.GetBool(env)
}

// 从配置文件中读取配置
func Config() {
	debug := GetEnvInfo("MXSHOP_DEBUG")
	zap.S().Infof("MXSHOP_DEBUG IS %t", debug)
	configFilePrefix := "config"
	configFileName := fmt.Sprintf("inventory_srv/%s-pro.yaml", configFilePrefix)
	//zap.S().Info(configFileName)
	if debug {
		configFileName = fmt.Sprintf("inventory_srv/%s-debug.yaml", configFilePrefix)
	}

	v := viper.New()
	v.SetConfigFile(configFileName)

	if err := v.ReadInConfig(); err != nil {
		zap.S().Fatal("读取配置文件错误： ", err)
	}

	if err := v.Unmarshal(&global.NacosConfig); err != nil {
		zap.S().Fatal("序列化配置文件错误", err)
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: global.NacosConfig.Host,
			Port:   uint64(global.NacosConfig.Port),
		},
	}
	clientConfig := constant.ClientConfig{
		NamespaceId:         global.NacosConfig.Namespace, // 如果需要支持多namespace，我们可以创建多个client,它们有不同的NamespaceId。当namespace是public时，此处填空字符串。
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "tmp/nacos/log",
		CacheDir:            "tmp/nacos/cache",
		LogLevel:            "debug",
	}

	// 创建动态配置客户端的另一种方式 (推荐)
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)

	if err != nil {
		panic(err)
	}

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: global.NacosConfig.DataId,
		Group:  global.NacosConfig.Group})

	if err != nil {
		panic(err)
	}

	//fmt.Println(content)

	err = json.Unmarshal([]byte(content), &global.ServerConfig)

	if err != nil {
		zap.S().Fatalf("读取nacos配置失败： %s\n", err)
	}

}
