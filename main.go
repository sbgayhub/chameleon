package main

import (
	"embed"

	"github.com/sbgayhub/chameleon/backend/application"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.NewApp()
	err := wails.Run(&options.App{
		Title:     "Chameleon",
		Width:     950,
		Height:    600,
		MinWidth:  950,
		MinHeight: 550,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless: true,
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
			app.CertMgr,
			app.ConfigMgr,
			app.ChannelMgr,
			app.StatsMgr,
		},
		StartHidden: app.ConfigMgr.GetConfig().General.StartMinimized,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "2F4F1516-F78B-4921-9734-B3A5B65F8B6E",
			OnSecondInstanceLaunch: app.OnSecondInstanceLaunch,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
