package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/apperr"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	"oktopus/internal/db/proxyinstances/domain"
	proxyinspectservice "oktopus/internal/db/proxyinspect/service"
	"oktopus/internal/pki"
	"oktopus/internal/proxy/acl/squid"
)

const (
	instanceBundleFormat        = "oktopus-instance-bundle"
	instanceBundleFormatVersion = 1
)

const (
	bundleEntryManifest     = "manifest.json"
	bundleEntryInstance     = "instance.json"
	bundleEntryACLPolicy    = "acl_policy.json"
	bundleEntryACLLists     = "acl_lists.json"
	bundleEntryInspectRules = "inspect_rules.json"
	bundleEntryCACert       = "ca/ca.crt"
	bundleEntryCAKey        = "ca/ca.key"
)

type instanceBundleManifest struct {
	Format        string `json:"format"`
	FormatVersion int    `json:"format_version"`
	ExportedAt    string `json:"exported_at"`
	InstanceName  string `json:"instance_name"`
	SourceID      string `json:"source_instance_id"`
}

type instanceBundleSettings struct {
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	Listen              string `json:"listen"`
	ConnectMode         string `json:"connect_mode"`
	AuthEnabled         bool   `json:"auth_enabled"`
	AuthStaticUsers     string `json:"auth_static_users"`
	AuthRealm           string `json:"auth_realm"`
	AuthBackend         string `json:"auth_backend"`
	AuthCacheTTLMinutes int64  `json:"auth_cache_ttl_minutes"`
	LDAPURL             string `json:"ldap_url"`
	LDAPBaseDN          string `json:"ldap_base_dn"`
	LDAPBindDN          string `json:"ldap_bind_dn"`
	LDAPBindPassword    string `json:"ldap_bind_password,omitempty"`
}

type instanceBundleACLPolicy struct {
	ConfigText string `json:"config_text"`
}

type instanceBundleACLList struct {
	Name                string `json:"name"`
	ListType            string `json:"list_type"`
	Body                string `json:"body"`
	SourceMode          string `json:"source_mode"`
	SourceURL           string `json:"source_url,omitempty"`
	PollIntervalMinutes int    `json:"poll_interval_minutes"`
}

type instanceBundleInspectRule struct {
	Name      string `json:"name"`
	Script    string `json:"script"`
	Action    int16  `json:"action"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

type instanceBundleInspectRules struct {
	Rules []instanceBundleInspectRule `json:"rules"`
}

// ExportInstanceBundle собирает ZIP с настройками одного инстанса (ACL, lists, inspect, CA).
func (s *Service) ExportInstanceBundle(ctx context.Context, instanceID uuid.UUID) ([]byte, string, error) {
	inst, err := s.repo.Get(ctx, instanceID)
	if err != nil {
		return nil, "", err
	}
	if s.acl == nil {
		return nil, "", fmt.Errorf("acl service unavailable")
	}
	pol, err := s.acl.GetPolicy(ctx, instanceID)
	if err != nil {
		return nil, "", fmt.Errorf("acl policy: %w", err)
	}
	listNames := squidListNames(pol.ConfigText)
	var lists []instanceBundleACLList
	if len(listNames) > 0 {
		dbLists, err := s.acl.ListNamedListsByNames(ctx, listNames)
		if err != nil {
			return nil, "", err
		}
		lists = make([]instanceBundleACLList, len(dbLists))
		for i, l := range dbLists {
			lists[i] = instanceBundleACLList{
				Name:                l.Name,
				ListType:            l.ListType,
				Body:                l.Body,
				SourceMode:          l.SourceMode,
				SourceURL:           l.SourceURL,
				PollIntervalMinutes: l.PollIntervalMinutes,
			}
		}
	}
	var inspectRules instanceBundleInspectRules
	if s.inspect != nil {
		rules, err := s.inspect.List(ctx, instanceID)
		if err != nil {
			return nil, "", fmt.Errorf("inspect rules: %w", err)
		}
		inspectRules.Rules = make([]instanceBundleInspectRule, len(rules))
		for i, r := range rules {
			inspectRules.Rules[i] = instanceBundleInspectRule{
				Name:      r.Name,
				Script:    r.Script,
				Action:    r.Action,
				Enabled:   r.Enabled,
				SortOrder: r.SortOrder,
			}
		}
	}
	manifest := instanceBundleManifest{
		Format:        instanceBundleFormat,
		FormatVersion: instanceBundleFormatVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		InstanceName:  inst.Name,
		SourceID:      inst.ID.String(),
	}
	settings := instanceSettingsToBundle(inst)

	var certPEM, keyPEM []byte
	if info, err := pki.InspectCAFiles(inst.CACertPath, inst.CAKeyPath); err == nil {
		if info.CertExists {
			certPEM, _ = os.ReadFile(inst.CACertPath)
		}
		if info.KeyExists {
			keyPEM, _ = os.ReadFile(inst.CAKeyPath)
		}
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	if err := writeJSONEntry(zw, bundleEntryManifest, manifest); err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if err := writeJSONEntry(zw, bundleEntryInstance, settings); err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if err := writeJSONEntry(zw, bundleEntryACLPolicy, instanceBundleACLPolicy{ConfigText: pol.ConfigText}); err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if err := writeJSONEntry(zw, bundleEntryACLLists, lists); err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if err := writeJSONEntry(zw, bundleEntryInspectRules, inspectRules); err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if len(certPEM) > 0 {
		if err := writeBytesEntry(zw, bundleEntryCACert, certPEM); err != nil {
			_ = zw.Close()
			return nil, "", err
		}
	}
	if len(keyPEM) > 0 {
		if err := writeBytesEntry(zw, bundleEntryCAKey, keyPEM); err != nil {
			_ = zw.Close()
			return nil, "", err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	filename := sanitizeBundleFilename(inst.Name) + "-oktopus.zip"
	return buf.Bytes(), filename, nil
}

// ImportInstanceBundle применяет ZIP к существующему инстансу (настройки, ACL, inspect, CA).
func (s *Service) ImportInstanceBundle(ctx context.Context, instanceID uuid.UUID, r io.ReaderAt, size int64) (domain.Instance, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleReadFailed)
	}
	manifestRaw, err := readZipJSON[instanceBundleManifest](zr, bundleEntryManifest)
	if err != nil {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	if manifestRaw.Format != instanceBundleFormat || manifestRaw.FormatVersion != instanceBundleFormatVersion {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	settings, err := readZipJSON[instanceBundleSettings](zr, bundleEntryInstance)
	if err != nil {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	policy, err := readZipJSON[instanceBundleACLPolicy](zr, bundleEntryACLPolicy)
	if err != nil {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	lists, err := readZipJSON[[]instanceBundleACLList](zr, bundleEntryACLLists)
	if err != nil {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	inspectDoc, err := readZipJSON[instanceBundleInspectRules](zr, bundleEntryInspectRules)
	if err != nil {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	certPEM, _ := readZipBytesOptional(zr, bundleEntryCACert)
	keyPEM, _ := readZipBytesOptional(zr, bundleEntryCAKey)

	if s.acl == nil {
		return domain.Instance{}, apperr.New(apperr.InstanceConfigImportFailed)
	}
	listInputs := bundleListsToSyncInputs(lists)
	if result := s.acl.AnalyzePolicy(ctx, policy.ConfigText, squidListInputsFromBundle(lists)); !result.OK {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	if err := s.acl.MergeNamedListsAndPublish(ctx, listInputs); err != nil {
		return domain.Instance{}, mapImportErr(err)
	}
	if _, err := s.importUpdateSettings(ctx, instanceID, settings); err != nil {
		return domain.Instance{}, err
	}
	if len(certPEM) > 0 && len(keyPEM) > 0 {
		if err := s.importCABundle(ctx, instanceID, certPEM, keyPEM); err != nil {
			return domain.Instance{}, err
		}
	} else if len(certPEM) > 0 || len(keyPEM) > 0 {
		return domain.Instance{}, apperr.New(apperr.ConfigBundleInvalid)
	}
	if _, err := s.acl.SetPolicyAndPublish(ctx, instanceID, policy.ConfigText); err != nil {
		return domain.Instance{}, mapImportErr(err)
	}
	if s.inspect != nil {
		inputs := make([]proxyinspectservice.SyncRuleInput, len(inspectDoc.Rules))
		for i, rule := range inspectDoc.Rules {
			script := rule.Script
			inputs[i] = proxyinspectservice.SyncRuleInput{
				Name:      rule.Name,
				Script:    &script,
				Action:    rule.Action,
				Enabled:   rule.Enabled,
				SortOrder: rule.SortOrder,
			}
		}
		if _, err := s.inspect.SyncAndPublish(ctx, instanceID, inputs); err != nil {
			return domain.Instance{}, mapImportErr(err)
		}
	}
	updated, err := s.repo.Get(ctx, instanceID)
	if err != nil {
		return domain.Instance{}, err
	}
	return updated, nil
}

// importUpdateSettings применяет поля из архива; при занятом listen отключает инстанс и сохраняет остальное.
func (s *Service) importUpdateSettings(ctx context.Context, instanceID uuid.UUID, settings instanceBundleSettings) (domain.Instance, error) {
	cur, err := s.repo.Get(ctx, instanceID)
	if err != nil {
		return domain.Instance{}, err
	}
	in := bundleSettingsToUpdateInput(settings)
	if taken, err := s.nameTakenByOther(ctx, instanceID, settings.Name); err != nil {
		return domain.Instance{}, err
	} else if taken {
		settings.Name = cur.Name
		in = bundleSettingsToUpdateInput(settings)
	}
	if taken, err := s.listenTakenByOther(ctx, instanceID, settings.Listen); err != nil {
		return domain.Instance{}, err
	} else if taken {
		return s.importUpdateSettingsDisabled(ctx, instanceID, settings, cur.Listen)
	}
	updated, err := s.Update(ctx, instanceID, in)
	if err == nil {
		return updated, nil
	}
	if isNameConflict(err) {
		settings.Name = cur.Name
		in = bundleSettingsToUpdateInput(settings)
		updated, err = s.Update(ctx, instanceID, in)
		if err == nil {
			return updated, nil
		}
	}
	if !isListenConflict(err) {
		return domain.Instance{}, err
	}
	retryListen := strings.TrimSpace(settings.Listen)
	if retryListen == "" {
		retryListen = cur.Listen
	}
	return s.importUpdateSettingsDisabled(ctx, instanceID, settings, retryListen)
}

func (s *Service) importUpdateSettingsDisabled(ctx context.Context, instanceID uuid.UUID, settings instanceBundleSettings, listen string) (domain.Instance, error) {
	disabled := false
	in := bundleSettingsToUpdateInput(settings)
	in.Enabled = &disabled
	in.Listen = &listen
	return s.Update(ctx, instanceID, in)
}

func (s *Service) nameTakenByOther(ctx context.Context, instanceID uuid.UUID, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return false, err
	}
	for _, inst := range list {
		if inst.ID != instanceID && inst.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func isNameConflict(err error) bool {
	code, ok := apperr.CodeOf(err)
	return ok && code == apperr.InstanceNameTaken
}

func (s *Service) listenTakenByOther(ctx context.Context, instanceID uuid.UUID, listen string) (bool, error) {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return false, nil
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return false, err
	}
	for _, inst := range list {
		if inst.ID != instanceID && inst.Listen == listen {
			return true, nil
		}
	}
	return false, nil
}

func isListenConflict(err error) bool {
	if code, ok := apperr.CodeOf(err); ok && code == apperr.ListenAddressInUse {
		return true
	}
	if mapped := apperr.MapProxyApply(err); mapped != err {
		if code, ok := apperr.CodeOf(mapped); ok && code == apperr.ListenAddressInUse {
			return true
		}
	}
	return errors.Is(err, domain.ErrListenTaken)
}

func (s *Service) importCABundle(ctx context.Context, instanceID uuid.UUID, certPEM, keyPEM []byte) error {
	if _, err := s.repo.Get(ctx, instanceID); err != nil {
		return err
	}
	outDir := domain.ConfigDir(instanceID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if _, _, err := pki.SaveCAFiles(outDir, certPEM, keyPEM, true); err != nil {
		return err
	}
	cert, key := domain.DefaultCAPaths(instanceID)
	if _, err := s.repo.UpdateCAPaths(ctx, instanceID, cert, key); err != nil {
		return err
	}
	if err := s.applyFleetIgnoringListenConflict(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) applyFleetIgnoringListenConflict(ctx context.Context) error {
	_, err := s.applyFleet(ctx)
	if err == nil || isListenConflict(err) {
		return nil
	}
	return err
}

func instanceSettingsToBundle(inst domain.Instance) instanceBundleSettings {
	return instanceBundleSettings{
		Name:                inst.Name,
		Enabled:             inst.Enabled,
		Listen:              inst.Listen,
		ConnectMode:         inst.ConnectMode,
		AuthEnabled:         inst.AuthEnabled,
		AuthStaticUsers:     inst.AuthStaticUsers,
		AuthRealm:           inst.AuthRealm,
		AuthBackend:         inst.AuthBackend,
		AuthCacheTTLMinutes: inst.AuthCacheTTLMinutes,
		LDAPURL:             inst.LDAPURL,
		LDAPBaseDN:          inst.LDAPBaseDN,
		LDAPBindDN:          inst.LDAPBindDN,
		LDAPBindPassword:    inst.LDAPBindPassword,
	}
}

func bundleSettingsToUpdateInput(b instanceBundleSettings) UpdateInput {
	name := strings.TrimSpace(b.Name)
	enabled := b.Enabled
	listen := b.Listen
	connectMode := b.ConnectMode
	authEnabled := b.AuthEnabled
	authStatic := b.AuthStaticUsers
	authRealm := b.AuthRealm
	authBackend := b.AuthBackend
	authCache := b.AuthCacheTTLMinutes
	ldapURL := b.LDAPURL
	ldapBase := b.LDAPBaseDN
	ldapBind := b.LDAPBindDN
	in := UpdateInput{
		Enabled:             &enabled,
		Listen:              &listen,
		ConnectMode:         &connectMode,
		AuthEnabled:         &authEnabled,
		AuthStaticUsers:     &authStatic,
		AuthRealm:           &authRealm,
		AuthBackend:         &authBackend,
		AuthCacheTTLMinutes: &authCache,
		LDAPURL:             &ldapURL,
		LDAPBaseDN:          &ldapBase,
		LDAPBindDN:          &ldapBind,
	}
	if name != "" {
		in.Name = &name
	}
	if b.LDAPBindPassword != "" {
		pw := b.LDAPBindPassword
		in.LDAPBindPassword = &pw
	}
	return in
}

func bundleListsToSyncInputs(lists []instanceBundleACLList) []proxyaclservice.SyncNamedListInput {
	out := make([]proxyaclservice.SyncNamedListInput, len(lists))
	for i, l := range lists {
		out[i] = proxyaclservice.SyncNamedListInput{
			Name:                l.Name,
			ListType:            l.ListType,
			Body:                l.Body,
			SourceMode:          l.SourceMode,
			SourceURL:           l.SourceURL,
			PollIntervalMinutes: l.PollIntervalMinutes,
		}
	}
	return out
}

func squidListInputsFromBundle(lists []instanceBundleACLList) []squid.ListInput {
	out := make([]squid.ListInput, len(lists))
	for i, l := range lists {
		out[i] = squid.ListInput{Name: l.Name, ListType: l.ListType, Body: l.Body}
	}
	return out
}

func squidListNames(configText string) []string {
	cfg, _ := squid.ParsePolicyConfig(configText)
	if cfg == nil {
		return nil
	}
	return squid.ListNamesToMergeFromDB(cfg)
}

func writeJSONEntry(zw *zip.Writer, name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeBytesEntry(zw, name, data)
}

func writeBytesEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readZipJSON[T any](zr *zip.Reader, name string) (T, error) {
	var zero T
	data, err := readZipBytes(zr, name)
	if err != nil {
		return zero, err
	}
	if err := json.Unmarshal(data, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func readZipBytes(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		return data, readErr
	}
	return nil, fmt.Errorf("missing %s", name)
}

func readZipBytesOptional(zr *zip.Reader, name string) ([]byte, error) {
	data, err := readZipBytes(zr, name)
	if err != nil {
		return nil, nil
	}
	return data, nil
}

func mapImportErr(err error) error {
	if err == nil {
		return nil
	}
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return err
	}
	return apperr.Wrap(apperr.InstanceConfigImportFailed, err)
}

func sanitizeBundleFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "instance"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "instance"
	}
	return out
}
