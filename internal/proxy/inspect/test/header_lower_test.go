package inspect_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/proxy/inspect"
)

func TestEval_uploadHeuristic_GET_noMatch(t *testing.T) {
	script := `-- test
function inspect(ctx)
  local method = ctx.method or ""
  if method ~= "POST" and method ~= "PUT" and method ~= "PATCH" then
    return false
  end
  local cd = ctx:header("Content-Disposition")
  if cd ~= "" and cd:lower():find("filename=", 1, true) then
    return true
  end
  local ct = (ctx.content_type or ""):lower()
  if ct == "" then
    return false
  end
  return false
end`
	prog, err := inspect.BuildProgram([]inspect.RuleInput{{
		ID: uuid.New(), Name: "h", Action: 0, Script: script,
	}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := inspect.NewRunner(prog).Eval(inspect.RequestContext{Method: "GET", URL: "https://bitrix/ajax.php"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Matched {
		t.Fatalf("GET should not match: %+v", res)
	}
}

func TestEval_uploadHeuristic_POST_disposition(t *testing.T) {
	script := `function inspect(ctx)
  local method = ctx.method or ""
  if method ~= "POST" and method ~= "PUT" and method ~= "PATCH" then
    return false
  end
  local cd = ctx:header("Content-Disposition")
  if cd ~= "" and cd:lower():find("filename=", 1, true) then
    return true
  end
  return false
end`
	prog, err := inspect.BuildProgram([]inspect.RuleInput{{
		ID:     uuid.MustParse("01a0f662-e1ae-7983-91f9-f9253011094c"),
		Name:   "h",
		Action: 0,
		Script: script,
	}})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "http://example.com/", nil)
	req.Header.Set("Content-Disposition", `form-data; name="file"; filename="a.bin"`)
	res, err := inspect.NewRunner(prog).Eval(inspect.RequestContextFromHTTP(context.Background(), req))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !res.Matched || !res.Deny || res.RuleID != "01a0f662-e1ae-7983-91f9-f9253011094c" {
		t.Fatalf("got %+v", res)
	}
}
