package inspect

import (
	lua "github.com/yuin/gopher-lua"
)

func mergeLogScratch(scratch map[string]any, L *lua.LState, idx int) {
	if scratch == nil {
		return
	}
	if L.Get(idx).Type() == lua.LTTable && (L.GetTop() == idx || L.GetTop() == idx+1 && L.Get(idx+1).Type() == lua.LTTable) {
		t := L.CheckTable(idx)
		t.ForEach(func(k, v lua.LValue) {
			key, ok := luaKeyString(k)
			if !ok {
				return
			}
			scratch[key] = luaValueToGo(v)
		})
		return
	}
	key := L.CheckString(idx)
	if L.GetTop() < idx+1 {
		return
	}
	scratch[key] = luaValueToGo(L.Get(idx + 1))
}

func luaKeyString(v lua.LValue) (string, bool) {
	switch x := v.(type) {
	case lua.LString:
		return string(x), true
	case lua.LNumber:
		return x.String(), true
	default:
		return "", false
	}
}

func luaValueToGo(v lua.LValue) any {
	switch x := v.(type) {
	case lua.LBool:
		return bool(x)
	case lua.LString:
		return string(x)
	case lua.LNumber:
		return float64(x)
	case *lua.LTable:
		return luaTableToGo(x)
	case *lua.LNilType:
		return nil
	default:
		return nil
	}
}

func luaTableToGo(t *lua.LTable) any {
	if t == nil {
		return nil
	}
	// array-like
	maxN := 0
	onlyArray := true
	t.ForEach(func(k, v lua.LValue) {
		if n, ok := k.(lua.LNumber); ok {
			if int(n) > maxN {
				maxN = int(n)
			}
			return
		}
		onlyArray = false
	})
	if onlyArray && maxN > 0 {
		arr := make([]any, maxN)
		for i := 1; i <= maxN; i++ {
			arr[i-1] = luaValueToGo(t.RawGetInt(i))
		}
		return arr
	}
	m := make(map[string]any)
	t.ForEach(func(k, v lua.LValue) {
		key, ok := luaKeyString(k)
		if !ok {
			return
		}
		m[key] = luaValueToGo(v)
	})
	return m
}
