package sandbox

import (
	"context"
	"testing"
)

func TestExecuteNativeReturnsNonZeroError(t *testing.T) {
	d := &Docker{nativeMode: true, currentCwd: t.TempDir()}
	out, err := d.Execute(context.Background(), "printf failed-output; exit 7")
	if err == nil {
		t.Fatal("expected a non-zero command to return an error")
	}
	if out != "failed-output" {
		t.Fatalf("unexpected partial output: %q", out)
	}
}
