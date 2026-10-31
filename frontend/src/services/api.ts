import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// 自动附加 token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('admin_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// 401 → 跳转登录
api.interceptors.response.use(
  r => r,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('admin_token')
      localStorage.removeItem('admin_username')
      window.location.href = '/'
    }
    return Promise.reject(err)
  }
)

export const authAPI = {
  status: () => fetch('/api/auth/status').then(r => r.json()) as Promise<{ setup_required: boolean }>,
  setup: (username: string, password: string) =>
    fetch('/api/auth/setup', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) }).then(r => r.json()),
  login: (username: string, password: string) =>
    fetch('/api/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) }).then(r => r.json()),
  logout: () => api.post('/auth/logout'),
  changePassword: (old_password: string, new_password: string) =>
    api.put('/auth/password', { old_password, new_password }).then(r => r.data),
}

export interface Server {
	role: 'exit' | 'relay'
	 node?: ManagedNode
  id: string
  name: string
  ip: string
  country_code: string
  ssh_port: number
  ssh_jump_server_id?: string
  ssh_user: string
  ssh_key_type: string
  tags: string[]
  status: string
  created_at: string
  updated_at: string
}

export interface ManagedNode {
  server_id: string
  role: 'relay' | 'exit'
  layout: 'relay' | 'exit' | 'legacy'
  inbound_tag: string
  short_id: string
  egress_ip: string
  state: 'adopted' | 'pending' | 'publishing' | 'active' | 'failed'
  last_error: string
  config_hash: string
  updated_at: string
}

export interface CombinationRequest {
  name: string
  relay_server_id: string
  exit_server_id: string
}

export interface RelayCombination extends CombinationRequest {
  id: string
  relay_server_name: string
  exit_server_name: string
  account_count: number
  subscription_count: number
  connection_ready: boolean
  created_at: string
}

export interface BatchSubscriptionResult {
  id?: string
  name: string
  status: 'created' | 'added' | 'removed' | 'skipped' | 'failed'
  error?: string
}

export const relayAPI = {
  list: () => api.get<RelayCombination[]>('/relay-combinations').then(r => r.data),
  create: (data: CombinationRequest) => api.post<{id: string}>('/relay-combinations', data).then(r => r.data),
  update: (id: string, data: CombinationRequest) => api.put(`/relay-combinations/${id}`, data),
  delete: (id: string) => api.delete(`/relay-combinations/${id}`),
  adopt: (id: string, layout: string) => api.post<ManagedNode>(`/servers/${id}/adopt`, { layout }, { timeout: 120000 }).then(r => r.data),
  publish: (id: string) => api.post(`/servers/${id}/publish`).then(r => r.data),
  monitor: (id: string) => api.post(`/servers/${id}/monitor`, {}, { timeout: 180000 }).then(r => r.data),
}

export interface Account {
  combination_id?: string
  combination_name?: string
  transport: boolean
  id: string
  server_id: string
  uuid: string
  email: string
  protocols: string[]
  enabled: boolean
  traffic_limit: number
  traffic_used: number
  created_at: string
  updated_at: string
}

export interface AccountWithServer extends Account {
  server_name: string
  server_ip: string
  server_country_code: string
}

export interface AccountMapping {
  server_id?: string
  combination_id?: string
  account_id?: string
  auto_create?: boolean
}

export interface Subscription {
  id: string
  name: string
  remark: string
  uuid: string
  enable: boolean
  traffic_limit: number
  traffic_used: number
  created_at: string
  updated_at: string
  accounts?: AccountWithServer[]
}

export interface CreateSubscriptionRequest {
  name: string
  remark?: string
  traffic_limit?: number
  account_mappings: AccountMapping[]
}

export const accountAPI = {
  listByServer: (serverId: string) =>
    api.get<Account[]>(`/servers/${serverId}/accounts`).then(r => r.data),
  get: (id: string) => api.get<Account>(`/accounts/${id}`).then(r => r.data),
  create: (serverId: string, data: Partial<Account>) =>
    api.post<Account>(`/servers/${serverId}/accounts`, data).then(r => r.data),
  update: (id: string, data: Partial<Account>) =>
    api.put(`/accounts/${id}`, data),
  delete: (id: string) => api.delete(`/accounts/${id}`),
  subscribe: (id: string, type?: string) =>
    api.get(`/accounts/${id}/subscribe`, { params: { type } }).then(r => r.data),
  import: (serverId: string) =>
    api.post<{ message: string; accounts: Account[] }>(`/servers/${serverId}/accounts/import`).then(r => r.data),
}

export interface MetricPoint {
  time: string
  value: number
}

export interface BandwidthPoint {
  time: string
  value: number
}

export interface NodeStatusMetrics {
  cpu: MetricPoint[]
  memory: MetricPoint[]
  disk: MetricPoint[]
  bandwidth_in: BandwidthPoint[]
  bandwidth_out: BandwidthPoint[]
}

export interface NodeStatusCurrent {
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  bandwidth_in: number
  bandwidth_out: number
  v2ray_status: string
  reported_at: string
}

export interface NodeStatusResponse {
  server_id: string
  metrics: NodeStatusMetrics
  current: NodeStatusCurrent
  v2ray_restarts: number
}

export interface NodeStatus {
  id: string
  server_id: string
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  bandwidth_in: number
  bandwidth_out: number
  v2ray_status: string
  reported_at: string
}

export const serverAPI = {
  list: () => api.get<Server[]>('/servers').then(r => r.data),
  get: (id: string) => api.get<Server>(`/servers/${id}`).then(r => r.data),
  create: (data: Partial<Server>) => api.post<Server>('/servers', data).then(r => r.data),
  update: (id: string, data: Partial<Server>) => api.put(`/servers/${id}`, data),
  delete: (id: string) => api.delete(`/servers/${id}`),
  restartXray: (id: string) => api.post(`/servers/${id}/restart-xray`).then(r => r.data),
  syncXray: (id: string) => api.post(`/servers/${id}/sync-xray`).then(r => r.data),
  pingAll: () =>
    api.post<{ results: { server_id: string; latency_ms: number }[] }>('/servers/ping').then(r => r.data),
}

export interface AccountTrafficSeries {
  account_id: string
  email: string
  server_name: string
  points: BandwidthPoint[]
}

export const subscriptionAPI = {
  batch: (data: {action?: 'add' | 'remove'; names?: string[]; subscription_ids?: string[]; server_id?: string; combination_id?: string; traffic_limit?: number; remark?: string}) =>
    api.post<{results: BatchSubscriptionResult[]}>('/subscriptions/batch', data, {timeout: 120000}).then(r => r.data),
  list: (serverId?: string) => {
    const params = serverId ? { server_id: serverId } : {}
    return api.get<Subscription[]>('/subscriptions', { params }).then(r => r.data)
  },
  listFull: () => api.get<Subscription[]>('/subscriptions/full').then(r => r.data),
  create: (data: CreateSubscriptionRequest) => api.post<Subscription>('/subscriptions', data).then(r => r.data),
  update: (id: string, data: { name?: string; remark?: string; enable?: boolean; traffic_limit?: number }) =>
    api.put(`/subscriptions/${id}`, data),
  delete: (id: string) => api.delete(`/subscriptions/${id}`),
  getLink: (id: string) => api.get<{ link: string; encoded: string }>(`/subscriptions/${id}/link`).then(r => r.data),
  addAccount: (id: string, data: AccountMapping) =>
    api.post(`/subscriptions/${id}/accounts`, data),
  removeAccount: (id: string, accountId: string) =>
    api.delete(`/subscriptions/${id}/accounts/${accountId}`),
  updateAccountOrder: (id: string, order: { id: string; sort_order: number }[]) =>
    api.put(`/subscriptions/${id}/accounts/order`, { order }),
  getAccounts: (id: string) =>
    api.get<AccountWithServer[]>(`/subscriptions/${id}/accounts`).then(r => r.data),
  getTrafficLogs: (id: string, range: string = '1d') =>
    api.get<BandwidthPoint[]>(`/subscriptions/${id}/traffic`, { params: { range } }).then(r => r.data),
  getAccountTrafficLogs: (id: string, range: string = '1d') =>
    api.get<AccountTrafficSeries[]>(`/subscriptions/${id}/accounts/traffic`, { params: { range } }).then(r => r.data),
}

export const logAPI = {
  getNodeStatuses: (timeRange: string = '1h', serverId?: string): Promise<NodeStatusResponse[]> => {
  const params = new URLSearchParams({ time_range: timeRange })
  if (serverId) params.append('server_id', serverId)
  return api.get(`/logs/node-status?${params.toString()}`).then(r => r.data)
},
}
