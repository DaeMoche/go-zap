package initialize

import (
	"flag"
	"fmt"
	"go-zap/common"
	cfg "go-zap/config"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func Viper() error {
	in := getConfigFile()

	v := viper.New()
	v.SetConfigFile(in)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败：%v \n", err)
	}

	v.WatchConfig()
	v.OnConfigChange(func(event fsnotify.Event) {
		fmt.Println("config file changed:", event.Name)
		if err := v.Unmarshal(&common.Config); err != nil {
			fmt.Println(err)
		}
	})

	// 将配置赋值给全局变量
	if err := v.Unmarshal(&common.Config); err != nil {
		fmt.Println(err)
		return fmt.Errorf("fatal error unmarshal config: %v", err)
	}

	common.Viper = v
	return nil
}

func getConfigFile() (config string) {

	flag.StringVar(&config, "c", "", "选择配置文件")
	flag.Parse()
	if config != "" {
		return
	}

	if env := os.Getenv(cfg.ConfigType); env != "" {
		config = env
		return
	}

	switch cfg.Mode() {
	case cfg.DebugMode:
		config = cfg.DebugConfigure
	case cfg.ReleaseMode:
		config = cfg.ReleaseConfigure
	case cfg.TestMode:
		config = cfg.TestConfigure
	default:
		config = cfg.DefaultConfigure
	}

	if _, err := os.Stat(config); err != nil || os.IsNotExist(err) {
		config = cfg.DefaultConfigure
		fmt.Printf("配置文件路径不存在, 使用默认配置文件路径: %s\n", config)
	}

	return
}
