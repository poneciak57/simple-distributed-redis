package main

import (
	"fmt"
	"main/src/config"
	"main/src/service"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	config, err := config.LoadConfig("config/default.yaml")
	if err != nil {
		panic(err)
	}
	storageService := service.NewStorageService(config)
	redisService := service.NewRedisServices(storageService, config)
	tcpManager := service.NewTcpServiceManager(redisService)
	if err := tcpManager.Start(); err != nil {
		panic(err)
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")
	if err := tcpManager.Stop(); err != nil {
		fmt.Printf("Error stopping server: %v\n", err)
	}
}
