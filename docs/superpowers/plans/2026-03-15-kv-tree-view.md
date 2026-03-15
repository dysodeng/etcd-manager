# KV 管理视图模式切换 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 KV 管理页面添加列表/树形视图模式切换，树形模式将同一 key 前缀的 key 按树状聚合展示。

**Architecture:** 纯前端改动。新增一个 `buildKVTree` 工具函数，将扁平的 KVItem 列表按 `/` 分隔符拆分为树形节点。页面顶部增加视图切换按钮（列表/树形），列表模式保持现有 Table，树形模式使用 Ant Design 的 Tree 组件展示。点击树形叶子节点可查看/编辑 KV。

**Tech Stack:** React 18, TypeScript, Ant Design (Tree, Segmented)

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `web/src/pages/kv/index.tsx` | 添加视图模式切换，集成树形视图 |
| Create | `web/src/pages/kv/KVTreeView.tsx` | 树形视图组件（Tree + 详情面板） |
| Create | `web/src/pages/kv/buildKVTree.ts` | KVItem[] 转树形数据结构的纯函数 |

---

## Chunk 1: 树形数据构建 + 树形视图组件 + 页面集成

### Task 1: 创建 buildKVTree 工具函数

**Files:**
- Create: `web/src/pages/kv/buildKVTree.ts`

- [ ] **Step 1: 创建 buildKVTree.ts**

该函数将扁平的 KVItem 列表按 `/` 分隔符构建为树形结构。中间节点（目录）只有 title 和 children，叶子节点携带完整 KVItem 数据。

```typescript
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
      const pathSoFar = '/' + parts.slice(0, i + 1).join('/')
      const isLast = i === parts.length - 1

      let child = current.children?.find((c) => c.key === pathSoFar)
      if (!child) {
        child = {
          key: pathSoFar,
          title: parts[i],
          isLeaf: isLast,
          kvItem: isLast ? item : undefined,
          children: isLast ? undefined : [],
        }
        current.children!.push(child)
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
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/kv/buildKVTree.ts
git commit -m "feat: add buildKVTree utility for KV tree view"
```

---

### Task 2: 创建 KVTreeView 组件

**Files:**
- Create: `web/src/pages/kv/KVTreeView.tsx`

- [ ] **Step 1: 创建 KVTreeView.tsx**

树形视图组件，左侧 Tree 导航，选中叶子节点时右侧显示详情。

```tsx
import { useState } from 'react'
import { Tree, Card, Space, Button, Empty, Popconfirm, Tag, Descriptions, message } from 'antd'
import { FileOutlined, FolderOutlined, FolderOpenOutlined } from '@ant-design/icons'
import type { KVItem } from '@/types'
import type { KVTreeNode } from './buildKVTree'
import MonacoEditor from '@/components/MonacoEditor'

interface Props {
  treeData: KVTreeNode[]
  isAdmin: boolean
  onEdit: (item: KVItem) => void
  onDelete: (key: string) => void
}

export default function KVTreeView({ treeData, isAdmin, onEdit, onDelete }: Props) {
  const [selectedNode, setSelectedNode] = useState<KVTreeNode | null>(null)

  const handleSelect = (_: unknown, info: { node: KVTreeNode }) => {
    setSelectedNode(info.node.kvItem ? info.node : null)
  }

  const renderTreeIcon = (props: { isLeaf?: boolean; expanded?: boolean }) => {
    if (props.isLeaf) return <FileOutlined />
    return props.expanded ? <FolderOpenOutlined /> : <FolderOutlined />
  }

  if (treeData.length === 0) {
    return <Empty description="暂无 KV 数据" />
  }

  return (
    <div style={{ display: 'flex', gap: 16 }}>
      <Card style={{ width: 380, minHeight: 500, overflow: 'auto' }} size="small" title="Key 树">
        <Tree<KVTreeNode>
          treeData={treeData}
          fieldNames={{ key: 'key', title: 'title', children: 'children' }}
          showIcon
          icon={renderTreeIcon}
          defaultExpandAll
          onSelect={(_, info) => handleSelect(_, info as unknown as { node: KVTreeNode })}
          selectedKeys={selectedNode ? [selectedNode.key] : []}
        />
      </Card>

      <Card style={{ flex: 1, minHeight: 500 }} size="small" title={selectedNode ? selectedNode.key : 'Key 详情'}>
        {selectedNode?.kvItem ? (
          <div>
            <Descriptions column={2} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="Key">
                <span style={{ fontFamily: 'monospace' }}>{selectedNode.kvItem.key}</span>
              </Descriptions.Item>
              <Descriptions.Item label="Version">
                <Tag>{selectedNode.kvItem.version}</Tag>
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginBottom: 12 }}>
              <MonacoEditor value={selectedNode.kvItem.value} language="json" readOnly height={360} />
            </div>

            {isAdmin && (
              <Space>
                <Button type="primary" size="small" onClick={() => onEdit(selectedNode.kvItem!)}>
                  编辑
                </Button>
                <Popconfirm title="确认删除？" onConfirm={() => { onDelete(selectedNode.kvItem!.key); setSelectedNode(null) }}>
                  <Button danger size="small">删除</Button>
                </Popconfirm>
              </Space>
            )}
          </div>
        ) : (
          <Empty description="选择左侧 Key 查看详情" />
        )}
      </Card>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/kv/KVTreeView.tsx
git commit -m "feat: add KVTreeView component"
```

---

### Task 3: 页面集成视图模式切换

**Files:**
- Modify: `web/src/pages/kv/index.tsx`

- [ ] **Step 1: 修改 KVPage，添加视图模式切换**

在 `web/src/pages/kv/index.tsx` 中：

1. 添加 imports：
```typescript
import { Segmented } from 'antd'
import { UnorderedListOutlined, ApartmentOutlined } from '@ant-design/icons'
import KVTreeView from './KVTreeView'
import { buildKVTree } from './buildKVTree'
```

2. 添加视图模式 state：
```typescript
const [viewMode, setViewMode] = useState<'list' | 'tree'>('list')
```

3. 在工具栏 `<Space>` 中，刷新按钮之后、新建按钮之前插入视图切换：
```tsx
<Segmented
  value={viewMode}
  onChange={(v) => setViewMode(v as 'list' | 'tree')}
  options={[
    { value: 'list', icon: <UnorderedListOutlined /> },
    { value: 'tree', icon: <ApartmentOutlined /> },
  ]}
/>
```

4. 将现有 `<Table .../>` 替换为条件渲染：
```tsx
{viewMode === 'list' ? (
  <Table
    rowKey="key"
    columns={columns}
    dataSource={items}
    loading={loading}
    pagination={false}
    size="middle"
  />
) : (
  <KVTreeView
    treeData={buildKVTree(items)}
    isAdmin={isAdmin}
    onEdit={openEdit}
    onDelete={handleDelete}
  />
)}
```

完整修改后的文件：

```tsx
import { useEffect, useState } from 'react'
import { Table, Button, Input, Space, Modal, Form, message, Popconfirm, Segmented } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined, UnorderedListOutlined, ApartmentOutlined } from '@ant-design/icons'
import type { KVItem } from '@/types'
import { kvApi } from '@/api/kv'
import { useAuthStore } from '@/stores/auth'
import MonacoEditor from '@/components/MonacoEditor'
import KVTreeView from './KVTreeView'
import { buildKVTree } from './buildKVTree'

export default function KVPage() {
  const [items, setItems] = useState<KVItem[]>([])
  const [loading, setLoading] = useState(false)
  const [prefix, setPrefix] = useState('/')
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<KVItem | null>(null)
  const [form] = Form.useForm()
  const [editorValue, setEditorValue] = useState('')
  const isAdmin = useAuthStore((s) => s.user?.role === 'admin')
  const [viewMode, setViewMode] = useState<'list' | 'tree'>('list')

  const fetchData = async (p?: string) => {
    setLoading(true)
    try {
      const data = await kvApi.list(p ?? prefix)
      setItems(data ?? [])
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData('/') }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setEditorValue('')
    setModalOpen(true)
  }

  const openEdit = (item: KVItem) => {
    setEditing(item)
    form.setFieldsValue({ key: item.key })
    setEditorValue(item.value)
    setModalOpen(true)
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    try {
      if (editing) {
        await kvApi.update(values.key as string, editorValue)
        message.success('更新成功')
      } else {
        await kvApi.create(values.key as string, editorValue)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const handleDelete = async (key: string) => {
    try {
      await kvApi.delete(key)
      message.success('删除成功')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    }
  }

  const columns = [
    { title: 'Key', dataIndex: 'key', key: 'key', ellipsis: true },
    {
      title: 'Value', dataIndex: 'value', key: 'value', ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v.length > 80 ? v.slice(0, 80) + '...' : v}</span>,
    },
    { title: 'Version', dataIndex: 'version', key: 'version', width: 100 },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: unknown, record: KVItem) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)} disabled={!isAdmin}>编辑</Button>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.key)} disabled={!isAdmin}>
            <Button size="small" danger disabled={!isAdmin}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Input
          prefix={<SearchOutlined />}
          placeholder="Key 前缀"
          value={prefix}
          onChange={(e) => setPrefix(e.target.value)}
          onPressEnter={() => fetchData()}
          style={{ width: 300 }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
        <Segmented
          value={viewMode}
          onChange={(v) => setViewMode(v as 'list' | 'tree')}
          options={[
            { value: 'list', icon: <UnorderedListOutlined /> },
            { value: 'tree', icon: <ApartmentOutlined /> },
          ]}
        />
        {isAdmin && <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建</Button>}
      </Space>

      {viewMode === 'list' ? (
        <Table
          rowKey="key"
          columns={columns}
          dataSource={items}
          loading={loading}
          pagination={false}
          size="middle"
        />
      ) : (
        <KVTreeView
          treeData={buildKVTree(items)}
          isAdmin={isAdmin}
          onEdit={openEdit}
          onDelete={handleDelete}
        />
      )}

      <Modal
        title={editing ? '编辑 KV' : '新建 KV'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        width={700}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="key" label="Key" rules={[{ required: true, message: '请输入 Key' }]}>
            <Input disabled={!!editing} placeholder="例如: /app/config/key" />
          </Form.Item>
          <Form.Item label="Value">
            <MonacoEditor value={editorValue} onChange={setEditorValue} height={400} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
```

- [ ] **Step 2: 验证前端编译**

Run: `cd /Users/dysodeng/project/go/etcd-manager/web && npx tsc --noEmit`
Expected: 编译成功，无类型错误

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/kv/
git commit -m "feat: add list/tree view mode toggle for KV management"
```
