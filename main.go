package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"jahaninia.ir/cloud/asterisk/pbx/jolAmi"
	"jahaninia.ir/cloud/asterisk/pbx/jolConfigurtion"
)

var (
	configSetting jolConfigurtion.Configurtion
	err           error
)

func main() {
	log.Println("Start-V2")
	var cfgPath = ""
	if e := os.Getenv("CFG_PATH"); len(e) > 0 {
		cfgPath = e
	} else {
		fmt.Println("Not set CFG_PATH")
		os.Exit(1)
	}
	configSetting, err = jolConfigurtion.LoadConfigFile(cfgPath)
	if err != nil {
		log.Panic(err)
	}
	fmt.Sprintf("%s", configSetting)
	ami := jolAmi.NewJolAmi(configSetting.Asterisk.HostAmi,
		configSetting.Asterisk.PortAmi,
		configSetting.Asterisk.UsernameAmi,
		configSetting.Asterisk.PasswordAmi,
		configSetting.Token,
		configSetting.UrlAnswer,
		configSetting.UrlReject,
		configSetting.UrlEndCall,
		configSetting.UrlCall,
		configSetting.Length,
		configSetting.Pool,
		configSetting.Debug)

	var wg sync.WaitGroup
	wg.Add(1)
	go ami.Run(&wg)
	wg.Wait()
	log.Println("End")
	os.Exit(0)
}
