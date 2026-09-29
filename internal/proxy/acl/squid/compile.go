package squid

import (
	"oktopus/internal/proxy/acl"
)

// Compile собирает Engine из конфига и list-сущностей БД.
func Compile(configText string, lists []ListInput) (*acl.Engine, error) {
	res := Analyze(NormalizePolicyText(configText), lists)
	if !res.OK {
		return nil, &CompileErrors{Diagnostics: res.Diagnostics}
	}
	if res.Engine == nil {
		return acl.EmptyEngine(), nil
	}
	return res.Engine, nil
}

// ValidateConfig проверяет конфиг и lists без сборки Engine.
func ValidateConfig(configText string, lists []ListInput) error {
	res := AnalyzeValidate(configText, lists)
	if !res.OK {
		return &CompileErrors{Diagnostics: res.Diagnostics}
	}
	return nil
}
