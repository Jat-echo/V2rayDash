import type {Account, AccountMapping, RelayCombination, Server} from '../services/api'

export const serverRole = (server: Server) => server.role || server.node?.role || 'exit'
export const targetMapping = (key: string): AccountMapping => key.startsWith('relay:')
  ? {combination_id: key.slice(6), auto_create: true}
  : {server_id: key.slice(7), auto_create: true}
export const accountTargetKey = (account: Account) => account.combination_id ? `relay:${account.combination_id}` : `direct:${account.server_id}`
export const isDirectAccount = (account: Account, serverID: string) => account.server_id === serverID && !account.transport && !account.combination_id
export const combinationPath = (item: RelayCombination) => `${item.relay_server_name} → ${item.exit_server_name}`
export const nodeTargetOptions = (servers: Server[], combinations: RelayCombination[]) => [
  {label: '直连节点', options: servers.filter(server => serverRole(server) === 'exit').map(server => ({value: `direct:${server.id}`, label: `直连 · ${server.name}`}))},
  {label: '代理链路', options: combinations.map(item => ({value: `relay:${item.id}`, label: item.name === combinationPath(item) ? `中转 · ${item.name}` : `${item.name} · ${combinationPath(item)}`}))},
]
