package squid

import "sort"

// ParsePolicyConfig разбирает текст политики (для validate без повторного parse).
func ParsePolicyConfig(configText string) (*Config, []Diagnostic) {
	var diags []Diagnostic
	cfg := parseConfigCollect(configText, &diags)
	return cfg, diags
}

// ReferencedACLNames — имена acl из http_access, ssl_verify и delay_access (для выборочной подгрузки lists).
func ReferencedACLNames(configText string) map[string]struct{} {
	cfg, _ := ParsePolicyConfig(configText)
	if cfg == nil {
		return nil
	}
	return ReferencedACLNamesFromConfig(cfg)
}

// ReferencedACLNamesFromConfig — имена acl, на которые ссылается уже разобранный конфиг.
func ReferencedACLNamesFromConfig(cfg *Config) map[string]struct{} {
	if cfg == nil {
		return nil
	}
	names := make(map[string]struct{})
	collectLineAccessACLs(cfg.HTTPAccess, names)
	collectLineAccessACLs(cfg.SSLVerify, names)
	for _, line := range cfg.Delay.Access {
		for _, cl := range line.ACLs {
			names[cl.Name] = struct{}{}
		}
	}
	return names
}

func collectLineAccessACLs(lines []Line, names map[string]struct{}) {
	for _, line := range lines {
		for _, cl := range line.AccessACLs {
			names[cl.Name] = struct{}{}
		}
	}
}

// ListNamesToMergeFromDB — имена named list, которые нужно подтянуть из БД:
// упомянуты в http_access / ssl_verify / delay_access, но не объявлены acl в тексте policy.
func ListNamesToMergeFromDB(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	refs := ReferencedACLNamesFromConfig(cfg)
	if len(refs) == 0 {
		return nil
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		if _, inCfg := cfg.DefinitionAt[name]; inCfg {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
