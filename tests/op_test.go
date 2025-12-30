package tests

import (
	"main/src/protocol"
	"testing"
)

func TestOpParserGET(t *testing.T) {
	t.Run("Normal with simple strings", func(t *testing.T) {
		inp := []byte("*2\r\n+GET\r\n+mykey\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if op.Kind != protocol.GET {
			t.Errorf("Expected GET operation, got %v", op.Kind)
		}

		payload, ok := op.Payload.(protocol.OpPayloadGet)
		if !ok {
			t.Fatalf("Expected OpPayloadGet, got %T", op.Payload)
		}

		if payload.Key != "mykey" {
			t.Errorf("Expected key 'mykey', got '%s'", payload.Key)
		}
	})

	t.Run("Normal with bulk strings", func(t *testing.T) {
		inp := []byte("*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if op.Kind != protocol.GET {
			t.Errorf("Expected GET operation, got %v", op.Kind)
		}

		payload, ok := op.Payload.(protocol.OpPayloadGet)
		if !ok {
			t.Fatalf("Expected OpPayloadGet, got %T", op.Payload)
		}

		if payload.Key != "mykey" {
			t.Errorf("Expected key 'mykey', got '%s'", payload.Key)
		}
	})

	t.Run("Mixed string types", func(t *testing.T) {
		inp := []byte("*2\r\n+GET\r\n$6\r\nmykey2\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		payload, ok := op.Payload.(protocol.OpPayloadGet)
		if !ok {
			t.Fatalf("Expected OpPayloadGet, got %T", op.Payload)
		}

		if payload.Key != "mykey2" {
			t.Errorf("Expected key 'mykey2', got '%s'", payload.Key)
		}
	})

	t.Run("Too few arguments", func(t *testing.T) {
		inp := []byte("*1\r\n+GET\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for GET with too few arguments")
		}
		if err.Error() != "GET operation requires 1 argument" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Too many arguments", func(t *testing.T) {
		inp := []byte("*3\r\n+GET\r\n+key1\r\n+key2\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for GET with too many arguments")
		}
	})

	t.Run("Non-string key", func(t *testing.T) {
		inp := []byte("*2\r\n+GET\r\n:123\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for GET with non-string key")
		}
		if err.Error() != "GET operation key must be a string" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestOpParserSET(t *testing.T) {
	t.Run("Normal with simple strings", func(t *testing.T) {
		inp := []byte("*3\r\n+SET\r\n+mykey\r\n+myvalue\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if op.Kind != protocol.SET {
			t.Errorf("Expected SET operation, got %v", op.Kind)
		}

		payload, ok := op.Payload.(protocol.OpPayloadSet)
		if !ok {
			t.Fatalf("Expected OpPayloadSet, got %T", op.Payload)
		}

		if payload.Key != "mykey" {
			t.Errorf("Expected key 'mykey', got '%s'", payload.Key)
		}

		val, ok := payload.Value.(protocol.Resp2SimpleString)
		if !ok {
			t.Fatalf("Expected Resp2SimpleString value, got %T", payload.Value)
		}
		if string(val) != "myvalue" {
			t.Errorf("Expected value 'myvalue', got '%s'", string(val))
		}
	})

	t.Run("Normal with bulk strings", func(t *testing.T) {
		inp := []byte("*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$6\r\nvalue1\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if op.Kind != protocol.SET {
			t.Errorf("Expected SET operation, got %v", op.Kind)
		}

		payload, ok := op.Payload.(protocol.OpPayloadSet)
		if !ok {
			t.Fatalf("Expected OpPayloadSet, got %T", op.Payload)
		}

		if payload.Key != "key1" {
			t.Errorf("Expected key 'key1', got '%s'", payload.Key)
		}

		val, ok := payload.Value.(protocol.Resp2BulkString)
		if !ok {
			t.Fatalf("Expected Resp2BulkString value, got %T", payload.Value)
		}
		if string(val) != "value1" {
			t.Errorf("Expected value 'value1', got '%s'", string(val))
		}
	})

	t.Run("Value with special characters", func(t *testing.T) {
		inp := []byte("*3\r\n+SET\r\n+key\r\n$13\r\nhello\r\nworld!\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		payload, ok := op.Payload.(protocol.OpPayloadSet)
		if !ok {
			t.Fatalf("Expected OpPayloadSet, got %T", op.Payload)
		}

		val, ok := payload.Value.(protocol.Resp2BulkString)
		if !ok {
			t.Fatalf("Expected Resp2BulkString value, got %T", payload.Value)
		}
		if string(val) != "hello\r\nworld!" {
			t.Errorf("Expected value 'hello\\r\\nworld!', got '%s'", string(val))
		}
	})

	t.Run("Too few arguments", func(t *testing.T) {
		inp := []byte("*2\r\n+SET\r\n+key\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for SET with too few arguments")
		}
		if err.Error() != "SET operation requires 2 arguments" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Too many arguments", func(t *testing.T) {
		inp := []byte("*4\r\n+SET\r\n+key\r\n+value\r\n+extra\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for SET with too many arguments")
		}
	})

	t.Run("Non-string key", func(t *testing.T) {
		inp := []byte("*3\r\n+SET\r\n:123\r\n+value\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for SET with non-string key")
		}
		if err.Error() != "SET operation key must be a string" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Non-string value", func(t *testing.T) {
		inp := []byte("*3\r\n+SET\r\n+key\r\n:456\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		payload, ok := op.Payload.(protocol.OpPayloadSet)
		if !ok {
			t.Fatalf("Expected OpPayloadSet, got %T", op.Payload)
		}

		val, ok := payload.Value.(protocol.Resp2Integer)
		if !ok {
			t.Fatalf("Expected Resp2Integer value, got %T", payload.Value)
		}
		if int64(val) != 456 {
			t.Errorf("Expected value 456, got %d", val)
		}
	})
}

func TestOpParserDELETE(t *testing.T) {
	t.Run("Normal with simple strings", func(t *testing.T) {
		inp := []byte("*2\r\n+DELETE\r\n+mykey\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if op.Kind != protocol.DELETE {
			t.Errorf("Expected DELETE operation, got %v", op.Kind)
		}

		payload, ok := op.Payload.(protocol.OpPayloadDelete)
		if !ok {
			t.Fatalf("Expected OpPayloadDelete, got %T", op.Payload)
		}

		if payload.Key != "mykey" {
			t.Errorf("Expected key 'mykey', got '%s'", payload.Key)
		}
	})

	t.Run("Normal with bulk strings", func(t *testing.T) {
		inp := []byte("*2\r\n$6\r\nDELETE\r\n$8\r\ntodelete\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		op, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if op.Kind != protocol.DELETE {
			t.Errorf("Expected DELETE operation, got %v", op.Kind)
		}

		payload, ok := op.Payload.(protocol.OpPayloadDelete)
		if !ok {
			t.Fatalf("Expected OpPayloadDelete, got %T", op.Payload)
		}

		if payload.Key != "todelete" {
			t.Errorf("Expected key 'todelete', got '%s'", payload.Key)
		}
	})

	t.Run("Too few arguments", func(t *testing.T) {
		inp := []byte("*1\r\n+DELETE\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for DELETE with too few arguments")
		}
		if err.Error() != "DELETE operation requires 1 argument" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Too many arguments", func(t *testing.T) {
		inp := []byte("*3\r\n+DELETE\r\n+key1\r\n+key2\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for DELETE with too many arguments")
		}
	})

	t.Run("Non-string key", func(t *testing.T) {
		inp := []byte("*2\r\n+DELETE\r\n:789\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for DELETE with non-string key")
		}
		if err.Error() != "DELETE operation key must be a string" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestOpParserInvalidCases(t *testing.T) {
	t.Run("Unknown operation type", func(t *testing.T) {
		inp := []byte("*2\r\n+UNKNOWN\r\n+key\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for unknown operation type")
		}
		if err.Error() != "unknown operation type: UNKNOWN" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Empty array", func(t *testing.T) {
		inp := []byte("*0\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for empty array")
		}
		if err.Error() != "expected RESP2 array for operation" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Non-array value", func(t *testing.T) {
		inp := []byte("+NOTANARRAY\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for non-array value")
		}
		if err.Error() != "expected RESP2 array for operation" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Integer as operation type", func(t *testing.T) {
		inp := []byte("*2\r\n:123\r\n+key\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for integer as operation type")
		}
		if err.Error() != "unknown operation type: " {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Error as operation type", func(t *testing.T) {
		inp := []byte("*2\r\n-ERR test\r\n+key\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		_, err := opParser.Parse()
		if err == nil {
			t.Fatalf("Expected error for error type as operation")
		}
	})
}

func TestOpParserMultipleOperations(t *testing.T) {
	t.Run("Parse multiple operations in sequence", func(t *testing.T) {
		inp := []byte("*2\r\n+GET\r\n+key1\r\n*3\r\n+SET\r\n+key2\r\n+value2\r\n*2\r\n+DELETE\r\n+key3\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		opParser := protocol.MakeOpParser(parser)

		// First operation: GET
		op1, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse 1 failed: %v", err)
		}
		if op1.Kind != protocol.GET {
			t.Errorf("Expected first operation to be GET, got %v", op1.Kind)
		}
		payload1 := op1.Payload.(protocol.OpPayloadGet)
		if payload1.Key != "key1" {
			t.Errorf("Expected key 'key1', got '%s'", payload1.Key)
		}

		// Second operation: SET
		op2, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse 2 failed: %v", err)
		}
		if op2.Kind != protocol.SET {
			t.Errorf("Expected second operation to be SET, got %v", op2.Kind)
		}
		payload2 := op2.Payload.(protocol.OpPayloadSet)
		val2, ok := payload2.Value.(protocol.Resp2SimpleString)
		if !ok {
			t.Fatalf("Expected Resp2SimpleString value, got %T", payload2.Value)
		}
		if payload2.Key != "key2" || string(val2) != "value2" {
			t.Errorf("Expected key 'key2' and value 'value2', got '%s' and '%s'", payload2.Key, string(val2))
		}

		// Third operation: DELETE
		op3, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Parse 3 failed: %v", err)
		}
		if op3.Kind != protocol.DELETE {
			t.Errorf("Expected third operation to be DELETE, got %v", op3.Kind)
		}
		payload3 := op3.Payload.(protocol.OpPayloadDelete)
		if payload3.Key != "key3" {
			t.Errorf("Expected key 'key3', got '%s'", payload3.Key)
		}

		// Fourth parse should return EOF
		_, err = opParser.Parse()
		if err == nil {
			t.Errorf("Expected EOF error on fourth parse")
		}
	})
}

func TestOpRender(t *testing.T) {
	renderParser := protocol.MakeOpParser(protocol.NewResp2ParserFromBytes(nil))
	t.Run("Render GET operation", func(t *testing.T) {
		op := &protocol.Op{
			Kind: protocol.GET,
			Payload: protocol.OpPayloadGet{
				Key: "testkey",
			},
		}

		data, err := renderParser.Render(op)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Parse it back to verify
		parser := protocol.NewResp2ParserFromBytes(data)
		opParser := protocol.MakeOpParser(parser)
		parsedOp, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Re-parse failed: %v", err)
		}

		if parsedOp.Kind != protocol.GET {
			t.Errorf("Expected GET operation, got %v", parsedOp.Kind)
		}

		payload := parsedOp.Payload.(protocol.OpPayloadGet)
		if payload.Key != "testkey" {
			t.Errorf("Expected key 'testkey', got '%s'", payload.Key)
		}
	})

	t.Run("Render SET operation", func(t *testing.T) {
		op := &protocol.Op{
			Kind: protocol.SET,
			Payload: protocol.OpPayloadSet{
				Key:   "mykey",
				Value: "myvalue",
			},
		}

		data, err := renderParser.Render(op)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Parse it back to verify
		parser := protocol.NewResp2ParserFromBytes(data)
		opParser := protocol.MakeOpParser(parser)
		parsedOp, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Re-parse failed: %v", err)
		}

		if parsedOp.Kind != protocol.SET {
			t.Errorf("Expected SET operation, got %v", parsedOp.Kind)
		}

		payload := parsedOp.Payload.(protocol.OpPayloadSet)
		val, ok := payload.Value.(protocol.Resp2SimpleString)
		if !ok {
			// It might be bulk string depending on implementation details of Render
			// But "myvalue" is short and no special chars, so likely SimpleString if Render supports it
			// Actually Render implementation for string:
			// if len(v) == 0 { ... } else if containsCRLF(v) || len(v) > 512 { ... } return ... SimpleString
			// So it should be SimpleString
			t.Fatalf("Expected Resp2SimpleString value, got %T", payload.Value)
		}

		if payload.Key != "mykey" || string(val) != "myvalue" {
			t.Errorf("Expected key 'mykey' and value 'myvalue', got '%s' and '%s'", payload.Key, string(val))
		}
	})

	t.Run("Render DELETE operation", func(t *testing.T) {
		op := &protocol.Op{
			Kind: protocol.DELETE,
			Payload: protocol.OpPayloadDelete{
				Key: "todelete",
			},
		}

		data, err := renderParser.Render(op)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Parse it back to verify
		parser := protocol.NewResp2ParserFromBytes(data)
		opParser := protocol.MakeOpParser(parser)
		parsedOp, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Re-parse failed: %v", err)
		}

		if parsedOp.Kind != protocol.DELETE {
			t.Errorf("Expected DELETE operation, got %v", parsedOp.Kind)
		}

		payload := parsedOp.Payload.(protocol.OpPayloadDelete)
		if payload.Key != "todelete" {
			t.Errorf("Expected key 'todelete', got '%s'", payload.Key)
		}
	})

	t.Run("Render SET with special characters", func(t *testing.T) {
		op := &protocol.Op{
			Kind: protocol.SET,
			Payload: protocol.OpPayloadSet{
				Key:   "key",
				Value: "line1\r\nline2\r\nline3",
			},
		}

		data, err := renderParser.Render(op)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Parse it back to verify
		parser := protocol.NewResp2ParserFromBytes(data)
		opParser := protocol.MakeOpParser(parser)
		parsedOp, err := opParser.Parse()
		if err != nil {
			t.Fatalf("Re-parse failed: %v", err)
		}

		payload := parsedOp.Payload.(protocol.OpPayloadSet)
		val, ok := payload.Value.(protocol.Resp2BulkString)
		if !ok {
			t.Fatalf("Expected Resp2BulkString value, got %T", payload.Value)
		}

		if string(val) != "line1\r\nline2\r\nline3" {
			t.Errorf("Expected value with CRLF, got '%s'", string(val))
		}
	})
}
