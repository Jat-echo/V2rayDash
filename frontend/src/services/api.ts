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

export interface Subscription {
  id: string
  server_id: string
  name: string
  uuid: string
  enable: boolean
  traffic_limit: number
  traffic_used: number
  created_at: string
  updated_at: string
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
  create: (data: Partial<Subscription>) => api.post<Subscription>('/subscriptions', data).then(r => r.data),
  delete: (id: string) => api.delete(`/subscriptions/${id}`),
  getLink: (id: string) => api.get<{ link: string; encoded: string }>(`/subscriptions/${id}/link`).then(r => r.data),
}

export const logAPI = {
  list: (params?: { start_time?: string; end_time?: string; target_type?: string }) =>
    api.get<OperationLog[]>('/logs/operation', { params }).then(r => r.data),
}