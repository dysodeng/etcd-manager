import { describe, expect, it } from 'vitest'
import type { KVItem } from '@/types'
import { buildKVTree } from './buildKVTree'

const item = (key: string, value: string): KVItem => ({ key, value, version: 1, create_revision: 1, mod_revision: 1 })

describe('buildKVTree', () => {
  it('keeps nested keys and original leaf records', () => {
    const source = [item('/services/gateway/url', 'http://gateway'), item('/services/gateway/timeout', '3s'), item('/feature', 'on')]
    const tree = buildKVTree(source)
    expect(tree.map((node) => node.key)).toEqual(['/services', '/feature'])
    const gateway = tree[0]?.children?.[0]
    expect(gateway?.key).toBe('/services/gateway')
    expect(gateway?.children?.find((node) => node.title === 'url')?.kvItem).toEqual(source[0])
  })
})
