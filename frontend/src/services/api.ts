import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

export interface Server {
  id: string
  name: string
  ip: string
  ssh_port: number
  ssh_user: string
  tags: string[]
  status: string
  created_at: string
  updated_at: string
}

export interface Account {
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
}

export interface AccountMapping {
  server_id: string
  account_id?: string
  auto_create?: boolean
}

export interface Subscription {
  id: string
  name: string
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
    api.post<{ message: string; accounts: Account[] }>(`/servers/${serverId}/accounts/import`),
}

export interface OperationLog {
  id: string
  operator: string
  action: string
  target_type: string
  target_id: string
  detail: Record<string, any>
  ip: string
  created_at: string
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
}

export const subscriptionAPI = {
  list: (serverId?: string) => {
    const params = serverId ? { server_id: serverId } : {}
    return api.get<Subscription[]>('/subscriptions', { params }).then(r => r.data)
  },
  listFull: () => api.get<Subscription[]>('/subscriptions/full').then(r => r.data),
  create: (data: CreateSubscriptionRequest) => api.post<Subscription>('/subscriptions', data).then(r => r.data),
  delete: (id: string) => api.delete(`/subscriptions/${id}`),
  getLink: (id: string) => api.get<{ link: string; encoded: string }>(`/subscriptions/${id}/link`).then(r => r.data),
  addAccount: (id: string, data: AccountMapping) =>
    api.post(`/subscriptions/${id}/accounts`, data),
  removeAccount: (id: string, accountId: string) =>
    api.delete(`/subscriptions/${id}/accounts/${accountId}`),
  updateAccountOrder: (id: string, order: { id: string; sort_order: number }[]) =>
    api.put(`/subscriptions/${id}/accounts/order`, order),
  getAccounts: (id: string) =>
    api.get<AccountWithServer[]>(`/subscriptions/${id}/accounts`).then(r => r.data),
}

export const logAPI = {
  list: (params?: { start_time?: string; end_time?: string; target_type?: string }) =>
    api.get<OperationLog[]>('/logs/operation', { params }).then(r => r.data),
  getNodeStatuses: () => api.get<NodeStatus[]>('/logs/node-status').then(r => r.data),
}