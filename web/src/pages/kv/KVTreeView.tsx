import { useState } from 'react'
import { Tree, Space, Button, Popconfirm, Tag, Descriptions } from 'antd'
import { FileOutlined, FolderOutlined, FolderOpenOutlined } from '@ant-design/icons'
import type { KVItem } from '@/types'
import type { KVTreeNode } from './buildKVTree'
import MonacoEditor from '@/components/MonacoEditor'
import { CopyableCode, EmptyState } from '@/components/ui'

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
    return <EmptyState title="暂无 KV 数据" description="当前前缀下没有可展示的键值" />
  }

  return (
    <div className="resource-split">
      <section className="resource-tree" aria-label="Key 树">
        <Tree<KVTreeNode>
          treeData={treeData}
          fieldNames={{ key: 'key', title: 'title', children: 'children' }}
          showIcon
          icon={renderTreeIcon}
          defaultExpandAll={false}
          onSelect={(_, info) => handleSelect(_, info as unknown as { node: KVTreeNode })}
          selectedKeys={selectedNode ? [selectedNode.key] : []}
        />
      </section>

      <section className="resource-detail" aria-label="Key 详情">
        {selectedNode?.kvItem ? (
          <div className="resource-detail__content">
            <Descriptions className="resource-descriptions" column={2} size="small">
              <Descriptions.Item label="Key">
                <CopyableCode value={selectedNode.kvItem.key} />
              </Descriptions.Item>
              <Descriptions.Item label="Version">
                <Tag>{selectedNode.kvItem.version}</Tag>
              </Descriptions.Item>
            </Descriptions>

            <div className="resource-editor">
              <MonacoEditor value={selectedNode.kvItem.value} readOnly height={360} />
            </div>

            {isAdmin && (
              <Space>
                <Button type="primary" size="small" onClick={() => onEdit(selectedNode.kvItem!)}>
                  编辑
                </Button>
                <Popconfirm
                  title="确认删除此键值？"
                  description={`将永久删除 ${selectedNode.kvItem.key}`}
                  onConfirm={() => { onDelete(selectedNode.kvItem!.key); setSelectedNode(null) }}
                >
                  <Button danger size="small">删除</Button>
                </Popconfirm>
              </Space>
            )}
          </div>
        ) : (
          <EmptyState title="选择一个 Key" description="从左侧树中选择 Key 查看详情" />
        )}
      </section>
    </div>
  )
}
