import { WarningOutlined } from '@ant-design/icons'
import { Alert, Button, Checkbox, Modal } from 'antd'
import type { EnvSyncStatus } from '@/api/sync'

interface SyncRestorePanelProps {
  statuses: EnvSyncStatus[]
  selectedIds: string[]
  open: boolean
  restoring: boolean
  onOpen: () => void
  onClose: () => void
  onDismiss: () => void
  onSelectionChange: (ids: string[]) => void
  onRestore: () => void | Promise<void>
}

export default function SyncRestorePanel({
  statuses,
  selectedIds,
  open,
  restoring,
  onOpen,
  onClose,
  onDismiss,
  onSelectionChange,
  onRestore,
}: SyncRestorePanelProps) {
  const selectedNames = statuses
    .filter((status) => selectedIds.includes(status.environment_id))
    .map((status) => status.environment_name)

  return (
    <>
      {statuses.length > 0 && (
        <Alert
          className="app-sync-alert"
          message={`检测到 ${statuses.length} 个环境的配置在当前 etcd 集群中不存在，可能需要恢复`}
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          action={<Button size="small" type="primary" onClick={onOpen}>查看详情</Button>}
          closable
          onClose={onDismiss}
        />
      )}
      <Modal
        title="配置恢复"
        open={open}
        onOk={onRestore}
        onCancel={onClose}
        confirmLoading={restoring}
        okText="恢复选中环境"
        cancelText="取消"
        destroyOnHidden
        className="app-modal app-modal--danger"
        okButtonProps={{ danger: true, disabled: selectedIds.length === 0 }}
      >
        <p>以下环境在数据库中有配置记录，但当前 etcd 集群中没有对应数据。选择要恢复的环境：</p>
        {selectedNames.length > 0 && (
          <Alert
            className="app-modal-alert"
            type="warning"
            showIcon
            message={`即将恢复：${selectedNames.join('、')}`}
            description="恢复会把数据库中保存的配置重新写入当前 etcd 集群。"
          />
        )}
        <Checkbox.Group
          value={selectedIds}
          onChange={(values) => onSelectionChange(values as string[])}
          className="app-sync-options"
        >
          {statuses.map((status) => (
            <Checkbox key={status.environment_id} value={status.environment_id}>
              {status.environment_name}（数据库中 {status.db_key_count} 个配置）
            </Checkbox>
          ))}
        </Checkbox.Group>
      </Modal>
    </>
  )
}
