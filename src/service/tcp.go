package service

import (
	"fmt"
	"main/src"
	"main/src/config"
	"net"
	"sync/atomic"
	"time"
)

type TcpMetadata struct {
	BaseMetadata
	Host string
	Port int
}

type TcpMetrics struct {
	BaseMetrics
	TotalRequests    int64
	OnMessageErrors  int64 // number of errors encountered in OnMessage
	InFlightRequests int64 // number of in-flight connections being processed
	RejectedRequests int64 // number of requests rejected due to queue full
	QueueSize        int64 // current number of items in queue
	ActiveWorkers    int64 // current number of active workers
}

type TcpServiceManager struct {
	service Service[net.Conn, BaseMetrics, TcpMetadata]
	metrics TcpMetrics
	logger  *config.Logger
	pool    *src.WorkerPool[net.Conn]
	timeout time.Duration
}

// NewTcpServiceManager creates a new TcpServiceManager that listens on the given address and port,
// Service should not close the net.Conn passed to OnMessage, TcpServiceManager will handle closing it.
func NewTcpServiceManager(service Service[net.Conn, BaseMetrics, TcpMetadata], cfg *config.Config, logger *config.Logger) *TcpServiceManager {
	manager := &TcpServiceManager{
		service: service,
		logger:  logger,
		timeout: time.Duration(cfg.Redis.Timeout) * time.Second,
	}

	manager.pool = src.NewWorkerPool(
		int64(cfg.Redis.MaxConnections),
		int64(cfg.Redis.MaxPending),
		int64(cfg.Redis.IdleConnectionsPerWorker),
		time.Duration(cfg.Redis.WorkerTTL)*time.Second,
		int64(cfg.Redis.BaseWorkers),
		func(conn net.Conn) error {
			defer conn.Close()
			atomic.AddInt64(&manager.metrics.InFlightRequests, 1)
			atomic.AddInt64(&manager.metrics.QueueSize, -1)
			defer atomic.AddInt64(&manager.metrics.InFlightRequests, -1)

			if err := service.OnMessage(conn); err != nil {
				atomic.AddInt64(&manager.metrics.OnMessageErrors, 1)
				return err
			}
			return nil
		},
		false,
		manager.logger,
	)

	return manager
}

func (s *TcpServiceManager) Stop() error {
	return nil
}

func (s *TcpServiceManager) loop(l net.Listener) {
	s.pool.Start()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}

		if s.timeout > 0 {
			conn.SetDeadline(time.Now().Add(s.timeout))
		}

		atomic.AddInt64(&s.metrics.TotalRequests, 1)

		if err := s.pool.Put(conn); err != nil {
			atomic.AddInt64(&s.metrics.RejectedRequests, 1)
			s.logger.Warn("Connection rejected: %v", err)
			conn.Close()
		} else {
			atomic.AddInt64(&s.metrics.QueueSize, 1)
		}
	}
}

func (s *TcpServiceManager) Start() error {
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", s.service.Metadata().Host, s.service.Metadata().Port))
	if err != nil {
		return err
	}
	go s.loop(l)
	return nil
}

func (s *TcpServiceManager) Metrics() TcpMetrics {
	service_m := s.service.Metrics()
	m := TcpMetrics{}
	m.IsHealthy = service_m.IsHealthy
	m.InFlightRequests = atomic.LoadInt64(&s.metrics.InFlightRequests)
	m.TotalRequests = atomic.LoadInt64(&s.metrics.TotalRequests)
	m.OnMessageErrors = atomic.LoadInt64(&s.metrics.OnMessageErrors)
	m.RejectedRequests = atomic.LoadInt64(&s.metrics.RejectedRequests)
	m.QueueSize = atomic.LoadInt64(&s.metrics.QueueSize)
	m.ActiveWorkers = atomic.LoadInt64(&s.pool.ActiveWorkers)
	return m
}

func (s *TcpServiceManager) Metadata() TcpMetadata {
	meta := s.service.Metadata()
	return meta
}
