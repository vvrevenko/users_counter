package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	app, err := NewApp()
	Die(err)
	ctx := context.Background()
	Die(app.Run(ctx))
}

func Die(err error) {
	if err == nil {
		return
	}
	log.Println(err.Error())
	os.Exit(1)
}