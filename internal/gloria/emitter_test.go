package gloria

import (
	"testing"

	"fz/internal/utils"
)

func TestEmitAndExecMain(t *testing.T) {
	src := `fn main() { @rax = 10; @rbx = 32; @rax += @rbx; return @rax }`
	code, err := Emit(src)
	if err != nil {
		t.Fatalf("emit error: %v", err)
	}
	if len(code) == 0 {
		t.Fatalf("no code emitted")
	}
	t.Logf("code: %x", code)
	out := utils.ExecRawRet(code)
	if out != 42 {
		t.Fatalf("unexpected return: %d", out)
	}
}
