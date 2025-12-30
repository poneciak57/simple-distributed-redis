package service

import (
	"fmt"
	"io"
	"main/src/config"
	"main/src/protocol"
	"net"
)

type RedisService struct {
	meta    TcpMetadata
	storage *StorageService
	cfg     *config.Config
}

func NewRedisServices(storage *StorageService, config *config.Config) *RedisService {
	return &RedisService{
		meta: TcpMetadata{
			BaseMetadata: BaseMetadata{
				Name:    "RedisService",
				Version: "1.0.0",
			},
			Host: config.Redis.Host,
			Port: config.Redis.Port,
		},
		storage: storage,
		cfg:     config,
	}
}

func (s *RedisService) OnMessage(conn net.Conn) error {
	parser := protocol.NewResp2Parser(conn)
	opParser := protocol.MakeOpParser(parser)

	for {
		op, err := opParser.Parse()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to parse operation: %w", err)
		}

		var response []byte

		// Maybe make storage work on bytes directly to avoid conversions

		switch op.Kind {
		case protocol.GET:
			val, err := s.storage.Get(op.Payload.(protocol.OpPayloadGet).Key)
			if err != nil {
				response = []byte(fmt.Sprintf("-ERR %v\r\n", err))
			} else {
				response, err = parser.Render(val)
				if err != nil {
					response = []byte(fmt.Sprintf("-ERR %v\r\n", err))
				}
			}
		case protocol.SET:
			err = s.storage.Set(op.Payload.(protocol.OpPayloadSet).Key, op.Payload.(protocol.OpPayloadSet).Value)
			if err != nil {
				response = []byte(fmt.Sprintf("-ERR %v\r\n", err))
			} else {
				response = []byte("+OK\r\n")
			}
		case protocol.DELETE:
			err := s.storage.Delete(op.Payload.(protocol.OpPayloadDelete).Key)
			if err != nil {
				response = []byte(fmt.Sprintf("-ERR %v\r\n", err))
			} else {
				response = []byte("+OK\r\n")
			}
		case protocol.PING:
			response = []byte("+PONG\r\n")
		default:
			// It is an error on the client side, respond with error
			response = []byte("-ERR unknown operation\r\n")
		}

		_, err = conn.Write(response)
		if err != nil {
			return fmt.Errorf("failed to write response: %w", err)
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
