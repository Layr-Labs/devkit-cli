package main

import (
	"devkit-cli/pkg/plugin"
	"github.com/urfave/cli/v2"
)

type TestPlugin struct {
}

func (p *TestPlugin) Version() string {
	return "v1.0.0"
}

func (p *TestPlugin) Name() string {
	return "TestPlugin"
}

func (p *TestPlugin) GetCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "test",
			Usage: "Test command from TestPlugin",
			Action: func(c *cli.Context) error {
				println("Test command executed")
				return nil
			},
		},
	}
}

func GetPlugin() plugin.IPlugin {
	return &TestPlugin{}
}
