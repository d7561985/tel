package cli

import (
	"fmt"

	"github.com/d7561985/tel/example/demo/client/v2/pkg/httptest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/mgr"
	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	mgsThreads               = "controller_threads"
	defaultControllerThreads = 40
)

func Controller() *cli.Command {
	return &cli.Command{
		Name:    "controller",
		Aliases: []string{"c", "mgr"},
		Usage:   "kind of worker which examine telemetry and perform some requests to http server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: httpServer, Value: defaultHttpServer},
			&cli.IntFlag{Name: mgsThreads, Value: defaultControllerThreads},
		},
		Action: func(ctx *cli.Context) error {
			cfg := tel.GetConfigFromEnv()
			cfg.LogEncode = "console"
			cfg.Namespace = "TEST"
			cfg.Service = "CONTROLLER"
			cfg.MonitorConfig.Enable = false

			t, closer := tel.New(ctx.Context, cfg)
			defer closer()

			t.Info(cfg.Service, tel.String("collector", cfg.Addr), tel.Int("threads", ctx.Int(mgsThreads)))
			if ctx.Int(mgsThreads) <= 0 {
				return errors.WithStack(fmt.Errorf("mgsThreads <= 0"))
			}

			hClt, err := httptest.NewClient("http://" + ctx.String(httpServer))
			if err != nil {
				t.Fatal("http client", tel.Error(err))
			}

			// important add tel
			ccx := tel.WithContext(ctx.Context, t)
			err = mgr.New(t, hClt).Start(ccx, ctx.Int(mgsThreads))
			return errors.WithStack(err)
		},
	}
}
