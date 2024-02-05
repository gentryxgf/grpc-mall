package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf = new(SrvConfig)

type SrvConfig struct {
	Name    string `mapstructure:"name"`
	Mode    string `mapstructure:"mode"`
	Version string `mapstructure:"version"`

	StartTime string `mapstructure:"start_time"`
	MachineId int    `mapstructure:"machine_id"`

	Ip       string `mapstructure:"ip"`
	RpcPort  int    `mapstructure:"rpcPort"`
	HttpPort int    `mapstructure:"httpPort"`

	*MySQLConfig  `mapstructure:"mysql"`
	*LogConfig    `mapstructure:"log"`
	*ConsulConfig `mapstructure:"consul"`
}

type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DB           string `mapstructure:"dbname"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

type ConsulConfig struct {
	Address string `mapstructure:"address"`
}

func Init(filepath string) (err error) {
	//指定配置文件路径
	viper.SetConfigFile(filepath)

	//读取配置信息
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config.viper.ReadInConfig failed, err:%v\n", err)
		return
	}

	//把读取的配置信息反序列化到Conf变量中
	if err := viper.Unmarshal(Conf); err != nil {
		fmt.Printf("config.viper.Unmarshal failed, err:%v\n", err)
	}

	//配置文件监听
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Printf("配置文件修改了")
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("config.viper.Unmarshal failed, err:%v\n", err)
		}
	})
	return

}
