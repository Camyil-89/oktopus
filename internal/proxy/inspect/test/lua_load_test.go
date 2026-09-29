package inspect_test

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLuaLoadInspect0(t *testing.T) {
	src := `inspect_0 = (function()
function inspect(ctx)
  return false
end
return inspect
end)()`
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(src); err != nil {
		t.Fatal(err)
	}
	fn := L.GetGlobal("inspect_0")
	if fn.Type() != lua.LTFunction {
		t.Fatalf("type %v", fn.Type())
	}
}
