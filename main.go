package main

import (
	"main/src/config"
	"main/src/service"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log := config.NewLogger("Main")

	cfg, err := config.LoadConfig("config/default.yaml")
	if err != nil {
		panic(err)
	}

	level, err := config.ParseLevel(cfg.Logger.Level)
	if err == nil {
		config.SetDefaultLevel(level)
		log.SetLevel(level)
	}

	storageService := service.NewStorageService(cfg, log.Named("StorageService"))
	redisService := service.NewRedisServices(storageService, cfg, log.Named("RedisService"))
	tcpManager := service.NewTcpServiceManager(redisService)
	if err := tcpManager.Start(); err != nil {
		panic(err)
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	if err := tcpManager.Stop(); err != nil {
		log.Error("Error stopping server: %v", err)
	}
}
