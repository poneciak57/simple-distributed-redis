package service

import (
	"fmt"
	"io"
	"main/src/config"
	"main/src/protocol"
	"net"
	"time"
)

type RedisService struct {
	meta            TcpMetadata
	storage         *StorageService
	cfg             *config.Config
	logger          *config.Logger
	timeoutDuration time.Duration
}

func NewRedisServices(storage *StorageService, cfg *config.Config, logger *config.Logger) *RedisService {
	return &RedisService{
		meta: TcpMetadata{
			BaseMetadata: BaseMetadata{
				Name:    "RedisService",
				Version: "1.0.0",
			},
			Host: cfg.Redis.Host,
			Port: cfg.Redis.Port,
		},
		storage:         storage,
		cfg:             cfg,
		logger:          logger,
		timeoutDuration: time.Duration(cfg.Redis.Timeout) * time.Second,
	}
}

func errorResponse(err error) []byte {
	return []byte(fmt.Sprintf("-ERR %v\r\n", err))
}

func okResponse() []byte {
	return []byte("+OK\r\n")
}

func pongResponse() []byte {
	return []byte("+PONG\r\n")
}

func (s *RedisService) OnMessage(conn net.Conn) error {
	parser := protocol.NewResp2Parser(conn, s.cfg.Redis.MaxMessageSize)
	opParser := protocol.MakeOpParser(parser)

	for {
		op, err := opParser.Parse()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.logger.Warn("Connection timed out during read: %v", netErr)
				return err
			}
			return fmt.Errorf("failed to parse operation: %w", err)
		}

		s.logger.Debug("Processing operation: %s", op.Kind)

		var response []byte

		switch op.Kind {
		case protocol.GET:
			val, err := s.storage.Get(op.Payload.(protocol.OpPayloadGet).Key)
			if err != nil {
				response = errorResponse(err)
			} else {
				response, err = parser.Render(val)
				if err != nil {
					response = errorResponse(err)
				}
			}
		case protocol.SET:
			err = s.storage.Set(op.Payload.(protocol.OpPayloadSet).Key, op.Payload.(protocol.OpPayloadSet).Value)
			if err != nil {
				response = errorResponse(err)
			} else {
				response = okResponse()
			}
		case protocol.DELETE:
			err := s.storage.Delete(op.Payload.(protocol.OpPayloadDelete).Key)
			if err != nil {
				response = errorResponse(err)
			} else {
				response = okResponse()
			}
		case protocol.PING:
			response = pongResponse()
		default:
			// It is an error on the client side, respond with error
			response = errorResponse(fmt.Errorf("unknown operation"))
		}

		_, err = conn.Write(response)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.logger.Warn("Connection timed out during write: %v", netErr)
				return err
			}
			return fmt.Errorf("failed to write response: %w", err)
		}

		if s.timeoutDuration > 0 {
			conn.SetDeadline(time.Now().Add(s.timeoutDuration))
		}
	}
}

func (s *RedisService) Metadata() TcpMetadata {
	return s.meta
}

func (s *RedisService) Metrics() BaseMetrics {
	return BaseMetrics{
		IsHealthy: true,
	}
}
