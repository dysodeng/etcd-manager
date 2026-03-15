import { useState } from 'react'
import { Tree, Card, Space, Button, Empty, Popconfirm, Tag, Descriptions } from 'antd'
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
