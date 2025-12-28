package service

import (
	"errors"
	"sync/atomic"
	"time"
)

type ServiceStatus int32

const (
	STOPPED ServiceStatus = iota
	STARTING
	RUNNING
	STOPPING
)

type BaseMetadata struct {
	Name    string
	Version string
}

type BaseMetrics struct {
	IsHealthy bool
}

// Service defines the interface for a generic service that can process messages of type T.
type Service[T any, Metrics any, Metadata any] interface {
	OnMessage(msg T) error

	// Methods for introspection/health checks
	// Be careful this method may be called concurrently with OnMessage.
	Metrics() Metrics   // e.g. processed messages, errors, etc.
	Metadata() Metadata // e.g. service name, version, etc.
}

// ServiceManager manages the lifecycle of a Service. It can start, stop the service,
// and report its current status. It also provides a channel to send messages to the service.
// It should be safe to send messages from multiple goroutines.
// Messages are sent to the service via the OnMessage method synchronously.
type ServiceManager[T any, Metrics any, Metadata any] interface {
	Start() error
	SendMessage() <-chan T
	Stop() error

	// Methods for introspection/health checks
	Status() ServiceStatus
	Metadata() Metadata
	Metrics() Metrics
}

type SimpleServiceManager[T any, Metrics any, Metadata any] struct {
	service  Service[T, Metrics, Metadata]
	status   int32
	msgChan  chan T
	stopChan chan struct{}
}

func NewSimpleServiceManager[T any, Metrics any, Metadata any](service Service[T, Metrics, Metadata]) *SimpleServiceManager[T, Metrics, Metadata] {
	return &SimpleServiceManager[T, Metrics, Metadata]{
		service:  service,
		status:   int32(STOPPED),
		msgChan:  make(chan T, 100),
		stopChan: make(chan struct{}),
	}
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) loop() {
	atomic.StoreInt32(&m.status, int32(RUNNING))
	for {
		select {
		case msg := <-m.msgChan:
			m.service.OnMessage(msg)
		case <-m.stopChan:
			atomic.StoreInt32(&m.status, int32(STOPPED))
			return
		}
	}
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) SendMessage() <-chan T {
	return m.msgChan
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) Metrics() Metrics {
	return m.service.Metrics()
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) Metadata() Metadata {
	return m.service.Metadata()
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) Status() ServiceStatus {
	return ServiceStatus(atomic.LoadInt32(&m.status))
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) Start() error {
	if !atomic.CompareAndSwapInt32(&m.status, int32(STOPPED), int32(STARTING)) {
		return nil
	}

	go m.loop()
	return nil
}

func (m *SimpleServiceManager[T, Metrics, Metadata]) Stop() error {
	for {
		current := atomic.LoadInt32(&m.status)
		if ServiceStatus(current) != RUNNING && ServiceStatus(current) != STARTING {
			return nil
		}
		if atomic.CompareAndSwapInt32(&m.status, current, int32(STOPPING)) {
			break
		}
	}

	timeout := time.After(5 * time.Second)

	select {
	case m.stopChan <- struct{}{}:
		return nil
	case <-timeout:
		return errors.New("service stop timed out")
	}
}
