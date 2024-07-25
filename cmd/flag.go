package cmd

import (
	"github.com/ontio/layer2deploy/layer2config"
	"github.com/urfave/cli"
	"strings"
)

var (
	LogLevelFlag = cli.UintFlag{
		Name:  "loglevel",
		Usage: "Set the log level to `<level>` (0~6). 0:Trace 1:Debug 2:Info 3:Warn 4:Error 5:Fatal 6:MaxLevel",
		Value: uint(layer2config.DEFAULT_LOG_LEVEL),
	}
	ConfigfileFlag = cli.StringFlag{
		Name:   "config",
		Usage:  "specify configfile",
		Value:  "config.json",
		EnvVar: "CONFIG_FILE",
	}
)

func GetFlagName(flag cli.Flag) string {
	name := flag.GetName()
	if name == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(name, ",")[0])
}
