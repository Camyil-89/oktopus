package inspect

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func pushRequestContext(L *lua.LState, c RequestContext, logScratch map[string]any) {
	t := L.NewTable()
	L.SetField(t, "method", lua.LString(strings.ToUpper(c.Method)))
	L.SetField(t, "host", lua.LString(c.Host))
	L.SetField(t, "path", lua.LString(c.Path))
	L.SetField(t, "url", lua.LString(c.URL))
	L.SetField(t, "content_type", lua.LString(c.ContentType))
	L.SetField(t, "content_length", lua.LNumber(c.ContentLength))
	L.SetField(t, "user", lua.LString(c.User))
	L.SetField(t, "client_ip", lua.LString(c.ClientIP))

	groups := L.NewTable()
	for i, g := range c.Groups {
		groups.RawSetInt(i+1, lua.LString(g))
	}
	L.SetField(t, "groups", groups)

	names := L.NewTable()
	for i, name := range c.UploadFilenames {
		names.RawSetInt(i+1, lua.LString(name))
	}
	L.SetField(t, "upload_filenames", names)

	L.SetField(t, "header", L.NewFunction(func(L *lua.LState) int {
		name := luaStringArg(L, 1)
		L.Push(lua.LString(c.Header(name)))
		return 1
	}))
	L.SetField(t, "has_group", L.NewFunction(func(L *lua.LState) int {
		name := luaStringArg(L, 1)
		if c.HasGroup(name) {
			L.Push(lua.LTrue)
		} else {
			L.Push(lua.LFalse)
		}
		return 1
	}))
	L.SetField(t, "log", L.NewFunction(func(L *lua.LState) int {
		idx := 1
		if L.Get(idx).Type() == lua.LTTable {
			idx++
		}
		if L.GetTop() < idx {
			return 0
		}
		mergeLogScratch(logScratch, L, idx)
		return 0
	}))

	L.Push(t)
}

// luaStringArg — первый аргумент строки с учётом вызова ctx:fn("x") (self таблица на стеке).
func luaStringArg(L *lua.LState, idx int) string {
	if L.Get(idx).Type() == lua.LTTable {
		idx++
	}
	return L.CheckString(idx)
}

func tableStringField(t *lua.LTable, key string) string {
	v := t.RawGetString(key)
	if s, ok := v.(lua.LString); ok {
		return string(s)
	}
	return ""
}
