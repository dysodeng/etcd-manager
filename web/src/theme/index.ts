import type { ThemeConfig } from 'antd'
import { theme as antTheme } from 'antd'

export function createAppTheme(isDark: boolean): ThemeConfig {
  return {
    algorithm: isDark ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
    token: {
      colorPrimary: '#316ff6',
      colorSuccess: '#1d9b7d',
      colorWarning: '#d98b24',
      colorError: '#dc5151',
      colorInfo: '#316ff6',
      colorBgBase: isDark ? '#0d1420' : '#f5f7fb',
      colorTextBase: isDark ? '#e7edf6' : '#172033',
      borderRadius: 10,
      controlHeight: 36,
      fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    },
    components: {
      Button: { fontWeight: 600, primaryShadow: '0 8px 18px rgba(49, 111, 246, 0.20)' },
      Card: { paddingLG: 20 },
      Table: { headerBg: isDark ? '#151f2d' : '#f7f9fc', headerColor: isDark ? '#98a7ba' : '#6f7d91' },
      Menu: { darkItemBg: '#101a2f', darkSubMenuItemBg: '#101a2f', darkItemSelectedBg: '#244b8e' },
    },
  }
}
