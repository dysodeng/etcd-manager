import type { KVItem } from '@/types'

export interface KVTreeNode {
  key: string       // 完整路径，用作 Tree 的 key
  title: string     // 当前层级名称（最后一段）
  isLeaf: boolean
  kvItem?: KVItem   // 叶子节点携带原始 KV 数据
  children?: KVTreeNode[]
}

export function buildKVTree(items: KVItem[]): KVTreeNode[] {
  const root: KVTreeNode = { key: '', title: '', isLeaf: false, children: [] }

  for (const item of items) {
    const parts = item.key.split('/').filter(Boolean)
    let current = root

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]!
      const pathSoFar = '/' + parts.slice(0, i + 1).join('/')
      const isLast = i === parts.length - 1

      if (!current.children) current.children = []
      let child = current.children.find((c) => c.key === pathSoFar)
      if (!child) {
        child = {
          key: pathSoFar,
          title: part,
          isLeaf: isLast,
          kvItem: isLast ? item : undefined,
          children: isLast ? undefined : [],
        }
        current.children.push(child)
      } else if (isLast) {
        // 已存在的目录节点同时也是一个 key（如 /app 既是目录又有值）
        child.kvItem = item
      }

      current = child
    }
  }

  // 按目录优先、名称排序
  const sortChildren = (nodes: KVTreeNode[]) => {
    nodes.sort((a, b) => {
      if (a.isLeaf !== b.isLeaf) return a.isLeaf ? 1 : -1
      return a.title.localeCompare(b.title)
    })
    for (const node of nodes) {
      if (node.children?.length) sortChildren(node.children)
    }
  }
  if (root.children) sortChildren(root.children)

  return root.children ?? []
}
