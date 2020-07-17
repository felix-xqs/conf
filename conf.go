package xconf

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/shima-park/agollo"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"path"
	"strings"
	"time"
	"github.com/felix-xqs/conf/remote"
)

type ConfigLogLevel int

const (
	LevelTrace ConfigLogLevel = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
	LevelFatal
)

var SupportedExtensions = []string{"json", "toml", "yaml", "yml", "properties", "props", "prop", "hcl"}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
func SetLogLevel(level ConfigLogLevel) {
	// 打印 viper 配置文件加载过程
	jwwLogLevel := jww.Threshold(level)
	jww.SetStdoutThreshold(jwwLogLevel)
}

func init() {
	SetLogLevel(LevelInfo)
}

type XParam struct {
	AppId     string // 对应于apollo 的 appId
	Cluster   string // 对应apollo集群，默认为dev
	Namespace string // 对应于apollo 的 namespace， 默认为application
	Ip        string // 为apollo config server 的 SLB地址加端口
	// 和NewWithParam的bindObject参数一起使用，如果bindObject为nil可以忽略该参数
	// apollo配置项的key，内容为conf.yaml
	//ConfigKey      string
	ConfigType string // 默认yaml
	// 将当前AppId下的所有配置项保存在本地，此处设置保存文件路径
	// 如：/data/server/xng/xxx/application.json, 默认为application.json
	BackupFileName string
}

func resetParam(param *XParam) {
	configType := param.ConfigType
	if configType == "" {
		configType = "yaml"
		param.ConfigType = configType
	}

	if param.Namespace == "" {
		param.Namespace = param.AppId
	}

	// 依赖的agollo接口设置不合理，对于properties之外的支持有问题，这块暂时先这样处理.
	param.Namespace = fmt.Sprintf("%s.%s", param.Namespace, param.ConfigType)

	if param.BackupFileName == "" {
		param.BackupFileName = "application.json"
	}

	if param.Cluster == "" {
		param.Cluster = "default"
	}
}

func NewWithParam(param XParam) (conf *XConfig, err error) {
	if param.Ip == "" {
		return nil, errors.New("Xconf param miss Ip.")
	}
	if param.AppId == "" {
		return nil, errors.New("Xconf param miss AppId.")
	}
	resetParam(&param)

	namespace := param.Namespace
	configType := param.ConfigType
	backupFileName := param.BackupFileName
	cluster := param.Cluster

	defaultApolloOptions := []agollo.Option{
		agollo.DefaultNamespace(namespace),
		agollo.PreloadNamespaces(namespace),
		agollo.Cluster(cluster),
		agollo.BackupFile(backupFileName),
		agollo.AutoFetchOnCacheMiss(),
		agollo.FailTolerantOnBackupExists(),
	}

	//if bindObject != nil {
	//	xViper := viper.New()
	//
	//	varApollo, err := agollo.New(param.Ip, param.AppId, defaultApolloOptions...)
	//	if err != nil {
	//		return nil, err
	//	}
	//	conf = &XConfig{xViper, varApollo, &param, bindObject}
	//	xViper.SetConfigType(configType) // 默认走yaml
	//
	//	conf.startApolloForBindObject()
	//	conf.bindRemoteObject()
	//}
	remote.SetAgolloOptions(defaultApolloOptions...)
	remote.SetConfigType(configType, namespace)
	remote.SetAppID(param.AppId)
	xViper := viper.New()
	xViper.SetConfigType(configType)
	conf = &XConfig{xViper, nil, &param}
	err = xViper.AddRemoteProvider("apollo", param.Ip, namespace)
	err = xViper.ReadRemoteConfig()
	go xViper.WatchRemoteConfigOnChannel()

	return
}

func New() *XConfig {
	xViper := viper.New()
	return &XConfig{xViper, nil, nil}
}

type XConfig struct {
	viper   *viper.Viper
	xApollo agollo.Agollo
	xParam  *XParam
}

//
//func (conf *XConfig) bindRemoteObject() {
//	if conf.xApollo == nil {
//		return
//	}
//	str := conf.xApollo.Get(conf.xParam.ConfigKey, agollo.WithNamespace(conf.xParam.Namespace))
//	if str != "" {
//		err := conf.ReadConfig(str)
//		if err == nil {
//			_ = conf.Unmarshal(conf.bindObject)
//		}
//	}
//}
//
//func (conf *XConfig) startApolloForBindObject() {
//	if conf.xApollo == nil {
//		return
//	}
//	// 如果想监听并同步服务器配置变化，启动apollo长轮训
//	// 返回一个期间发生错误的error channel,按照需要去处理
//	errorCh := conf.xApollo.Start()
//
//	// 监听apollo配置更改事件
//	// 返回namespace和其变化前后的配置,以及可能出现的error
//	watchCh := conf.xApollo.Watch()
//
//	go func() {
//		for {
//			select {
//			case err := <-errorCh:
//				fmt.Println("Error:", err)
//			case resp := <-watchCh:
//				newValue, ok1 := resp.NewValue[conf.xParam.ConfigKey]
//				oldValue, ok2 := resp.OldValue[conf.xParam.ConfigKey]
//				if ok1 && ok2 {
//					if newValue != oldValue {
//						conf.bindRemoteObject()
//					}
//				}
//
//			}
//		}
//	}()
//
//}

func (conf *XConfig) ReadConfig(content string) error {
	var contentBytes = []byte(content)
	return conf.viper.ReadConfig(bytes.NewBuffer(contentBytes))
}

func (conf *XConfig) Unmarshal(rawVal interface{}) error {
	return conf.viper.Unmarshal(rawVal)
}

func (conf *XConfig) SetConfigName(in string) {
	conf.viper.SetConfigName(in)
}

func (conf *XConfig) SetConfigType(in string) {
	conf.viper.SetConfigType(in)
}
func (conf *XConfig) AddConfigPath(in string) {
	conf.viper.AddConfigPath(in)
}

func (conf *XConfig) ReadInConfig() error {
	return conf.viper.ReadInConfig()
}

func (conf *XConfig) Get(key string) interface{} {
	return conf.viper.Get(key)
}

func (conf *XConfig) GetString(key string) string {
	return conf.viper.GetString(key)
}

func (conf *XConfig) GetBool(key string) bool {
	return conf.viper.GetBool(key)
}

func (conf *XConfig) GetInt(key string) int {
	return conf.viper.GetInt(key)
}

func (conf *XConfig) GetInt32(key string) int32 {
	return conf.viper.GetInt32(key)
}

func (conf *XConfig) GetInt64(key string) int64 {
	return conf.viper.GetInt64(key)
}

func (conf *XConfig) GetFloat64(key string) float64 {
	return conf.viper.GetFloat64(key)
}

func (conf *XConfig) GetTime(key string) time.Time {
	return conf.viper.GetTime(key)
}

func (conf *XConfig) GetStringMap(key string) map[string]interface{} {
	return conf.viper.GetStringMap(key)
}

func (conf *XConfig) GetStringMapString(key string) map[string]string {
	return conf.viper.GetStringMapString(key)
}

func (conf *XConfig) AllSettings() map[string]interface{} {
	return conf.viper.AllSettings()
}

// confPath 配置文件路径  如：/data/server/pro/conf.yaml
// rawVal 配置文件映射的对象
func LoadConfig(confPath string, rawVal interface{}) (err error) {
	confPath = strings.Replace(confPath, "\\", "/", -1)
	fileDir := path.Dir(confPath)
	fileFullName := path.Base(confPath)
	fileExtension := path.Ext(fileFullName)
	if fileExtension == "" {
		fileExtension = ".yaml"
	}
	fileType := fileExtension[1:]
	if !stringInSlice(fileType, SupportedExtensions) {
		return viper.UnsupportedConfigError(fileType)
	}
	nameLen := len(fileFullName) - len(fileExtension)
	configName := fileFullName[:nameLen]
	conf := New()
	conf.SetConfigName(configName) // 配置文件的名字
	conf.SetConfigType(fileType)   // 配置文件的类型
	conf.AddConfigPath(fileDir)    // 配置文件的路径

	if err := conf.ReadInConfig(); err != nil {
		return err
	}

	if err := conf.Unmarshal(rawVal); err != nil {
		panic(fmt.Errorf("unable to decode into struct：  %s \n", err))
	}

	return nil
}

// xConfPath 配置文件路径  如：./xconf.yaml
// rawVal 配置文件映射的对象
func NewWithConfigFile(xConfPath string, rawVal interface{}) (xConfig *XConfig, err error) {
	xParam := XParam{}
	err = LoadConfig(xConfPath, &xParam)
	if err != nil {
		return
	}
	xConfig, err = NewWithParam(xParam)
	if err != nil {
		return
	}
	if rawVal != nil {
		err = xConfig.Unmarshal(rawVal)
	}
	return
}
