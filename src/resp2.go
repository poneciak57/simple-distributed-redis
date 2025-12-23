package src

// RESP2 parser and renderer for byte streams.
// reference: https://redis.io/docs/latest/develop/reference/protocol-spec/

import (
	"fmt"
	"io"
	"strconv"
)

type Resp2BytesParser struct {
	data []byte
}

func MakeResp2ByteParser(data []byte) Resp2BytesParser {
	return Resp2BytesParser{
		data: data,
	}
}

type Resp2Error string // Special type to distinguish errors
type Resp2Value interface{}
type Resp2Array []Resp2Value

func (p *Resp2BytesParser) parseUntilCRLF() ([]byte, error) {
	bytes := []byte{}
	for len(p.data) > 0 {
		b := p.data[0]
		p.data = p.data[1:]
		if b == '\r' {
			if len(p.data) > 0 && p.data[0] == '\n' {
				p.data = p.data[1:]
				return bytes, nil
			} else {
				return nil, fmt.Errorf("invalid CRLF termination")
			}
		}
		bytes = append(bytes, b)
	}
	return nil, fmt.Errorf("unterminated CRLF sequence")
}

func (p *Resp2BytesParser) parseSimpleString() (string, error) {
	str, err := p.parseUntilCRLF()
	return string(str), err
}

func (p *Resp2BytesParser) parseError() (Resp2Error, error) {
	// For simplicity, treat errors as simple strings here
	// But we has special type for them
	str, err := p.parseSimpleString()
	return Resp2Error(str), err
}

func (p *Resp2BytesParser) parseInteger() (int64, error) {
	strBytes, err := p.parseUntilCRLF()
	if err != nil {
		return 0, err
	}
	if len(strBytes) == 0 {
		return 0, fmt.Errorf("invalid integer format: empty string")
	}
	if (strBytes[0] == '-' || strBytes[0] == '+') && len(strBytes) == 1 {
		return 0, fmt.Errorf("invalid integer format")
	}
	if strBytes[0] == '+' {
		strBytes = strBytes[1:]
	}
	str := string(strBytes)
	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer format: %s", str)
	}
	return value, nil
}

func (p *Resp2BytesParser) parseBulkString() (string, error) {
	lengthBytes, err := p.parseUntilCRLF()
	if err != nil {
		return "", err
	}
	lengthStr := string(lengthBytes)
	length, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid bulk string length: %s", lengthStr)
	}
	if length == -1 {
		// Null bulk string
		return "", nil
	}
	if length < -1 {
		return "", fmt.Errorf("invalid bulk string length: %d", length)
	}
	if int64(len(p.data)) < length+2 {
		return "", fmt.Errorf("bulk string data too short")
	}
	strBytes := p.data[:length]
	p.data = p.data[length:]
	// Expect CRLF
	if len(p.data) < 2 || p.data[0] != '\r' || p.data[1] != '\n' {
		return "", fmt.Errorf("invalid bulk string termination")
	}
	p.data = p.data[2:]
	return string(strBytes), nil
}

func (p *Resp2BytesParser) parseArray() (Resp2Value, error) {
	lengthBytes, err := p.parseUntilCRLF()
	if err != nil {
		return nil, err
	}
	lengthStr := string(lengthBytes)
	length, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", lengthStr)
	}
	if length == -1 {
		// Null array
		return nil, nil
	}
	if length < -1 {
		return nil, fmt.Errorf("invalid array length: %d", length)
	}
	values := make([]Resp2Value, 0, length)
	for i := int64(0); i < length; i++ {
		value, err := p.Parse()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (p *Resp2BytesParser) Parse() (Resp2Value, error) {
	if len(p.data) == 0 {
		return nil, io.EOF
	}

	kind := p.data[0]
	p.data = p.data[1:]
	switch kind {
	case '+': // Simple String
		return p.parseSimpleString()
	case '-': // Error
		return p.parseError()
	case ':': // Integer
		return p.parseInteger()
	case '$': // Bulk String
		return p.parseBulkString()
	case '*': // Array
		return p.parseArray()
	default:
		return nil, fmt.Errorf("unknown RESP2 type: %c", kind)
	}
}

func containsCRLF(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' || s[i] == '\n' {
			return true
		}
	}
	return false
}

func (p *Resp2BytesParser) renderArray(v []Resp2Value) ([]byte, error) {
	if v == nil {
		return []byte("*-1\r\n"), nil
	}
	result := []byte("*" + strconv.FormatInt(int64(len(v)), 10) + "\r\n")
	for _, elem := range v {
		elemBytes, err := p.Render(elem)
		if err != nil {
			return nil, err
		}
		result = append(result, elemBytes...)
	}
	return result, nil
}

func (p *Resp2BytesParser) Render(value Resp2Value) ([]byte, error) {
	switch v := value.(type) {
	case string:
		// it might be simple string or bulk string
		// for simplicity we will check if it contains CR or LF if so we will use bulk string
		// or if it's too long it will be better for buffering
		if len(v) == 0 {
			// Null bulk string
			return []byte("$-1\r\n"), nil
		} else if containsCRLF(v) || len(v) > 512 {
			// bulk string
			return []byte("$" + strconv.Itoa(len(v)) + "\r\n" + v + "\r\n"), nil
		}
		// simple string
		return []byte("+" + v + "\r\n"), nil
	case Resp2Error:
		return []byte("-" + string(v) + "\r\n"), nil
	case int64:
		return []byte(":" + strconv.FormatInt(v, 10) + "\r\n"), nil
	case nil:
		// Null array
		return []byte("*-1\r\n"), nil
	case Resp2Array:
		return p.renderArray([]Resp2Value(v))
	case []Resp2Value:
		return p.renderArray(v)
	default:
		return nil, fmt.Errorf("unsupported RESP2 value type %T", value)
	}
}
