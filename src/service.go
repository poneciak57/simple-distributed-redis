package src

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

// Service defines the interface for a generic service that can process messages of type T.
// Messages are sent to the service via the OnMessage method synchronously.
type Service[T any] interface {
	OnMessage(msg T) error

	// Methods for introspection/health checks
	// Be careful this method may be called concurrently with OnMessage.
	Metrics() map[string]any  // e.g. processed messages, errors, etc.
	Metadata() map[string]any // e.g. service name, version, etc.
}

// ServiceManager manages the lifecycle of a Service. It can start, stop the service,
// and report its current status. It also provides a channel to send messages to the service.
// It should be safe to send messages from multiple goroutines.
type ServiceManager[T any] interface {
	Start() error
	SendMessage() <-chan T
	Stop() error

	// Methods for introspection/health checks
	Status() ServiceStatus
	Metadata() map[string]any
	Metrics() map[string]any
}

type SimpleServiceManager[T any] struct {
	service  Service[T]
	status   int32
	msgChan  chan T
	stopChan chan struct{}
}

func NewSimpleServiceManager[T any](service Service[T]) *SimpleServiceManager[T] {
	return &SimpleServiceManager[T]{
		service:  service,
		status:   int32(STOPPED),
		msgChan:  make(chan T, 100),
		stopChan: make(chan struct{}),
	}
}

func (m *SimpleServiceManager[T]) loop() {
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

func (m *SimpleServiceManager[T]) SendMessage() <-chan T {
	return m.msgChan
}

func (m *SimpleServiceManager[T]) Metrics() map[string]any {
	return m.service.Metrics()
}

func (m *SimpleServiceManager[T]) Metadata() map[string]any {
	return m.service.Metadata()
}

func (m *SimpleServiceManager[T]) Status() ServiceStatus {
	return ServiceStatus(atomic.LoadInt32(&m.status))
}

func (m *SimpleServiceManager[T]) Start() error {
	if !atomic.CompareAndSwapInt32(&m.status, int32(STOPPED), int32(STARTING)) {
		return nil
	}

	go m.loop()
	return nil
}

func (m *SimpleServiceManager[T]) Stop() error {
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
