package src

// RESP2 parser and renderer for byte streams.
// reference: https://redis.io/docs/latest/develop/reference/protocol-spec/

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type Resp2Parser struct {
	reader *bufio.Reader
}

func NewResp2Parser(r io.Reader) *Resp2Parser {
	return &Resp2Parser{
		reader: bufio.NewReader(r),
	}
}

func NewResp2ParserFromBytes(data []byte) *Resp2Parser {
	return NewResp2Parser(bytes.NewReader(data))
}

// RESP2 value types for type-safe parsing
type Resp2SimpleString string // Simple strings: +OK\r\n
type Resp2BulkString string   // Bulk strings: $5\r\nhello\r\n
type Resp2Error string        // Errors: -ERR message\r\n
type Resp2Integer int64       // Integers: :1000\r\n

type Resp2Value interface{}
type Resp2Array []Resp2Value

func (p *Resp2Parser) parseUntilCRLF() ([]byte, error) {
	line, err := p.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("invalid CRLF termination")
	}
	// Return without the \r\n
	return line[:len(line)-2], nil
}

func (p *Resp2Parser) parseSimpleString() (Resp2SimpleString, error) {
	str, err := p.parseUntilCRLF()
	return Resp2SimpleString(str), err
}

func (p *Resp2Parser) parseError() (Resp2Error, error) {
	// For simplicity, treat errors as simple strings here
	// But we has special type for them
	str, err := p.parseUntilCRLF()
	return Resp2Error(str), err
}

func (p *Resp2Parser) parseInteger() (Resp2Integer, error) {
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
	return Resp2Integer(value), nil
}

func (p *Resp2Parser) parseBulkString() (Resp2BulkString, error) {
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
	// Read exact number of bytes
	strBytes := make([]byte, length)
	_, err = io.ReadFull(p.reader, strBytes)
	if err != nil {
		return "", fmt.Errorf("bulk string data too short: %w", err)
	}
	// Expect CRLF
	crlf := make([]byte, 2)
	_, err = io.ReadFull(p.reader, crlf)
	if err != nil {
		return "", fmt.Errorf("missing CRLF after bulk string: %w", err)
	}
	if crlf[0] != '\r' || crlf[1] != '\n' {
		return "", fmt.Errorf("invalid bulk string termination")
	}
	return Resp2BulkString(strBytes), nil
}

func (p *Resp2Parser) parseArray() (Resp2Value, error) {
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

func (p *Resp2Parser) Parse() (Resp2Value, error) {
	kindByte, err := p.reader.ReadByte()
	if err != nil {
		return nil, err
	}

	kind := kindByte
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

func (p *Resp2Parser) renderArray(v []Resp2Value) ([]byte, error) {
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

func (p *Resp2Parser) Render(value Resp2Value) ([]byte, error) {
	switch v := value.(type) {
	case Resp2SimpleString:
		return []byte("+" + string(v) + "\r\n"), nil
	case Resp2BulkString:
		if len(v) == 0 {
			// Null bulk string
			return []byte("$-1\r\n"), nil
		}
		return []byte("$" + strconv.Itoa(len(v)) + "\r\n" + string(v) + "\r\n"), nil
	case string:
		// For backwards compatibility: treat plain strings as bulk strings
		// Check if it contains CRLF or is long
		if len(v) == 0 {
			return []byte("$-1\r\n"), nil
		} else if containsCRLF(v) || len(v) > 512 {
			return []byte("$" + strconv.Itoa(len(v)) + "\r\n" + v + "\r\n"), nil
		}
		// Simple string for short strings without CRLF
		return []byte("+" + v + "\r\n"), nil
	case Resp2Error:
		return []byte("-" + string(v) + "\r\n"), nil
	case Resp2Integer:
		return []byte(":" + strconv.FormatInt(int64(v), 10) + "\r\n"), nil
	case int64:
		// For backwards compatibility
		return []byte(":" + strconv.FormatInt(v, 10) + "\r\n"), nil
	case int:
		// For convenience
		return []byte(":" + strconv.FormatInt(int64(v), 10) + "\r\n"), nil
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
