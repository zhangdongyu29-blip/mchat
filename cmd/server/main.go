package main

import (
	"log"

	"github.com/zhangdongyu29-blip/mchat/internal/config"
	"github.com/zhangdongyu29-blip/mchat/internal/db"
	"github.com/zhangdongyu29-blip/mchat/internal/server"
)

func main() {
	cfg := config.Load()

	if err := db.Connect(cfg); err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	r := server.NewRouter(cfg)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
