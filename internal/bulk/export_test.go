package bulk

import (
	"bytes"
	"context"
	"net/url"
	"testing"
)

type stubExporter struct{}

func (stubExporter) Name() string                            { return "stub" }
func (stubExporter) Headers() []string                       { return []string{"a", "b"} }
func (stubExporter) StreamRows(_ context.Context, _ url.Values, write func([]string) error) error {
	return write([]string{"x", "y"})
}

func TestWriteCSV_EmitsBOM(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(context.Background(), &buf, stubExporter{}, url.Values{}); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("expected UTF-8 BOM prefix, got %q", buf.Bytes()[:4])
	}
}
