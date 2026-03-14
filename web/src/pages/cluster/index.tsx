import { useEffect, useState, useCallback, useRef } from 'react'
import { Card, Descriptions, Table, Tag, Spin, Button, Space, Statistic, Row, Col, Result, message } from 'antd'
import { ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, DisconnectOutlined } from '@ant-design/icons'
import type { ClusterStatus, ClusterMetrics } from '@/types'
import { clusterApi } from '@/api/cluster'

export default function ClusterPage() {
  const [status, setStatus] = useState<ClusterStatus | null>(null)
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const hasData = useRef(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [s, m] = await Promise.all([clusterApi.status(), clusterApi.metrics()])
      setStatus(s)
      setMetrics(m)
      hasData.current = true
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '无法连接 etcd 集群'
      setError(msg)
      if (hasData.current) message.error(msg)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  const healthData = metrics?.health
    ? Object.entries(metrics.health).map(([endpoint, healthy]) => ({ endpoint, healthy }))
    : []

  if (loading && !status) return <Spin style={{ display: 'block', margin: '48px auto' }} />

  if (error && !status) {
    return (
      <Result
        icon={<DisconnectOutlined />}
        title="无法连接 etcd 集群"
        subTitle={error}
        extra={<Button type="primary" icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>重试</Button>}
      />
    )
  }

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>刷新</Button>
      </Space>

      {metrics && (
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={6}><Card><Statistic title="成员数量" value={metrics.member_count} /></Card></Col>
          <Col span={6}><Card><Statistic title="DB 大小" value={(metrics.db_size / 1024 / 1024).toFixed(2)} suffix="MB" /></Card></Col>
          <Col span={6}><Card><Statistic title="Leader ID" value={metrics.leader_id} /></Card></Col>
          <Col span={6}><Card><Statistic title="etcd 版本" value={metrics.version} /></Card></Col>
        </Row>
      )}

      {status && (
        <Card title="集群成员" style={{ marginBottom: 24 }}>
          <Descriptions column={1} size="small">
            <Descriptions.Item label="Leader">{status.leader}</Descriptions.Item>
          </Descriptions>
          <Table
            rowKey="id"
            dataSource={status.members}
            pagination={false}
            size="small"
            columns={[
              { title: 'ID', dataIndex: 'id', key: 'id' },
              { title: '名称', dataIndex: 'name', key: 'name' },
              { title: 'Peer URLs', dataIndex: 'peer_urls', key: 'peer_urls', render: (urls: string[]) => urls.join(', ') },
              { title: 'Client URLs', dataIndex: 'client_urls', key: 'client_urls', render: (urls: string[]) => urls.join(', ') },
            ]}
          />
        </Card>
      )}

      {healthData.length > 0 && (
        <Card title="健康检查">
          <Table
            rowKey="endpoint"
            dataSource={healthData}
            columns={[
              { title: 'Endpoint', dataIndex: 'endpoint', key: 'endpoint' },
              {
                title: '状态', dataIndex: 'healthy', key: 'healthy',
                render: (h: boolean) => h
                  ? <Tag icon={<CheckCircleOutlined />} color="success">健康</Tag>
                  : <Tag icon={<CloseCircleOutlined />} color="error">异常</Tag>,
              },
            ]}
            pagination={false}
            size="small"
          />
        </Card>
      )}
    </>
  )
}
