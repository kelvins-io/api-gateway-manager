/** Kong Gateway 3.4.2 plugin config form fields for plugins without hand-written templates. */

export type FormFieldType = 'string' | 'number' | 'boolean' | 'select' | 'multiselect' | 'csv' | 'textarea' | 'json'

export const HTTP_METHODS = ['GET', 'HEAD', 'PUT', 'PATCH', 'POST', 'DELETE', 'OPTIONS', 'TRACE', 'CONNECT'] as const
export const PROXY_CACHE_METHODS = ['HEAD', 'GET', 'POST', 'PATCH', 'PUT'] as const
export const SESSION_LOGOUT_METHODS = ['GET', 'POST', 'DELETE'] as const
export const JWT_CLAIMS = ['exp', 'nbf'] as const

export interface FormField {
  key: string
  label: string
  type: FormFieldType
  required?: boolean
  default?: unknown
  options?: Array<string | number>
  placeholder?: string
  min?: number
  max?: number
  hint?: string
}

export const pluginFormSchemas: Record<string, FormField[]> = {
  'ldap-auth': [
    { key: 'ldap_host', label: 'ldap_host', type: 'string', required: true, placeholder: 'ldap.example.com' },
    { key: 'ldap_port', label: 'ldap_port', type: 'number', required: true, default: 389, min: 1, max: 65535 },
    { key: 'ldaps', label: 'ldaps', type: 'boolean', default: false },
    { key: 'start_tls', label: 'start_tls', type: 'boolean', default: false },
    { key: 'verify_ldap_host', label: 'verify_ldap_host', type: 'boolean', default: false },
    { key: 'base_dn', label: 'base_dn', type: 'string', required: true, placeholder: 'dc=example,dc=com' },
    { key: 'attribute', label: 'attribute', type: 'string', required: true, placeholder: 'cn' },
    { key: 'cache_ttl', label: 'cache_ttl', type: 'number', required: true, default: 60, min: 0 },
    { key: 'hide_credentials', label: 'hide_credentials', type: 'boolean', default: false },
    { key: 'timeout', label: 'timeout', type: 'number', default: 10000, min: 0 },
    { key: 'keepalive', label: 'keepalive', type: 'number', default: 60000, min: 0 },
    { key: 'anonymous', label: 'anonymous', type: 'string' },
    { key: 'header_type', label: 'header_type', type: 'string', default: 'ldap' },
  ],
  oauth2: [
    { key: 'scopes', label: 'scopes', type: 'csv', placeholder: 'email,profile' },
    { key: 'mandatory_scope', label: 'mandatory_scope', type: 'boolean', default: false },
    { key: 'provision_key', label: 'provision_key', type: 'string', required: true, placeholder: '留空将自动生成' },
    { key: 'token_expiration', label: 'token_expiration', type: 'number', default: 7200, min: 0 },
    { key: 'enable_authorization_code', label: 'enable_authorization_code', type: 'boolean', default: false },
    { key: 'enable_implicit_grant', label: 'enable_implicit_grant', type: 'boolean', default: false },
    { key: 'enable_client_credentials', label: 'enable_client_credentials', type: 'boolean', default: false },
    { key: 'enable_password_grant', label: 'enable_password_grant', type: 'boolean', default: false },
    { key: 'hide_credentials', label: 'hide_credentials', type: 'boolean', default: false },
    { key: 'accept_http_if_already_terminated', label: 'accept_http_if_already_terminated', type: 'boolean', default: false },
    { key: 'anonymous', label: 'anonymous', type: 'string' },
    { key: 'global_credentials', label: 'global_credentials', type: 'boolean', default: false },
    { key: 'auth_header_name', label: 'auth_header_name', type: 'string', default: 'authorization' },
    { key: 'refresh_token_ttl', label: 'refresh_token_ttl', type: 'number', default: 1209600, min: 0 },
    { key: 'reuse_refresh_token', label: 'reuse_refresh_token', type: 'boolean', default: false },
    { key: 'pkce', label: 'pkce', type: 'select', default: 'lax', options: ['none', 'lax', 'strict'] },
  ],
  session: [
    { key: 'secret', label: 'secret', type: 'string', placeholder: '留空使用随机密钥' },
    { key: 'storage', label: 'storage', type: 'select', default: 'cookie', options: ['cookie', 'kong'] },
    { key: 'audience', label: 'audience', type: 'string', default: 'default' },
    { key: 'idling_timeout', label: 'idling_timeout', type: 'number', default: 900, min: 0 },
    { key: 'rolling_timeout', label: 'rolling_timeout', type: 'number', default: 3600, min: 0 },
    { key: 'absolute_timeout', label: 'absolute_timeout', type: 'number', default: 86400, min: 0 },
    { key: 'stale_ttl', label: 'stale_ttl', type: 'number', default: 10, min: 0 },
    { key: 'cookie_name', label: 'cookie_name', type: 'string', default: 'session' },
    { key: 'cookie_path', label: 'cookie_path', type: 'string', default: '/' },
    { key: 'cookie_domain', label: 'cookie_domain', type: 'string' },
    { key: 'cookie_same_site', label: 'cookie_same_site', type: 'select', default: 'Strict', options: ['Strict', 'Lax', 'None', 'Default'] },
    { key: 'cookie_http_only', label: 'cookie_http_only', type: 'boolean', default: true },
    { key: 'cookie_secure', label: 'cookie_secure', type: 'boolean', default: true },
    { key: 'remember', label: 'remember', type: 'boolean', default: false },
    { key: 'remember_cookie_name', label: 'remember_cookie_name', type: 'string', default: 'remember' },
    { key: 'logout_methods', label: 'logout_methods', type: 'multiselect', default: ['POST', 'DELETE'], options: [...SESSION_LOGOUT_METHODS] },
    { key: 'logout_query_arg', label: 'logout_query_arg', type: 'string', default: 'session_logout' },
  ],
  acme: [
    { key: 'account_email', label: 'account_email', type: 'string', required: true, placeholder: 'admin@example.com' },
    {
      key: 'api_uri',
      label: 'api_uri',
      type: 'select',
      default: 'https://acme-v02.api.letsencrypt.org/directory',
      options: [
        'https://acme-v02.api.letsencrypt.org/directory',
        'https://acme-staging-v02.api.letsencrypt.org/directory',
      ],
    },
    { key: 'tos_accepted', label: 'tos_accepted', type: 'boolean', default: false },
    { key: 'cert_type', label: 'cert_type', type: 'select', default: 'rsa', options: ['rsa', 'ecc'] },
    { key: 'rsa_key_size', label: 'rsa_key_size', type: 'select', default: 4096, options: [2048, 3072, 4096] },
    { key: 'renew_threshold_days', label: 'renew_threshold_days', type: 'number', default: 14, min: 0 },
    { key: 'domains', label: 'domains', type: 'csv', placeholder: 'example.com,*.example.com' },
    { key: 'allow_any_domain', label: 'allow_any_domain', type: 'boolean', default: false },
    { key: 'fail_backoff_minutes', label: 'fail_backoff_minutes', type: 'number', default: 5, min: 0 },
    { key: 'storage', label: 'storage', type: 'select', default: 'shm', options: ['kong', 'shm', 'redis', 'consul', 'vault'] },
    { key: 'storage_config', label: 'storage_config', type: 'json', default: '{}', hint: 'Kong 3.4.2 storage_config 对象' },
  ],
  'bot-detection': [
    { key: 'allow', label: 'allow', type: 'csv', placeholder: 'User-Agent 正则，逗号分隔' },
    { key: 'deny', label: 'deny', type: 'csv', placeholder: 'User-Agent 正则，逗号分隔' },
  ],
  'proxy-cache': [
    { key: 'response_code', label: 'response_code', type: 'csv', default: '200,301,404', required: true },
    { key: 'request_method', label: 'request_method', type: 'multiselect', default: ['GET', 'HEAD'], required: true, options: [...PROXY_CACHE_METHODS] },
    { key: 'content_type', label: 'content_type', type: 'csv', default: 'text/plain,application/json', required: true },
    { key: 'cache_ttl', label: 'cache_ttl', type: 'number', default: 300, min: 1 },
    { key: 'strategy', label: 'strategy', type: 'select', required: true, default: 'memory', options: ['memory'] },
    { key: 'cache_control', label: 'cache_control', type: 'boolean', default: false },
    { key: 'ignore_uri_case', label: 'ignore_uri_case', type: 'boolean', default: false },
    { key: 'storage_ttl', label: 'storage_ttl', type: 'number', min: 0 },
    { key: 'memory.dictionary_name', label: 'memory.dictionary_name', type: 'string', default: 'kong_db_cache' },
    { key: 'vary_query_params', label: 'vary_query_params', type: 'csv' },
    { key: 'vary_headers', label: 'vary_headers', type: 'csv' },
  ],
  'response-ratelimiting': [
    { key: 'header_name', label: 'header_name', type: 'string', default: 'x-kong-limit' },
    { key: 'limit_by', label: 'limit_by', type: 'select', default: 'consumer', options: ['consumer', 'credential', 'ip'] },
    { key: 'policy', label: 'policy', type: 'select', default: 'local', options: ['local', 'cluster', 'redis'] },
    { key: 'fault_tolerant', label: 'fault_tolerant', type: 'boolean', default: true },
    { key: 'block_on_first_violation', label: 'block_on_first_violation', type: 'boolean', default: false },
    { key: 'hide_client_headers', label: 'hide_client_headers', type: 'boolean', default: false },
    { key: 'limits', label: 'limits', type: 'json', required: true, default: '{"video":{"minute":100}}', hint: 'map：限流名 -> {second|minute|hour|...}' },
    { key: 'redis_host', label: 'redis_host', type: 'string' },
    { key: 'redis_port', label: 'redis_port', type: 'number', default: 6379, min: 1, max: 65535 },
    { key: 'redis_password', label: 'redis_password', type: 'string' },
    { key: 'redis_username', label: 'redis_username', type: 'string' },
    { key: 'redis_database', label: 'redis_database', type: 'number', default: 0, min: 0 },
    { key: 'redis_timeout', label: 'redis_timeout', type: 'number', default: 2000, min: 0 },
    { key: 'redis_ssl', label: 'redis_ssl', type: 'boolean', default: false },
    { key: 'redis_ssl_verify', label: 'redis_ssl_verify', type: 'boolean', default: false },
  ],
  'aws-lambda': [
    { key: 'function_name', label: 'function_name', type: 'string', placeholder: 'my-function' },
    { key: 'aws_region', label: 'aws_region', type: 'string', placeholder: 'us-east-1' },
    { key: 'aws_key', label: 'aws_key', type: 'string' },
    { key: 'aws_secret', label: 'aws_secret', type: 'string' },
    { key: 'aws_assume_role_arn', label: 'aws_assume_role_arn', type: 'string' },
    { key: 'aws_role_session_name', label: 'aws_role_session_name', type: 'string', default: 'kong' },
    { key: 'qualifier', label: 'qualifier', type: 'string' },
    { key: 'invocation_type', label: 'invocation_type', type: 'select', default: 'RequestResponse', options: ['RequestResponse', 'Event', 'DryRun'] },
    { key: 'log_type', label: 'log_type', type: 'select', default: 'Tail', options: ['Tail', 'None'] },
    { key: 'host', label: 'host', type: 'string' },
    { key: 'port', label: 'port', type: 'number', default: 443, min: 1, max: 65535 },
    { key: 'timeout', label: 'timeout', type: 'number', default: 60000, min: 0 },
    { key: 'keepalive', label: 'keepalive', type: 'number', default: 60000, min: 0 },
    { key: 'disable_https', label: 'disable_https', type: 'boolean', default: false },
    { key: 'unhandled_status', label: 'unhandled_status', type: 'number', min: 100, max: 999 },
    { key: 'forward_request_method', label: 'forward_request_method', type: 'boolean', default: false },
    { key: 'forward_request_uri', label: 'forward_request_uri', type: 'boolean', default: false },
    { key: 'forward_request_headers', label: 'forward_request_headers', type: 'boolean', default: false },
    { key: 'forward_request_body', label: 'forward_request_body', type: 'boolean', default: false },
    { key: 'is_proxy_integration', label: 'is_proxy_integration', type: 'boolean', default: false },
    { key: 'awsgateway_compatible', label: 'awsgateway_compatible', type: 'boolean', default: false },
    { key: 'proxy_url', label: 'proxy_url', type: 'string' },
    { key: 'skip_large_bodies', label: 'skip_large_bodies', type: 'boolean', default: true },
    { key: 'base64_encode_body', label: 'base64_encode_body', type: 'boolean', default: true },
    { key: 'aws_imds_protocol_version', label: 'aws_imds_protocol_version', type: 'select', default: 'v1', options: ['v1', 'v2'] },
  ],
  'azure-functions': [
    { key: 'appname', label: 'appname', type: 'string', required: true },
    { key: 'functionname', label: 'functionname', type: 'string', required: true },
    { key: 'hostdomain', label: 'hostdomain', type: 'string', required: true, default: 'azurewebsites.net' },
    { key: 'routeprefix', label: 'routeprefix', type: 'string', default: 'api' },
    { key: 'apikey', label: 'apikey', type: 'string' },
    { key: 'clientid', label: 'clientid', type: 'string' },
    { key: 'https', label: 'https', type: 'boolean', default: true },
    { key: 'https_verify', label: 'https_verify', type: 'boolean', default: false },
    { key: 'timeout', label: 'timeout', type: 'number', default: 600000, min: 0 },
    { key: 'keepalive', label: 'keepalive', type: 'number', default: 60000, min: 0 },
  ],
  'pre-function': [
    { key: 'access', label: 'access', type: 'textarea', placeholder: 'Lua 代码（将作为 access 阶段唯一函数）', hint: '至少填写一个阶段' },
    { key: 'certificate', label: 'certificate', type: 'textarea' },
    { key: 'rewrite', label: 'rewrite', type: 'textarea' },
    { key: 'header_filter', label: 'header_filter', type: 'textarea' },
    { key: 'body_filter', label: 'body_filter', type: 'textarea' },
    { key: 'log', label: 'log', type: 'textarea' },
  ],
  'post-function': [
    { key: 'access', label: 'access', type: 'textarea', placeholder: 'Lua 代码（将作为 access 阶段唯一函数）', hint: '至少填写一个阶段' },
    { key: 'certificate', label: 'certificate', type: 'textarea' },
    { key: 'rewrite', label: 'rewrite', type: 'textarea' },
    { key: 'header_filter', label: 'header_filter', type: 'textarea' },
    { key: 'body_filter', label: 'body_filter', type: 'textarea' },
    { key: 'log', label: 'log', type: 'textarea' },
  ],
  datadog: [
    { key: 'host', label: 'host', type: 'string', default: 'localhost' },
    { key: 'port', label: 'port', type: 'number', default: 8125, min: 1, max: 65535 },
    { key: 'prefix', label: 'prefix', type: 'string', default: 'kong' },
    { key: 'service_name_tag', label: 'service_name_tag', type: 'string', default: 'name' },
    { key: 'status_tag', label: 'status_tag', type: 'string', default: 'status' },
    { key: 'consumer_tag', label: 'consumer_tag', type: 'string', default: 'consumer' },
    { key: 'metrics', label: 'metrics', type: 'json', hint: '留空使用 Kong 默认 metrics；或填写 metrics 数组 JSON' },
  ],
  opentelemetry: [
    { key: 'endpoint', label: 'endpoint', type: 'string', required: true, placeholder: 'http://otel-collector:4318/v1/traces' },
    { key: 'headers', label: 'headers', type: 'json', default: '{}', hint: 'HTTP headers 对象，如 {"Authorization":"Bearer ..."}' },
    { key: 'resource_attributes', label: 'resource_attributes', type: 'json', default: '{}' },
    { key: 'header_type', label: 'header_type', type: 'select', default: 'preserve', options: ['preserve', 'ignore', 'b3', 'b3-single', 'w3c', 'jaeger', 'ot', 'aws'] },
    { key: 'http_response_header_for_traceid', label: 'http_response_header_for_traceid', type: 'string' },
    { key: 'connect_timeout', label: 'connect_timeout', type: 'number', default: 1000, min: 0 },
    { key: 'send_timeout', label: 'send_timeout', type: 'number', default: 5000, min: 0 },
    { key: 'read_timeout', label: 'read_timeout', type: 'number', default: 5000, min: 0 },
    { key: 'batch_span_count', label: 'batch_span_count', type: 'number', min: 0 },
    { key: 'batch_flush_delay', label: 'batch_flush_delay', type: 'number', min: 0 },
  ],
  prometheus: [
    { key: 'per_consumer', label: 'per_consumer', type: 'boolean', default: false },
    { key: 'status_code_metrics', label: 'status_code_metrics', type: 'boolean', default: false },
    { key: 'latency_metrics', label: 'latency_metrics', type: 'boolean', default: false },
    { key: 'bandwidth_metrics', label: 'bandwidth_metrics', type: 'boolean', default: false },
    { key: 'upstream_health_metrics', label: 'upstream_health_metrics', type: 'boolean', default: false },
  ],
  statsd: [
    { key: 'host', label: 'host', type: 'string', default: 'localhost' },
    { key: 'port', label: 'port', type: 'number', default: 8125, min: 1, max: 65535 },
    { key: 'prefix', label: 'prefix', type: 'string', default: 'kong' },
    { key: 'use_tcp', label: 'use_tcp', type: 'boolean', default: false },
    { key: 'hostname_in_prefix', label: 'hostname_in_prefix', type: 'boolean', default: false },
    { key: 'udp_packet_size', label: 'udp_packet_size', type: 'number', default: 0, min: 0 },
    { key: 'consumer_identifier_default', label: 'consumer_identifier_default', type: 'select', default: 'custom_id', options: ['consumer_id', 'custom_id', 'username'] },
    { key: 'service_identifier_default', label: 'service_identifier_default', type: 'select', default: 'service_name_or_host', options: ['service_id', 'service_name', 'service_host', 'service_name_or_host'] },
    { key: 'workspace_identifier_default', label: 'workspace_identifier_default', type: 'select', default: 'workspace_id', options: ['workspace_id', 'workspace_name'] },
    { key: 'allow_status_codes', label: 'allow_status_codes', type: 'csv', placeholder: '200-299,301,404' },
    { key: 'metrics', label: 'metrics', type: 'json', hint: '留空使用 Kong 默认 metrics' },
  ],
  zipkin: [
    { key: 'http_endpoint', label: 'http_endpoint', type: 'string', placeholder: 'http://zipkin:9411/api/v2/spans' },
    { key: 'local_service_name', label: 'local_service_name', type: 'string', default: 'kong' },
    { key: 'sample_ratio', label: 'sample_ratio', type: 'number', default: 0.001, min: 0, max: 1 },
    { key: 'default_service_name', label: 'default_service_name', type: 'string' },
    { key: 'include_credential', label: 'include_credential', type: 'boolean', default: true },
    { key: 'traceid_byte_count', label: 'traceid_byte_count', type: 'select', default: 16, options: [8, 16] },
    { key: 'header_type', label: 'header_type', type: 'select', default: 'preserve', options: ['preserve', 'ignore', 'b3', 'b3-single', 'w3c', 'jaeger', 'ot', 'aws'] },
    { key: 'default_header_type', label: 'default_header_type', type: 'select', default: 'b3', options: ['b3', 'b3-single', 'w3c', 'jaeger', 'ot', 'aws'] },
    { key: 'tags_header', label: 'tags_header', type: 'string', default: 'Zipkin-Tags' },
    { key: 'http_span_name', label: 'http_span_name', type: 'select', default: 'method', options: ['method', 'method_path'] },
    { key: 'phase_duration_flavor', label: 'phase_duration_flavor', type: 'select', default: 'annotations', options: ['annotations', 'tags'] },
    { key: 'http_response_header_for_traceid', label: 'http_response_header_for_traceid', type: 'string' },
    { key: 'connect_timeout', label: 'connect_timeout', type: 'number', default: 2000, min: 0 },
    { key: 'send_timeout', label: 'send_timeout', type: 'number', default: 5000, min: 0 },
    { key: 'read_timeout', label: 'read_timeout', type: 'number', default: 5000, min: 0 },
    { key: 'static_tags', label: 'static_tags', type: 'json', default: '[]', hint: '[{"name":"env","value":"prod"}]' },
  ],
  'grpc-gateway': [
    { key: 'proto', label: 'proto', type: 'string', placeholder: '/path/to/file.proto' },
  ],
  'grpc-web': [
    { key: 'proto', label: 'proto', type: 'string', placeholder: '/path/to/file.proto' },
    { key: 'pass_stripped_path', label: 'pass_stripped_path', type: 'boolean', default: false },
    { key: 'allow_origin_header', label: 'allow_origin_header', type: 'string', default: '*' },
  ],
  'request-transformer': [
    { key: 'http_method', label: 'http_method', type: 'select', options: [...HTTP_METHODS] },
    { key: 'remove', label: 'remove', type: 'json', default: '{"body":[],"headers":[],"querystring":[]}' },
    { key: 'rename', label: 'rename', type: 'json', default: '{"body":[],"headers":[],"querystring":[]}' },
    { key: 'replace', label: 'replace', type: 'json', default: '{"body":[],"headers":[],"querystring":[],"uri":null}' },
    { key: 'add', label: 'add', type: 'json', default: '{"body":[],"headers":[],"querystring":[]}' },
    { key: 'append', label: 'append', type: 'json', default: '{"body":[],"headers":[],"querystring":[]}' },
  ],
  'response-transformer': [
    { key: 'remove', label: 'remove', type: 'json', default: '{"json":[],"headers":[]}' },
    { key: 'rename', label: 'rename', type: 'json', default: '{"headers":[]}' },
    { key: 'replace', label: 'replace', type: 'json', default: '{"json":[],"json_types":[],"headers":[]}' },
    { key: 'add', label: 'add', type: 'json', default: '{"json":[],"json_types":[],"headers":[]}' },
    { key: 'append', label: 'append', type: 'json', default: '{"json":[],"json_types":[],"headers":[]}' },
  ],
  'file-log': [
    { key: 'path', label: 'path', type: 'string', required: true, placeholder: '/tmp/file.log' },
    { key: 'reopen', label: 'reopen', type: 'boolean', default: false },
  ],
  'http-log': [
    { key: 'http_endpoint', label: 'http_endpoint', type: 'string', required: true, placeholder: 'http://example.com/logs' },
    { key: 'method', label: 'method', type: 'select', default: 'POST', options: ['POST', 'PUT', 'PATCH'] },
    { key: 'content_type', label: 'content_type', type: 'select', default: 'application/json', options: ['application/json', 'application/json; charset=utf-8'] },
    { key: 'timeout', label: 'timeout', type: 'number', default: 10000, min: 0 },
    { key: 'keepalive', label: 'keepalive', type: 'number', default: 60000, min: 0 },
    { key: 'headers', label: 'headers', type: 'json', default: '{}', hint: '额外 HTTP headers 对象' },
  ],
  loggly: [
    { key: 'key', label: 'key', type: 'string', required: true },
    { key: 'host', label: 'host', type: 'string', default: 'logs-01.loggly.com' },
    { key: 'port', label: 'port', type: 'number', default: 514, min: 1, max: 65535 },
    { key: 'tags', label: 'tags', type: 'csv', default: 'kong' },
    { key: 'log_level', label: 'log_level', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'successful_severity', label: 'successful_severity', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'client_errors_severity', label: 'client_errors_severity', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'server_errors_severity', label: 'server_errors_severity', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'timeout', label: 'timeout', type: 'number', default: 10000, min: 0 },
  ],
  syslog: [
    { key: 'log_level', label: 'log_level', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'successful_severity', label: 'successful_severity', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'client_errors_severity', label: 'client_errors_severity', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'server_errors_severity', label: 'server_errors_severity', type: 'select', default: 'info', options: ['debug', 'info', 'notice', 'warning', 'err', 'crit', 'alert', 'emerg'] },
    { key: 'facility', label: 'facility', type: 'select', default: 'user', options: ['auth', 'authpriv', 'cron', 'daemon', 'ftp', 'kern', 'lpr', 'mail', 'news', 'syslog', 'user', 'uucp', 'local0', 'local1', 'local2', 'local3', 'local4', 'local5', 'local6', 'local7'] },
  ],
  'tcp-log': [
    { key: 'host', label: 'host', type: 'string', required: true },
    { key: 'port', label: 'port', type: 'number', required: true, min: 1, max: 65535 },
    { key: 'timeout', label: 'timeout', type: 'number', default: 10000, min: 0 },
    { key: 'keepalive', label: 'keepalive', type: 'number', default: 60000, min: 0 },
    { key: 'tls', label: 'tls', type: 'boolean', default: false },
    { key: 'tls_sni', label: 'tls_sni', type: 'string' },
  ],
  'udp-log': [
    { key: 'host', label: 'host', type: 'string', required: true },
    { key: 'port', label: 'port', type: 'number', required: true, min: 1, max: 65535 },
    { key: 'timeout', label: 'timeout', type: 'number', default: 10000, min: 0 },
  ],
}

const PHASE_PLUGINS = new Set(['pre-function', 'post-function'])
const INT_CSV = new Set(['response_code'])

export function hasSchemaForm(plugin: string) {
  return Object.prototype.hasOwnProperty.call(pluginFormSchemas, plugin)
}

/** Form model keeps flat keys (including dotted paths like memory.dictionary_name). */
export function defaultSchemaValues(plugin: string): Record<string, unknown> {
  const fields = pluginFormSchemas[plugin] || []
  const out: Record<string, unknown> = {}
  for (const f of fields) {
    if (f.default !== undefined) {
      out[f.key] = f.type === 'json' ? String(f.default) : Array.isArray(f.default) ? [...(f.default as unknown[])] : f.default
    } else if (f.type === 'boolean') {
      out[f.key] = false
    } else if (f.type === 'number') {
      out[f.key] = undefined
    } else if (f.type === 'multiselect') {
      out[f.key] = []
    } else {
      out[f.key] = ''
    }
  }
  return out
}

export function fillSchemaValues(plugin: string, src: Record<string, unknown>): Record<string, unknown> {
  const base = defaultSchemaValues(plugin)
  const fields = pluginFormSchemas[plugin] || []
  for (const f of fields) {
    const raw = getPath(src, f.key)
    if (raw === undefined || raw === null) continue
    if (f.type === 'csv') {
      base[f.key] = Array.isArray(raw) ? raw.join(',') : String(raw)
    } else if (f.type === 'multiselect') {
      if (Array.isArray(raw)) base[f.key] = [...raw]
      else if (typeof raw === 'string' && raw.trim()) base[f.key] = raw.split(',').map((s) => s.trim()).filter(Boolean)
      else base[f.key] = []
    } else if (f.type === 'json') {
      base[f.key] = typeof raw === 'string' ? raw : JSON.stringify(raw, null, 2)
    } else if (f.type === 'textarea' && PHASE_PLUGINS.has(plugin)) {
      base[f.key] = Array.isArray(raw) ? (raw as string[]).join('\n\n') : String(raw)
    } else if (f.type === 'boolean') {
      base[f.key] = Boolean(raw)
    } else if (f.type === 'number' || (f.type === 'select' && typeof f.default === 'number')) {
      base[f.key] = Number(raw)
    } else {
      base[f.key] = raw
    }
  }
  return base
}

export function buildSchemaPayload(
  plugin: string,
  values: Record<string, unknown>,
): { ok: true; config: Record<string, unknown> } | { ok: false; error: string } {
  const fields = pluginFormSchemas[plugin]
  if (!fields) return { ok: false, error: '未知插件表单' }

  const out: Record<string, unknown> = {}

  for (const f of fields) {
    let val = values[f.key]
    if (f.type === 'string' || f.type === 'textarea') {
      val = val == null ? '' : String(val).trim()
      if (!val) {
        if (f.required) return { ok: false, error: `请填写 ${f.label}` }
        continue
      }
      if (PHASE_PLUGINS.has(plugin) && f.type === 'textarea') {
        setPath(out, f.key, [val])
      } else {
        setPath(out, f.key, val)
      }
    } else if (f.type === 'number') {
      if (val === '' || val === undefined || val === null) {
        if (f.required) return { ok: false, error: `请填写 ${f.label}` }
        continue
      }
      const n = Number(val)
      if (Number.isNaN(n)) return { ok: false, error: `${f.label} 必须是数字` }
      setPath(out, f.key, n)
    } else if (f.type === 'boolean') {
      setPath(out, f.key, Boolean(val))
    } else if (f.type === 'select') {
      if (val === '' || val === undefined || val === null) {
        if (f.required) return { ok: false, error: `请选择 ${f.label}` }
        continue
      }
      if (typeof f.default === 'number' || f.options?.every((o) => typeof o === 'number')) {
        setPath(out, f.key, Number(val))
      } else {
        setPath(out, f.key, val)
      }
    } else if (f.type === 'multiselect') {
      const arr = Array.isArray(val) ? val.map(String).filter(Boolean) : []
      if (!arr.length) {
        if (f.required) return { ok: false, error: `请选择 ${f.label}` }
        continue
      }
      setPath(out, f.key, arr)
    } else if (f.type === 'csv') {
      const text = val == null ? '' : String(val).trim()
      if (!text) {
        if (f.required) return { ok: false, error: `请填写 ${f.label}` }
        continue
      }
      const parts = text.split(',').map((s) => s.trim()).filter(Boolean)
      if (INT_CSV.has(f.key)) {
        setPath(out, f.key, parts.map((p) => Number(p)))
      } else {
        setPath(out, f.key, parts)
      }
    } else if (f.type === 'json') {
      const text = val == null ? '' : String(val).trim()
      if (!text) {
        if (f.required) return { ok: false, error: `请填写 ${f.label}` }
        continue
      }
      try {
        setPath(out, f.key, JSON.parse(text))
      } catch {
        return { ok: false, error: `${f.label} JSON 无效` }
      }
    }
  }

  if (plugin === 'oauth2') {
    if (
      !out.enable_authorization_code &&
      !out.enable_implicit_grant &&
      !out.enable_client_credentials &&
      !out.enable_password_grant
    ) {
      return { ok: false, error: 'oauth2 至少启用一种 grant' }
    }
    if (!out.provision_key) {
      out.provision_key = randomKey(32)
    }
  }

  if (plugin === 'session' && !out.secret) {
    out.secret = randomKey(32)
  }

  if (PHASE_PLUGINS.has(plugin)) {
    const phases = ['certificate', 'rewrite', 'access', 'header_filter', 'body_filter', 'log']
    if (!phases.some((p) => Array.isArray(out[p]) && (out[p] as unknown[]).length)) {
      return { ok: false, error: `${plugin} 至少配置一个阶段的 Lua` }
    }
  }

  if (plugin === 'acme' && out.tos_accepted !== true) {
    return { ok: false, error: 'acme 需要接受 ToS（tos_accepted=true）' }
  }

  return { ok: true, config: out }
}

function randomKey(len: number) {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  let s = ''
  for (let i = 0; i < len; i++) s += chars[Math.floor(Math.random() * chars.length)]
  return s
}

function getPath(obj: Record<string, unknown>, path: string): unknown {
  if (!path.includes('.')) return obj[path]
  const parts = path.split('.')
  let cur: unknown = obj
  for (const p of parts) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[p]
  }
  return cur
}

function setPath(obj: Record<string, unknown>, path: string, value: unknown) {
  if (!path.includes('.')) {
    obj[path] = value
    return
  }
  const parts = path.split('.')
  let cur: Record<string, unknown> = obj
  for (let i = 0; i < parts.length - 1; i++) {
    const p = parts[i]
    if (cur[p] == null || typeof cur[p] !== 'object') cur[p] = {}
    cur = cur[p] as Record<string, unknown>
  }
  cur[parts[parts.length - 1]] = value
}
