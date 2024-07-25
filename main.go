package main

import (
	"fmt"
	"github.com/howeyc/gopass"
	"github.com/urfave/cli"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ontio/layer2deploy/cmd"
	"github.com/ontio/layer2deploy/core"
	"github.com/ontio/layer2deploy/layer2config"
	"github.com/ontio/layer2deploy/restful"
	"github.com/ontio/ontology/common/log"
)

func setupAPP() *cli.App {
	app := cli.NewApp()
	app.Usage = "layer2deploy CLI"
	app.Action = startLayer2Deploy
	app.Copyright = "Copyright in 2020 The Ontology Authors"
	app.Flags = []cli.Flag{
		cmd.LogLevelFlag,
		cmd.ConfigfileFlag,
	}
	app.Before = func(context *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	return app
}

func main() {
	if err := setupAPP().Run(os.Args); err != nil {
		log.Errorf("App Get Eorror: %s", err)
		os.Exit(1)
	}
}

func startLayer2Deploy(ctx *cli.Context) {
	initLog(ctx)

	cfg, err := cmd.SetLayer2Config(ctx)
	if err != nil {
		log.Errorf("startLayer2Deploy N.0 %s", err)
		return
	}

	log.Infof("startLayer2Deploy Y.0 Config: %v", *cfg)

	sendService := core.NewSendService(cfg)
	core.DefSendService = sendService
	core.DefVerifyService = core.NewVerifyService(cfg)

	core.DefVerifyService.Cfg = cfg
	restful.NewRouter()
	startServer(cfg)

	if cfg.EnableSendService {
		log.Infof("SendService Enabled")
		go sendService.RepeantSendLogToChain()
	} else {
		log.Infof("SendService Disabled")
	}

	waitToExit(sendService)
}

func startServer(config *layer2config.Config) {
	router := restful.NewRouter()
	go router.Run(":" + config.RestPort)
}

func initLog(ctx *cli.Context) {
	logLevel := ctx.GlobalInt(cmd.GetFlagName(cmd.LogLevelFlag))
	log.InitLog(logLevel, log.Stdout)
}

// GetPassword gets password from user input
func getDBPassword() ([]byte, error) {
	fmt.Printf("DB Password:")
	passwd, err := gopass.GetPasswd()
	if err != nil {
		return nil, err
	}
	return passwd, nil
}

func waitToExit(s *core.SendService) {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func(s *core.SendService) {
		for sig := range sc {
			log.Infof("saga server received exit signal: %s.", sig.String())
			//s.QuitS <- true
			//s.Wg.Wait()
			close(exit)
			break
		}
	}(s)
	<-exit
}
