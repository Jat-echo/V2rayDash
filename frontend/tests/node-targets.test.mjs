import assert from 'node:assert/strict'
import {test} from 'node:test'
import {accountTargetKey, isDirectAccount} from '../src/utils/node-targets.ts'

test('direct account management excludes relay customers and automatic upstream credentials', () => {
  assert.equal(isDirectAccount({server_id:'landing'}, 'landing'), true)
  assert.equal(isDirectAccount({server_id:'landing',transport:true}, 'landing'), false)
  assert.equal(isDirectAccount({server_id:'relay',combination_id:'path-a'}, 'relay'), false)
  assert.equal(isDirectAccount({server_id:'another'}, 'landing'), false)
})

test('two paths on the same relay remain independent subscription targets', () => {
  assert.notEqual(accountTargetKey({server_id:'relay',combination_id:'path-a'}), accountTargetKey({server_id:'relay',combination_id:'path-b'}))
  assert.equal(accountTargetKey({server_id:'landing'}), 'direct:landing')
})
