package service

import (
	"fmt"
	"main/src/config"
	"net"
	"sync/atomic"
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
}

type TcpServiceManager struct {
	service Service[net.Conn, BaseMetrics, TcpMetadata]
	metrics TcpMetrics
	logger  *config.Logger
}

// NewTcpServiceManager creates a new TcpServiceManager that listens on the given address and port,
// Service should not close the net.Conn passed to OnMessage, TcpServiceManager will handle closing it.
func NewTcpServiceManager(service Service[net.Conn, BaseMetrics, TcpMetadata]) *TcpServiceManager {
	return &TcpServiceManager{
		service: service,
		logger:  config.NewLogger("TcpServiceManager"),
	}
}

func (s *TcpServiceManager) Stop() error {
	return nil
}

func (s *TcpServiceManager) loop(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		atomic.AddInt64(&s.metrics.InFlightRequests, 1)
		atomic.AddInt64(&s.metrics.TotalRequests, 1)
		go func() {
			err := s.service.OnMessage(conn)
			if err != nil {
				s.logger.Error("Error handling message: %v", err)
				atomic.AddInt64(&s.metrics.OnMessageErrors, 1)
			}
			atomic.AddInt64(&s.metrics.InFlightRequests, -1)
			conn.Close()
		}()
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
	return m
}

func (s *TcpServiceManager) Metadata() TcpMetadata {
	meta := s.service.Metadata()
	return meta
}
