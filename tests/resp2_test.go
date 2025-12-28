package tests

import (
	"main/src/protocol"
	"testing"
)

func TestResp2ParserSimpleText(t *testing.T) {

	t.Run("Normal", func(t *testing.T) {
		inp := []byte("+OK\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2SimpleString); ok {
			if string(instance) != "OK" {
				t.Errorf("Expected 'OK', got '%s'", string(instance))
			}
		} else {
			t.Fatalf("Expected Resp2SimpleString type, got %T", op)
		}

		op, err = parser.Parse()
		if err == nil {
			t.Errorf("Expected error on second parse")
		}
		if err.Error() != "EOF" {
			t.Errorf("Expected EOF error, got: %v", err)
		}
		if op != nil {
			t.Errorf("Expected nil op on error, got: %v", op)
		}
	})

	t.Run("Invalid CRLF", func(t *testing.T) {
		inp := []byte("+OK\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on invalid CRLF")
		}
	})

	t.Run("No termination", func(t *testing.T) {
		inp := []byte("+OK")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on unterminated input")
		}
	})
}

func TestResp2ParserError(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		inp := []byte("-ERR something went wrong\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2Error); ok {
			if string(instance) != "ERR something went wrong" {
				t.Errorf("Expected 'ERR something went wrong', got '%s'", string(instance))
			}
		} else {
			t.Fatalf("Expected Resp2Error type, got %T", op)
		}
	})

}

func TestResp2ParserBulkString(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		inp := []byte("$6\r\nfoobar\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2BulkString); ok {
			if string(instance) != "foobar" {
				t.Errorf("Expected 'foobar', got '%s'", string(instance))
			}
		} else {
			t.Fatalf("Expected Resp2BulkString type, got %T", op)
		}
	})

	t.Run("Null Bulk String", func(t *testing.T) {
		inp := []byte("$-1\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if bulkStr, ok := op.(protocol.Resp2BulkString); !ok || string(bulkStr) != "" {
			t.Errorf("Expected empty Resp2BulkString for null bulk string, got: %v (%T)", op, op)
		}
	})

	t.Run("Invalid Length", func(t *testing.T) {
		inp := []byte("$x\r\nfoobar\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on invalid length")
		}
	})

	t.Run("Insufficient Data", func(t *testing.T) {
		inp := []byte("$6\r\nfoo\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on insufficient data")
		}
	})

	t.Run("CLRF as Data", func(t *testing.T) {
		inp := []byte("$4\r\n\r\n\r\n\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2BulkString); ok {
			if string(instance) != "\r\n\r\n" {
				t.Errorf("Expected '\\r\\n\\r\\n', got '%s'", string(instance))
			}
		} else {
			t.Fatalf("Expected Resp2BulkString type, got %T", op)
		}
	})
}

func TestResp2ParserInteger(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		inp := []byte(":12345\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2Integer); ok {
			if int64(instance) != 12345 {
				t.Errorf("Expected 12345, got %d", int64(instance))
			}
		} else {
			t.Fatalf("Expected Resp2Integer type, got %T", op)
		}
	})

	t.Run("Plus Sign", func(t *testing.T) {
		inp := []byte(":+123\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2Integer); ok {
			if int64(instance) != 123 {
				t.Errorf("Expected 123, got %d", int64(instance))
			}
		} else {
			t.Fatalf("Expected Resp2Integer type, got %T", op)
		}
	})

	t.Run("Invalid Integer", func(t *testing.T) {
		inp := []byte(":12x34\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on invalid integer")
		}
	})

	t.Run("Minus Sign Only", func(t *testing.T) {
		inp := []byte(":-\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on invalid integer")
		}
	})

	t.Run("Minus sign", func(t *testing.T) {
		inp := []byte(":-123\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.(protocol.Resp2Integer); ok {
			if int64(instance) != -123 {
				t.Errorf("Expected -123, got %d", int64(instance))
			}
		} else {
			t.Fatalf("Expected Resp2Integer type, got %T", op)
		}
	})

}

func TestResp2ParserArray(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		inp := []byte("*3\r\n:1\r\n:2\r\n:3\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if instance, ok := op.([]protocol.Resp2Value); ok {
			if len(instance) != 3 {
				t.Errorf("Expected array of length 3, got %d", len(instance))
			} else {
				for i, v := range instance {
					if num, ok := v.(protocol.Resp2Integer); ok {
						expected := int64(i + 1)
						if int64(num) != expected {
							t.Errorf("Expected element %d to be %d, got %d", i, expected, int64(num))
						}
					} else {
						t.Errorf("Expected element %d to be Resp2Integer, got %T", i, v)
					}
				}
			}
		} else {
			t.Fatalf("Expected []Resp2Value type, got %T", op)
		}
	})

	t.Run("Null Array", func(t *testing.T) {
		inp := []byte("*-1\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		op, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if op != nil {
			t.Errorf("Expected nil for null array, got: %v", op)
		}
	})

	t.Run("Invalid Length", func(t *testing.T) {
		inp := []byte("*x\r\n:1\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on invalid array length")
		}
	})

	t.Run("Insufficient Elements", func(t *testing.T) {
		inp := []byte("*3\r\n:1\r\n:2\r\n")
		parser := protocol.NewResp2ParserFromBytes(inp)
		_, err := parser.Parse()
		if err == nil {
			t.Fatalf("Expected error on insufficient array elements")
		}
	})
}

func TestResp2Renderer(t *testing.T) {
	t.Run("Simple String", func(t *testing.T) {
		renderer := protocol.NewResp2ParserFromBytes(nil)
		value := "Hello, World!"
		data, err := renderer.Render(value)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		expected := []byte("+Hello, World!\r\n")
		if string(data) != string(expected) {
			t.Errorf("Expected %q, got %q", string(expected), string(data))
		}
	})

	t.Run("Integer", func(t *testing.T) {
		renderer := protocol.NewResp2ParserFromBytes(nil)
		value := int64(12345)
		data, err := renderer.Render(value)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		expected := []byte(":12345\r\n")
		if string(data) != string(expected) {
			t.Errorf("Expected %q, got %q", string(expected), string(data))
		}
	})

	t.Run("Bulk String", func(t *testing.T) {
		renderer := protocol.NewResp2ParserFromBytes(nil)
		value := "foo\r\nbar"
		data, err := renderer.Render(value)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		expected := []byte("$8\r\nfoo\r\nbar\r\n")
		if string(data) != string(expected) {
			t.Errorf("Expected %q, got %q", string(expected), string(data))
		}
	})

	t.Run("Array", func(t *testing.T) {
		renderer := protocol.NewResp2ParserFromBytes(nil)
		value := []protocol.Resp2Value{int64(1), "two", int64(3)}
		data, err := renderer.Render(value)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		expected := []byte("*3\r\n:1\r\n+two\r\n:3\r\n")
		if string(data) != string(expected) {
			t.Errorf("Expected %q, got %q", string(expected), string(data))
		}
	})

	t.Run("Null Bulk String", func(t *testing.T) {
		renderer := protocol.NewResp2ParserFromBytes(nil)
		value := ""
		data, err := renderer.Render(value)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		expected := []byte("$-1\r\n")
		if string(data) != string(expected) {
			t.Errorf("Expected %q, got %q", string(expected), string(data))
		}
	})

	t.Run("Null Array", func(t *testing.T) {
		renderer := protocol.NewResp2ParserFromBytes(nil)
		var value []protocol.Resp2Value = nil
		data, err := renderer.Render(value)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		expected := []byte("*-1\r\n")
		if string(data) != string(expected) {
			t.Errorf("Expected %q, got %q", string(expected), string(data))
		}
	})
}
