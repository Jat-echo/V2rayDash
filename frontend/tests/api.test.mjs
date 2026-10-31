import assert from 'node:assert/strict'
import { test } from 'node:test'
import axios from 'axios'

globalThis.localStorage = { getItem: () => null }
let lastRequest
axios.defaults.adapter = async config => {
  lastRequest = config
  return {
    config,
    data: { message: 'imported', accounts: [{ id: 'account-1' }] },
    status: 200,
    statusText: 'OK',
    headers: {},
  }
}
const { accountAPI, subscriptionAPI } = await import('../src/services/api.ts')

test('account import returns the response body used by the page', async () => {
  const result = await accountAPI.import('server-1')
  assert.equal(lastRequest.url, '/servers/server-1/accounts/import')
  assert.equal(result.accounts[0].id, 'account-1')
})

test('account order uses the object expected by the backend', async () => {
  const order = [{ id: 'account-1', sort_order: 0 }]
  await subscriptionAPI.updateAccountOrder('sub-1', order)
  assert.equal(lastRequest.url, '/subscriptions/sub-1/accounts/order')
  assert.deepEqual(JSON.parse(lastRequest.data), { order })
})
