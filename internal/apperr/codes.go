package apperr

// Code — машиночитаемый идентификатор ошибки для API (поле JSON error).
type Code string

const (
	InvalidJSON              Code = "invalid_json"
	MethodNotAllowed         Code = "method_not_allowed"
	ListenRequired           Code = "listen_required"
	ListenAddressInUse       Code = "listen_address_in_use"
	InstanceNameRequired     Code = "instance_name_required"
	InstanceNameTaken        Code = "instance_name_taken"
	InstanceNotFound         Code = "instance_not_found"
	InvalidConnectMode       Code = "invalid_connect_mode"
	AuthStaticUsersRequired  Code = "auth_static_users_required"
	LDAPURLRequired          Code = "ldap_url_required"
	ProxyMITMCAMissing       Code = "proxy_mitm_ca_missing"
	ProxyApplyFailed         Code = "proxy_apply_failed"
	CertNotFound             Code = "certificate_not_found"
	KeyNotFound              Code = "key_not_found"
	KeyBitsTooSmall          Code = "key_bits_too_small"
	CertFileRequired         Code = "cert_file_required"
	KeyFileRequired          Code = "key_file_required"
	HTMLFileRequired         Code = "html_file_required"
	InvalidMultipartForm     Code = "invalid_multipart_form"
	LoadProxySettingsFailed  Code = "load_proxy_settings_failed"
	UpdateProxySettingsFailed Code = "update_proxy_settings_failed"
	LoadReportsDashboardFailed  Code = "load_reports_dashboard_failed"
	UpdateReportsDashboardFailed Code = "update_reports_dashboard_failed"
	CAStatusFailed           Code = "ca_status_failed"
	ReadFileFailed           Code = "read_file_failed"
	ClearForbiddenPageFailed Code = "clear_forbidden_page_failed"
	ClearGatewayPageFailed   Code = "clear_gateway_page_failed"

	InvalidCredentials       Code = "invalid_credentials"
	UserDisabled             Code = "user_disabled"
	LoginTemporarilyDisabled Code = "login_temporarily_disabled"
	UsernamePasswordRequired Code = "username_password_required"
	LoginFailed              Code = "login_failed"
)
