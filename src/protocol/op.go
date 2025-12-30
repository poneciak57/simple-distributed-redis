package protocol

import "fmt"

type OpType int

const (
	GET OpType = iota
	SET
	DELETE
	PING
)

type OpPayloadPing struct {
}

type OpPayloadSet struct {
	Key   string
	Value Resp2Value
}

type OpPayloadGet struct {
	Key string
}

type OpPayloadDelete struct {
	Key string
}

type OpPayload interface{}

type Op struct {
	Kind    OpType
	Payload OpPayload
}

// Parser is responsible for parsing operations from a stream.
// Implementations should hold the underlying io.Reader and handle buffering.
type Parser[T any] interface {
	// Parse reads the next operation from the underlying stream.
	// It returns T or an error (e.g. io.EOF).
	Parse() (T, error)
}

// Renderer handles converting operations back to bytes.
type Renderer[T any] interface {
	Render(op T) ([]byte, error)
}

func OpParserIter(parser Parser[*Op]) func() (*Op, error) {
	return func() (*Op, error) {
		return parser.Parse()
	}
}

type OpParser struct {
	respParser *Resp2Parser
}

func MakeOpParser(respParser *Resp2Parser) OpParser {
	return OpParser{
		respParser: respParser,
	}
}

func (p *OpParser) Parse() (*Op, error) {
	value, err := p.respParser.Parse()
	if err != nil {
		return nil, err
	}

	// Accept both Resp2Array and []Resp2Value
	var array []Resp2Value
	switch v := value.(type) {
	case Resp2Array:
		array = v
	case []Resp2Value:
		array = v
	default:
		return nil, fmt.Errorf("expected RESP2 array for operation")
	}

	if len(array) == 0 {
		return nil, fmt.Errorf("expected RESP2 array for operation")
	}

	// Extract operation type - accept both simple and bulk strings
	opTypeStr := extractString(array[0])

	switch opTypeStr {
	case "GET":
		if len(array) != 2 {
			return nil, fmt.Errorf("GET operation requires 1 argument")
		}
		key := extractString(array[1])
		if key == "" && array[1] != nil {
			return nil, fmt.Errorf("GET operation key must be a string")
		}
		return &Op{
			Kind: GET,
			Payload: OpPayloadGet{
				Key: key,
			},
		}, nil
	case "SET":
		if len(array) != 3 {
			return nil, fmt.Errorf("SET operation requires 2 arguments")
		}
		key := extractString(array[1])
		if key == "" && array[1] != nil {
			return nil, fmt.Errorf("SET operation key must be a string")
		}
		return &Op{
			Kind: SET,
			Payload: OpPayloadSet{
				Key:   key,
				Value: array[2],
			},
		}, nil
	case "DELETE":
		if len(array) != 2 {
			return nil, fmt.Errorf("DELETE operation requires 1 argument")
		}
		key := extractString(array[1])
		if key == "" && array[1] != nil {
			return nil, fmt.Errorf("DELETE operation key must be a string")
		}
		return &Op{
			Kind: DELETE,
			Payload: OpPayloadDelete{
				Key: key,
			},
		}, nil
	case "PING":
		if len(array) != 1 {
			return nil, fmt.Errorf("PING operation requires no arguments")
		}
		return &Op{
			Kind:    PING,
			Payload: OpPayloadPing{},
		}, nil
	default:
		return nil, fmt.Errorf("unknown operation type: %s", opTypeStr)
	}
}

func (p *OpParser) Render(op *Op) ([]byte, error) {
	respParser := NewResp2ParserFromBytes(nil)
	var array Resp2Array

	switch op.Kind {
	case GET:
		payload := op.Payload.(OpPayloadGet)
		array = Resp2Array{
			Resp2SimpleString("GET"),
			Resp2BulkString(payload.Key),
		}
	case SET:
		payload := op.Payload.(OpPayloadSet)
		array = Resp2Array{
			Resp2SimpleString("SET"),
			Resp2BulkString(payload.Key),
			payload.Value,
		}
	case DELETE:
		payload := op.Payload.(OpPayloadDelete)
		array = Resp2Array{
			Resp2SimpleString("DELETE"),
			Resp2BulkString(payload.Key),
		}
	case PING:
		array = Resp2Array{
			Resp2SimpleString("PING"),
		}
	default:
		return nil, fmt.Errorf("unknown operation type: %v", op.Kind)
	}

	return respParser.Render(array)
}

// extractString extracts a string from various RESP2 string types
func extractString(value Resp2Value) string {
	switch v := value.(type) {
	case Resp2SimpleString:
		return string(v)
	case Resp2BulkString:
		return string(v)
	case string:
		return v
	default:
		return ""
	}
}
