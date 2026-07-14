import { CopyOutlined } from '@ant-design/icons'
import { Button, message, Tooltip } from 'antd'
import { copyText } from '@/utils'

export function CopyableCode({ value, copyValue = value }: { value: string; copyValue?: string }) {
  const handleCopy = async () => {
    await copyText(copyValue)
    message.success('已复制')
  }

  return (
    <span className="copyable-code">
      <code className="copyable-code__value">{value}</code>
      <Tooltip title="复制">
        <Button type="text" size="small" aria-label="复制" icon={<CopyOutlined />} onClick={handleCopy} />
      </Tooltip>
    </span>
  )
}
