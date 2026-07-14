interface ServiceCounts {
  instance_count: number
  healthy_count: number
}

type ServiceTone = 'default' | 'success' | 'warning'

interface ServiceSummary {
  services: number
  instances: number
  healthy: number
  healthDisplay: string
  tone: ServiceTone
}

export function buildServiceSummary(groups: ServiceCounts[]): ServiceSummary {
  const instances = groups.reduce((sum, group) => sum + group.instance_count, 0)
  const healthy = groups.reduce((sum, group) => sum + group.healthy_count, 0)
  const percent = instances === 0 ? 0 : (healthy / instances) * 100
  const tone: ServiceTone = instances === 0
    ? 'default'
    : healthy === instances
      ? 'success'
      : 'warning'

  return {
    services: groups.length,
    instances,
    healthy,
    healthDisplay: instances === 0 ? '0%' : `${percent.toFixed(1)}%`,
    tone,
  }
}
