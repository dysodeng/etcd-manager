# ETCD Manager Web UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the complete ETCD Manager web application as a clean, high-density cloud console while preserving every existing route, API call, permission rule, environment behavior, and theme mode.

**Architecture:** Keep the current React pages as data-owning containers and add a small presentation layer made of theme tokens, reusable page primitives, and focused layout components. Migrate pages by interaction pattern so each commit remains usable, test pure transformations and component structure with Vitest, and use production builds plus browser checks for CSS behavior.

**Tech Stack:** React 18, TypeScript, Vite 8, Ant Design 5, Zustand, Vitest 4, CSS custom properties, React DOM server rendering for structure tests

---

## File map

### New theme and style files

- `web/src/theme/index.ts` — builds Ant Design light/dark theme configuration.
- `web/src/theme/index.test.ts` — verifies semantic theme values.
- `web/src/styles/tokens.css` — light/dark CSS variables for surfaces, text, status, spacing, radius, shadow, and typography.
- `web/src/styles/primitives.css` — reusable page header, toolbar, card, badge, async-state, and code styles.
- `web/src/styles/layout.css` — application shell, sidebar, top bar, content viewport, and modal layout.
- `web/src/styles/pages.css` — shared page-specific grids, tables, resource views, service groups, and pagination styles.
- `web/src/styles/login.css` — isolated login composition and decoration.

### New presentation components

- `web/src/components/ui/PageHeader.tsx` — page identity and primary action.
- `web/src/components/ui/PageToolbar.tsx` — search, filters, secondary actions, and view controls.
- `web/src/components/ui/SectionCard.tsx` — titled content surface.
- `web/src/components/ui/MetricCard.tsx` — summary metric with semantic tone.
- `web/src/components/ui/StatusBadge.tsx` — consistent semantic status treatment.
- `web/src/components/ui/AsyncState.tsx` — loading, empty, and error presentations.
- `web/src/components/ui/CopyableCode.tsx` — monospaced value with copy affordance.
- `web/src/components/ui/index.ts` — stable public exports.
- `web/src/components/ui/ui.test.tsx` — server-rendered structure and semantics tests.

### New layout components

- `web/src/layouts/components/AppSidebar.tsx` — grouped permission-aware navigation.
- `web/src/layouts/components/AppHeader.tsx` — breadcrumb, environment, theme, and account controls.
- `web/src/layouts/components/PasswordModal.tsx` — password form and submission state.
- `web/src/layouts/components/EnvironmentManager.tsx` — environment table and editor modal.
- `web/src/layouts/components/SyncRestorePanel.tsx` — recovery alert and selection modal.
- `web/src/layouts/components/AppShell.test.tsx` — shell structure test.

### New page helpers and tests

- `web/src/pages/login/LoginView.tsx` — pure login presentation.
- `web/src/pages/login/LoginView.test.tsx` — login structure test.
- `web/src/pages/cluster/presentation.ts` and `.test.ts` — cluster metric formatting.
- `web/src/pages/kv/buildKVTree.test.ts` — protects tree behavior while changing layout.
- `web/src/pages/services/presentation.ts` and `.test.ts` — shared gateway/gRPC summary formatting.
- `web/src/pages/roles/permissions.ts` and `.test.ts` — preserves read/write permission coupling.

### Existing files modified

- `web/vite.config.ts`, `web/src/App.tsx`, `web/src/global.css`.
- `web/src/config/menu.ts`, `web/src/config/menu.test.ts`.
- `web/src/layouts/MainLayout.tsx`.
- `web/src/pages/login/index.tsx`.
- `web/src/pages/cluster/index.tsx`.
- `web/src/pages/kv/index.tsx`, `web/src/pages/kv/KVTreeView.tsx`.
- `web/src/pages/config/index.tsx`.
- `web/src/pages/gateway/index.tsx`, `web/src/pages/grpc/index.tsx`.
- `web/src/pages/users/index.tsx`, `web/src/pages/roles/index.tsx`, `web/src/pages/audit/index.tsx`.
- `web/src/components/MonacoEditor.tsx` only if its container needs the shared editor class; do not change Monaco behavior.

## Task 1: Establish theme tokens and test support for TSX

**Files:**
- Modify: `web/vite.config.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/global.css`
- Create: `web/src/theme/index.ts`
- Create: `web/src/theme/index.test.ts`
- Create: `web/src/styles/tokens.css`

- [ ] **Step 1: Write the failing theme test**

Create `web/src/theme/index.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { createAppTheme } from './index'

describe('createAppTheme', () => {
  it('uses the shared brand color in light mode', () => {
    const config = createAppTheme(false)
    expect(config.token?.colorPrimary).toBe('#316ff6')
    expect(config.token?.borderRadius).toBe(10)
  })

  it('uses dark surfaces without changing the brand color', () => {
    const config = createAppTheme(true)
    expect(config.token?.colorPrimary).toBe('#316ff6')
    expect(config.token?.colorBgBase).toBe('#0d1420')
  })
})
```

- [ ] **Step 2: Run the test and verify the missing module failure**

Run:

```bash
cd web && npm test -- src/theme/index.test.ts
```

Expected: FAIL because `web/src/theme/index.ts` does not exist.

- [ ] **Step 3: Implement the Ant Design theme factory**

Create `web/src/theme/index.ts`:

```ts
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
```

- [ ] **Step 4: Add global CSS tokens**

Create `web/src/styles/tokens.css` with both themes:

```css
:root {
  color-scheme: light;
  --app-bg: #f5f7fb;
  --surface: #ffffff;
  --surface-subtle: #f7f9fc;
  --surface-hover: #eef3fb;
  --border: #e2e7ef;
  --border-strong: #d5dce7;
  --text: #172033;
  --text-secondary: #6f7d91;
  --text-muted: #94a0b1;
  --sidebar: #101a2f;
  --sidebar-surface: #15233b;
  --primary: #316ff6;
  --primary-soft: #eaf1ff;
  --success: #1d9b7d;
  --success-soft: #e7f7f2;
  --warning: #d98b24;
  --warning-soft: #fff5e5;
  --danger: #dc5151;
  --danger-soft: #fff0f0;
  --radius-sm: 8px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --shadow-card: 0 6px 18px rgba(21, 33, 58, 0.05);
  --shadow-float: 0 18px 48px rgba(15, 26, 47, 0.16);
  --font-mono: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
}

:root[data-theme='dark'] {
  color-scheme: dark;
  --app-bg: #0d1420;
  --surface: #121c29;
  --surface-subtle: #151f2d;
  --surface-hover: #1b2939;
  --border: #273546;
  --border-strong: #344458;
  --text: #e7edf6;
  --text-secondary: #a0aec0;
  --text-muted: #76869a;
  --sidebar: #080f19;
  --sidebar-surface: #111d2d;
  --primary-soft: #182b52;
  --success-soft: #123a34;
  --warning-soft: #442f18;
  --danger-soft: #431f26;
  --shadow-card: none;
  --shadow-float: 0 18px 48px rgba(0, 0, 0, 0.35);
}
```

- [ ] **Step 5: Wire the theme and CSS into the application**

Replace the imports and inline `ConfigProvider` theme in `web/src/App.tsx`:

```diff
+import { useEffect } from 'react'
-import { ConfigProvider, theme as antTheme } from 'antd'
+import { ConfigProvider } from 'antd'
+import { createAppTheme } from '@/theme'

-      theme={{
-        algorithm: isDark ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
-      }}
+      theme={createAppTheme(isDark)}
```

Set the DOM theme attribute in an effect:

```tsx
useEffect(() => {
  document.documentElement.dataset.theme = isDark ? 'dark' : 'light'
}, [isDark])
```

Replace `web/src/global.css` with imports and the reset:

```css
@import './styles/tokens.css';

* { box-sizing: border-box; }
html, body, #root { width: 100%; min-width: 1280px; height: 100%; }
body { margin: 0; background: var(--app-bg); color: var(--text); font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
button, input, textarea, select { font: inherit; }
```

- [ ] **Step 6: Allow TSX tests and verify the foundation**

Change the Vite test include:

```ts
test: {
  environment: 'node',
  include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
},
```

Run:

```bash
cd web && npm test -- src/theme/index.test.ts && npm run typecheck && npm run build
```

Expected: theme tests PASS; typecheck and production build complete without errors.

- [ ] **Step 7: Commit**

```bash
git add web/vite.config.ts web/src/App.tsx web/src/global.css web/src/theme web/src/styles/tokens.css
git commit -m "feat(web): establish console theme tokens"
```

## Task 2: Build reusable page primitives

**Files:**
- Create: `web/src/components/ui/PageHeader.tsx`
- Create: `web/src/components/ui/PageToolbar.tsx`
- Create: `web/src/components/ui/SectionCard.tsx`
- Create: `web/src/components/ui/MetricCard.tsx`
- Create: `web/src/components/ui/StatusBadge.tsx`
- Create: `web/src/components/ui/AsyncState.tsx`
- Create: `web/src/components/ui/CopyableCode.tsx`
- Create: `web/src/components/ui/index.ts`
- Create: `web/src/components/ui/ui.test.tsx`
- Create: `web/src/styles/primitives.css`
- Modify: `web/src/global.css`

- [ ] **Step 1: Write failing structure tests**

Create `web/src/components/ui/ui.test.tsx`:

```tsx
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { MetricCard, PageHeader, StatusBadge } from './index'

describe('console UI primitives', () => {
  it('renders page identity and the primary action', () => {
    const html = renderToStaticMarkup(
      <PageHeader eyebrow="Cluster Overview" title="集群概览" description="实时状态" extra={<button>刷新</button>} />,
    )
    expect(html).toContain('page-header')
    expect(html).toContain('Cluster Overview')
    expect(html).toContain('实时状态')
    expect(html).toContain('刷新')
  })

  it('renders metrics and semantic status classes', () => {
    expect(renderToStaticMarkup(<MetricCard label="成员" value={3} hint="全部在线" tone="success" />)).toContain('metric-card--success')
    expect(renderToStaticMarkup(<StatusBadge tone="danger">异常</StatusBadge>)).toContain('status-badge--danger')
  })
})
```

- [ ] **Step 2: Run the test and verify exports are missing**

Run:

```bash
cd web && npm test -- src/components/ui/ui.test.tsx
```

Expected: FAIL because `web/src/components/ui/index.ts` does not exist.

- [ ] **Step 3: Implement the focused primitives**

Use these public interfaces:

```tsx
// PageHeader.tsx
import type { ReactNode } from 'react'

interface PageHeaderProps { eyebrow?: string; title: string; description?: string; extra?: ReactNode }
export function PageHeader({ eyebrow, title, description, extra }: PageHeaderProps) {
  return <header className="page-header"><div>{eyebrow && <div className="page-header__eyebrow">{eyebrow}</div>}<h1>{title}</h1>{description && <p>{description}</p>}</div>{extra && <div className="page-header__extra">{extra}</div>}</header>
}

// PageToolbar.tsx
import type { ReactNode } from 'react'
export function PageToolbar({ children, trailing }: { children: ReactNode; trailing?: ReactNode }) {
  return <div className="page-toolbar"><div className="page-toolbar__main">{children}</div>{trailing && <div className="page-toolbar__trailing">{trailing}</div>}</div>
}

// SectionCard.tsx
import type { ReactNode } from 'react'
export function SectionCard({ title, description, extra, children, className = '' }: { title?: ReactNode; description?: ReactNode; extra?: ReactNode; children: ReactNode; className?: string }) {
  return <section className={`section-card ${className}`.trim()}>{(title || description || extra) && <div className="section-card__header"><div><h2>{title}</h2>{description && <p>{description}</p>}</div>{extra}</div>}<div className="section-card__body">{children}</div></section>
}

// MetricCard.tsx
import type { ReactNode } from 'react'
type Tone = 'default' | 'primary' | 'success' | 'warning' | 'danger'
export function MetricCard({ label, value, hint, icon, tone = 'default' }: { label: ReactNode; value: ReactNode; hint?: ReactNode; icon?: ReactNode; tone?: Tone }) {
  return <article className={`metric-card metric-card--${tone}`}><div className="metric-card__top"><span>{label}</span>{icon}</div><strong>{value}</strong>{hint && <small>{hint}</small>}</article>
}

// StatusBadge.tsx
import type { ReactNode } from 'react'
export function StatusBadge({ tone, children }: { tone: 'success' | 'warning' | 'danger' | 'info' | 'neutral'; children: ReactNode }) {
  return <span className={`status-badge status-badge--${tone}`}><i aria-hidden="true" />{children}</span>
}
```

Implement `AsyncState.tsx` with explicit variants:

```tsx
import type { ReactNode } from 'react'
import { Button, Empty, Skeleton } from 'antd'

export function LoadingState({ rows = 4 }: { rows?: number }) {
  return <div className="async-state"><Skeleton active paragraph={{ rows }} /></div>
}

export function EmptyState({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return <div className="async-state"><Empty description={<><strong>{title}</strong>{description && <p>{description}</p>}</>}>{action}</Empty></div>
}

export function ErrorState({ title = '加载失败', description, onRetry }: { title?: string; description: string; onRetry?: () => void }) {
  return <div className="async-state async-state--error"><strong>{title}</strong><p>{description}</p>{onRetry && <Button onClick={onRetry}>重新加载</Button>}</div>
}
```

Implement `CopyableCode.tsx`:

```tsx
import { CopyOutlined } from '@ant-design/icons'
import { Button, message, Tooltip } from 'antd'
import { copyText } from '@/utils'

export function CopyableCode({ value, copyValue = value }: { value: string; copyValue?: string }) {
  const handleCopy = async () => {
    await copyText(copyValue)
    message.success('已复制')
  }
  return <span className="copyable-code"><code className="copyable-code__value">{value}</code><Tooltip title="复制"><Button type="text" size="small" aria-label="复制" icon={<CopyOutlined />} onClick={handleCopy} /></Tooltip></span>
}
```

Export all components from `web/src/components/ui/index.ts`:

```ts
export * from './AsyncState'
export * from './CopyableCode'
export * from './MetricCard'
export * from './PageHeader'
export * from './PageToolbar'
export * from './SectionCard'
export * from './StatusBadge'
```

- [ ] **Step 4: Add component styles**

Create `web/src/styles/primitives.css` with these required selectors and values:

```css
.page-header { display:flex; align-items:flex-end; justify-content:space-between; gap:24px; margin-bottom:20px; }
.page-header h1 { margin:4px 0; color:var(--text); font-size:24px; line-height:1.25; letter-spacing:-0.02em; }
.page-header p { margin:0; color:var(--text-secondary); }
.page-header__eyebrow { color:var(--primary); font-size:12px; font-weight:700; letter-spacing:.1em; text-transform:uppercase; }
.page-toolbar { min-height:52px; display:flex; align-items:center; justify-content:space-between; gap:16px; margin-bottom:16px; padding:8px; border:1px solid var(--border); border-radius:var(--radius-md); background:var(--surface); }
.page-toolbar__main, .page-toolbar__trailing { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.section-card { overflow:hidden; border:1px solid var(--border); border-radius:var(--radius-lg); background:var(--surface); box-shadow:var(--shadow-card); }
.section-card__header { min-height:58px; display:flex; align-items:center; justify-content:space-between; padding:14px 18px; border-bottom:1px solid var(--border); }
.section-card__header h2 { margin:0; font-size:16px; }.section-card__header p { margin:4px 0 0; color:var(--text-secondary); }
.section-card__body { min-width:0; }.metric-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:14px; margin-bottom:20px; }
.metric-card { min-height:112px; padding:18px; border:1px solid var(--border); border-radius:var(--radius-lg); background:var(--surface); box-shadow:var(--shadow-card); }
.metric-card__top { display:flex; align-items:center; justify-content:space-between; color:var(--text-secondary); }.metric-card strong { display:block; margin:12px 0 4px; font-size:24px; }.metric-card small { color:var(--text-muted); }
.metric-card--success small { color:var(--success); }.metric-card--warning small { color:var(--warning); }.metric-card--danger small { color:var(--danger); }
.status-badge { display:inline-flex; align-items:center; gap:6px; padding:4px 8px; border-radius:999px; font-size:12px; font-weight:600; }.status-badge i { width:6px; height:6px; border-radius:50%; background:currentColor; }
.status-badge--success { color:var(--success); background:var(--success-soft); }.status-badge--warning { color:var(--warning); background:var(--warning-soft); }.status-badge--danger { color:var(--danger); background:var(--danger-soft); }.status-badge--info { color:var(--primary); background:var(--primary-soft); }.status-badge--neutral { color:var(--text-secondary); background:var(--surface-subtle); }
.async-state { min-height:240px; display:flex; flex-direction:column; align-items:center; justify-content:center; padding:32px; text-align:center; }.async-state--error { color:var(--danger); }.async-state p { color:var(--text-secondary); }
.copyable-code { display:inline-flex; align-items:center; gap:6px; max-width:100%; }.copyable-code__value { overflow:hidden; color:var(--text); font-family:var(--font-mono); text-overflow:ellipsis; white-space:nowrap; }
```

Import it from `web/src/global.css` after `tokens.css`.

- [ ] **Step 5: Run tests and build**

Run:

```bash
cd web && npm test -- src/components/ui/ui.test.tsx && npm run typecheck && npm run build
```

Expected: component tests PASS; typecheck and build PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/ui web/src/styles/primitives.css web/src/global.css
git commit -m "feat(web): add console page primitives"
```

## Task 3: Redesign the login page

**Files:**
- Create: `web/src/pages/login/LoginView.tsx`
- Create: `web/src/pages/login/LoginView.test.tsx`
- Create: `web/src/styles/login.css`
- Modify: `web/src/pages/login/index.tsx`
- Modify: `web/src/global.css`

- [ ] **Step 1: Write the failing login structure test**

```tsx
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'
import LoginView from './LoginView'

describe('LoginView', () => {
  it('renders product identity, credentials, and the single primary action', () => {
    const html = renderToStaticMarkup(<LoginView loading={false} onFinish={vi.fn()} />)
    expect(html).toContain('login-page__brand')
    expect(html).toContain('登录管理中心')
    expect(html).toContain('用户名')
    expect(html).toContain('密码')
    expect(html).toContain('安全登录')
  })
})
```

- [ ] **Step 2: Verify the missing view failure**

Run `cd web && npm test -- src/pages/login/LoginView.test.tsx`.

Expected: FAIL because `LoginView.tsx` is missing.

- [ ] **Step 3: Implement the pure view and preserve the container logic**

Implement the complete pure view:

```tsx
import { LockOutlined, UserOutlined } from '@ant-design/icons'
import { Button, Form, Input } from 'antd'

interface LoginViewProps {
  loading: boolean
  onFinish: (values: { username: string; password: string }) => void | Promise<void>
}

export default function LoginView({ loading, onFinish }: LoginViewProps) {
  return <main className="login-page"><section className="login-page__brand"><div className="login-brand"><span>E</span><strong>etcd manager</strong></div><div className="login-hero"><h1>掌控集群，<br />从容管理配置。</h1><p>统一管理 etcd 集群、服务发现与配置生命周期，让基础设施状态清晰可见。</p><div className="login-features"><span>集群监控</span><span>权限控制</span><span>审计追踪</span></div></div></section><section className="login-page__form-panel"><div className="login-form"><span className="login-form__eyebrow">Welcome back</span><h2>登录管理中心</h2><p>使用管理员分配的账号继续</p><Form layout="vertical" size="large" onFinish={onFinish}><Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}><Input prefix={<UserOutlined />} placeholder="请输入用户名" autoComplete="username" /></Form.Item><Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}><Input.Password prefix={<LockOutlined />} placeholder="请输入密码" autoComplete="current-password" /></Form.Item><Button type="primary" htmlType="submit" loading={loading} block>安全登录</Button></Form><small className="login-form__security">连接已加密 · 请勿共享账号凭据</small></div></section></main>
}
```

Keep `LoginPage` in `index.tsx` responsible only for `useNavigate`, `useAuthStore`, loading state, and error message; replace its JSX with `<LoginView loading={loading} onFinish={onFinish} />`.

- [ ] **Step 4: Implement the responsive-free desktop login styles**

Create `web/src/styles/login.css` with a two-column `minmax(520px, 1.05fr) minmax(480px, .95fr)` grid, dark brand background, non-interactive radial gradients, a form width of `400px`, and dark-theme surface variables. Do not add mobile breakpoints; the global 1280px minimum applies.

```css
.login-page { min-height:100%; display:grid; grid-template-columns:minmax(520px,1.05fr) minmax(480px,.95fr); background:var(--app-bg); }
.login-page__brand { position:relative; overflow:hidden; padding:48px; color:#fff; background:#101a2f; }
.login-page__brand::before, .login-page__brand::after { content:""; position:absolute; border-radius:50%; pointer-events:none; }
.login-page__brand::before { width:520px; height:520px; top:-260px; right:-180px; background:radial-gradient(circle,#316ff6 0,rgba(49,111,246,.18) 48%,transparent 70%); }
.login-page__brand::after { width:420px; height:420px; left:-220px; bottom:-220px; background:radial-gradient(circle,#1d9b7d 0,rgba(29,155,125,.14) 50%,transparent 70%); }
.login-brand, .login-hero { position:relative; z-index:1; }.login-brand { display:flex; align-items:center; gap:12px; }.login-brand > span { width:38px; height:38px; display:grid; place-items:center; border-radius:11px; background:linear-gradient(135deg,#4f7cff,#2dc4ae); font-weight:800; }
.login-hero { position:absolute; left:48px; right:48px; bottom:14%; }.login-hero h1 { margin:0 0 18px; font-size:44px; line-height:1.16; letter-spacing:-.04em; }.login-hero p { max-width:520px; color:#a8b8cd; font-size:16px; line-height:1.8; }
.login-features { display:flex; gap:10px; margin-top:24px; }.login-features span { padding:7px 11px; border:1px solid rgba(255,255,255,.12); border-radius:999px; background:rgba(255,255,255,.06); color:#c9d5e5; font-size:12px; }
.login-page__form-panel { display:grid; place-items:center; padding:64px; background:var(--surface); }.login-form { width:400px; }.login-form__eyebrow { color:var(--primary); font-size:12px; font-weight:800; letter-spacing:.12em; text-transform:uppercase; }.login-form h2 { margin:8px 0; font-size:30px; }.login-form > p { margin:0 0 30px; color:var(--text-secondary); }.login-form__security { display:block; margin-top:18px; color:var(--text-muted); text-align:center; }
```

Import `login.css` from `global.css`.

- [ ] **Step 5: Verify login structure and build**

Run:

```bash
cd web && npm test -- src/pages/login/LoginView.test.tsx && npm run typecheck && npm run build
```

Expected: PASS with one login structure test and a successful production build.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/login web/src/styles/login.css web/src/global.css
git commit -m "feat(web): redesign login experience"
```

## Task 4: Split and redesign the application shell

**Files:**
- Modify: `web/src/config/menu.ts`
- Modify: `web/src/config/menu.test.ts`
- Modify: `web/src/layouts/MainLayout.tsx`
- Create: `web/src/layouts/components/AppSidebar.tsx`
- Create: `web/src/layouts/components/AppHeader.tsx`
- Create: `web/src/layouts/components/PasswordModal.tsx`
- Create: `web/src/layouts/components/EnvironmentManager.tsx`
- Create: `web/src/layouts/components/SyncRestorePanel.tsx`
- Create: `web/src/layouts/components/AppShell.test.tsx`
- Create: `web/src/styles/layout.css`
- Modify: `web/src/global.css`

- [ ] **Step 1: Write failing navigation-group tests**

Extend `menu.test.ts`:

```ts
import { getVisibleMenuGroups } from './menu'

it('keeps visible menu items in the approved console groups', () => {
  const superUser: UserProfile = { ...roleUser, is_super: true, role: null }
  expect(getVisibleMenuGroups(superUser).map((group) => [group.label, group.items.map((item) => item.key)])).toEqual([
    ['资源管理', ['/cluster', '/kv', '/config']],
    ['服务治理', ['/gateway', '/grpc']],
    ['系统管理', ['/users', '/roles', '/audit']],
  ])
})
```

- [ ] **Step 2: Run the test and verify the missing export**

Run `cd web && npm test -- src/config/menu.test.ts`.

Expected: FAIL because `getVisibleMenuGroups` is not exported.

- [ ] **Step 3: Add stable navigation group metadata**

Add `section: 'resources' | 'services' | 'system'` to `MenuItemConfig`, add the matching section to all eight items, and export:

```ts
const sections = [
  { key: 'resources', label: '资源管理' },
  { key: 'services', label: '服务治理' },
  { key: 'system', label: '系统管理' },
] as const

export function getVisibleMenuGroups(user: UserProfile | null) {
  const visible = new Set(getVisibleMenuKeys(user))
  return sections.map((section) => ({
    ...section,
    items: menuItemConfigs.filter((item) => item.section === section.key && visible.has(item.key)),
  })).filter((section) => section.items.length > 0)
}
```

- [ ] **Step 4: Write a failing shell structure test**

Create `AppShell.test.tsx`:

```tsx
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'
import type { Environment, UserProfile } from '@/types'
import AppHeader from './AppHeader'
import AppSidebar from './AppSidebar'

const user: UserProfile = { user_id: 'admin-1', username: 'admin', is_super: true, role: null }
const environment: Environment = { id: 'env-1', name: 'Production', key_prefix: '/prod/', config_prefix: 'config/', gateway_prefix: 'gateway/', grpc_prefix: 'grpc/', description: '生产环境', sort_order: 1, created_at: '', updated_at: '' }

describe('application shell', () => {
  it('renders grouped navigation and brand identity', () => {
    const html = renderToStaticMarkup(<AppSidebar user={user} pathname="/cluster" onNavigate={vi.fn()} />)
    expect(html).toContain('app-sidebar')
    expect(html).toContain('etcd manager')
    expect(html).toContain('资源管理')
    expect(html).toContain('服务治理')
    expect(html).toContain('系统管理')
  })

  it('renders the current environment and account', () => {
    const html = renderToStaticMarkup(<AppHeader environments={[environment]} current={environment} user={user} canManageEnvironment themeMode="light" onEnvironmentChange={vi.fn()} onManageEnvironment={vi.fn()} onThemeChange={vi.fn()} onChangePassword={vi.fn()} onLogout={vi.fn()} />)
    expect(html).toContain('app-header')
    expect(html).toContain('Production')
    expect(html).toContain('admin')
  })
})
```

Run the test; expected: FAIL because the two components do not exist.

- [ ] **Step 5: Implement focused layout components**

Move presentation without changing behavior:

- `AppSidebar` receives `user`, `pathname`, `onNavigate`, and optional cluster summary.
- `AppHeader` receives environments, current environment, user, theme mode, and callbacks for environment change, environment management, theme change, password, and logout.
- `PasswordModal` owns its Ant Design form and receives `open`, `loading`, `onCancel`, and `onSubmit(values: { old_password: string; new_password: string })`; change the MainLayout handler to consume those values instead of reading `pwdForm`.
- `EnvironmentManager` receives all environment data and CRUD callbacks; it owns the table, editor form, and editor-open state. Its `onSave(values: EnvironmentCreateRequest, editing: Environment | null)` callback passes validated values to MainLayout, where the existing create/update API branch remains.
- `SyncRestorePanel` receives statuses, selected IDs, restoring state, and callbacks.

Keep API calls, effects, permissions, and state variables in `MainLayout`; replace the 445-line JSX with:

Before the return, subscribe to the theme store with `const themeMode = useThemeStore((state) => state.mode)` and `const setThemeMode = useThemeStore((state) => state.setMode)`; implement `handleEnvironmentChange(id)` by finding the matching environment and calling the existing `setCurrent`.

```tsx
<Layout className="app-shell">
  <AppSidebar user={user} pathname={location.pathname} onNavigate={navigate} />
  <Layout className="app-workspace">
    <AppHeader environments={environments} current={current} user={user} canManageEnvironment={canManageEnv} themeMode={themeMode} onEnvironmentChange={handleEnvironmentChange} onManageEnvironment={() => setEnvOpen(true)} onThemeChange={setThemeMode} onChangePassword={() => setPwdOpen(true)} onLogout={handleLogout} />
    <SyncRestorePanel statuses={syncStatuses} selectedIds={selectedSyncEnvs} open={syncModalOpen} restoring={restoring} onOpen={openSyncModal} onClose={() => setSyncModalOpen(false)} onSelectionChange={setSelectedSyncEnvs} onRestore={handleRestore} />
    <Content className="app-content"><Outlet /></Content>
  </Layout>
  <PasswordModal open={pwdOpen} loading={pwdLoading} onCancel={() => setPwdOpen(false)} onSubmit={handleChangePassword} />
  <EnvironmentManager open={envOpen} environments={environments} canManage={canManageEnv} onClose={() => setEnvOpen(false)} onDelete={handleEnvDelete} onSave={handleEnvSave} />
</Layout>
```

- [ ] **Step 6: Add layout styles**

Create `layout.css` with fixed `220px` sidebar, `62px` header, full-height workspace, grouped menu labels, scrollable content, and `28px` desktop content padding. Use only CSS variables from `tokens.css`; preserve Ant Design modal portals.

```css
.app-shell { width:100%; height:100vh; background:var(--app-bg); }
.app-sidebar { position:relative; z-index:2; flex:0 0 220px; width:220px; min-width:220px; max-width:220px; display:flex; flex-direction:column; background:var(--sidebar); }
.app-sidebar__brand { height:72px; display:flex; align-items:center; gap:11px; padding:0 20px; color:#fff; }.app-sidebar__groups { flex:1; overflow:auto; padding:6px 12px; }.app-sidebar__label { padding:18px 10px 7px; color:#6f819e; font-size:11px; font-weight:700; letter-spacing:.1em; text-transform:uppercase; }
.app-workspace { min-width:0; height:100%; background:var(--app-bg); }.app-header { height:62px; flex:0 0 62px; display:flex; align-items:center; justify-content:space-between; padding:0 28px; border-bottom:1px solid var(--border); background:var(--surface); }.app-header__actions { display:flex; align-items:center; gap:10px; }
.app-content { min-width:0; overflow:auto; padding:28px; background:var(--app-bg); }.app-modal-section + .app-modal-section { margin-top:20px; padding-top:20px; border-top:1px solid var(--border); }
```

Import it from `global.css`.

- [ ] **Step 7: Verify permissions, shell, and build**

Run:

```bash
cd web && npm test -- src/config/menu.test.ts src/layouts/components/AppShell.test.tsx && npm run lint && npm run typecheck && npm run build
```

Expected: navigation and shell tests PASS; lint, typecheck, and build PASS.

- [ ] **Step 8: Commit**

```bash
git add web/src/config web/src/layouts web/src/styles/layout.css web/src/global.css
git commit -m "feat(web): redesign application shell"
```

## Task 5: Migrate the cluster overview

**Files:**
- Create: `web/src/pages/cluster/presentation.ts`
- Create: `web/src/pages/cluster/presentation.test.ts`
- Modify: `web/src/pages/cluster/index.tsx`
- Modify: `web/src/styles/pages.css`
- Modify: `web/src/global.css`

- [ ] **Step 1: Write failing metric transformation tests**

Create `presentation.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { formatBytes, getFragmentation } from './presentation'

describe('cluster presentation', () => {
  it('formats storage units', () => {
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1073741824)).toBe('1.00 GB')
  })

  it('maps fragmentation to semantic tones', () => {
    expect(getFragmentation(100, 71)).toEqual({ percent: 29, tone: 'success' })
    expect(getFragmentation(100, 70)).toEqual({ percent: 30, tone: 'warning' })
    expect(getFragmentation(100, 49)).toEqual({ percent: 51, tone: 'danger' })
  })
})
```

- [ ] **Step 2: Run the test and verify the missing helper**

Run `cd web && npm test -- src/pages/cluster/presentation.test.ts`.

Expected: FAIL because `presentation.ts` does not exist.

- [ ] **Step 3: Implement pure presentation helpers**

Move `formatBytes` and fragmentation calculations from the page into `presentation.ts`:

```ts
import type { ClusterMetrics } from '@/types'

export type MetricTone = 'success' | 'warning' | 'danger'
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}
export function getFragmentation(dbSize: number, dbSizeInUse: number): { percent: number; tone: MetricTone } {
  const percent = dbSize > 0 ? Math.round((1 - dbSizeInUse / dbSize) * 100) : 0
  return { percent, tone: percent > 50 ? 'danger' : percent >= 30 ? 'warning' : 'success' }
}
export function buildClusterMetricView(metrics: ClusterMetrics) {
  const fragmentation = getFragmentation(metrics.db_size, metrics.db_size_in_use)
  const healthValues = Object.values(metrics.health)
  const healthy = healthValues.filter(Boolean).length
  return [
    { key: 'members', label: '集群成员', value: metrics.member_count, hint: `${healthy}/${healthValues.length} 节点健康`, tone: healthy === healthValues.length ? 'success' as const : 'danger' as const },
    { key: 'db-size', label: '数据库大小', value: formatBytes(metrics.db_size), hint: `实际使用 ${formatBytes(metrics.db_size_in_use)}`, tone: 'default' as const },
    { key: 'fragmentation', label: '碎片率', value: `${fragmentation.percent}%`, hint: '数据库空间碎片', tone: fragmentation.tone },
    { key: 'version', label: 'etcd 版本', value: metrics.version, hint: `集群 ${metrics.cluster_id}`, tone: 'default' as const },
  ]
}
```

- [ ] **Step 4: Replace the cluster JSX with approved primitives**

Build the alert description before the return and use the approved primitives:

```tsx
const alarmList = <ul className="alarm-list">{alarms.map((alarm) => <li key={`${alarm.member_id}-${alarm.alarm_type}`}>成员 {alarm.member_id}：{alarm.alarm_type === 'NOSPACE' ? '磁盘空间不足' : alarm.alarm_type === 'CORRUPT' ? '数据损坏' : alarm.alarm_type}</li>)}</ul>

<PageHeader eyebrow="Cluster Overview" title="集群概览" description="实时掌握成员健康、存储使用与 Raft 同步状态" extra={<Button type="primary" icon={<ReloadOutlined />} onClick={fetchData} loading={loading}>刷新数据</Button>} />
{alarms.length > 0 && <Alert className="page-alert" type="error" showIcon message="集群报警" description={alarmList} />}
{metrics && <div className="metric-grid">{buildClusterMetricView(metrics).map((metric) => <MetricCard key={metric.key} {...metric} />)}</div>}
<SectionCard title="集群成员" description={`共 ${status?.members.length ?? 0} 个成员`}><Table className="data-table" rowKey="id" dataSource={status?.members ?? []} columns={memberColumns} pagination={false} size="middle" /></SectionCard>
```

Extract the current inline member column array to `const memberColumns` without changing its fields. Wrap detailed status and health tables in additional `SectionCard` blocks separated by `page-stack` gaps. Replace Ant Design status `Tag` instances with `StatusBadge`; preserve all table data and explanatory comments.

- [ ] **Step 5: Add shared page styles and verify**

Create `pages.css` with `.page-stack`, `.page-alert`, `.data-table`, `.page-pagination`, `.resource-split`, and `.service-groups`. Import it from `global.css`.

```css
.page-stack { display:flex; flex-direction:column; gap:20px; }.page-alert { margin-bottom:20px; }.alarm-list { margin:0; padding-left:20px; }.data-table .ant-table { background:transparent; }.data-table .ant-table-thead > tr > th { font-size:12px; font-weight:700; letter-spacing:.02em; }.data-table .ant-table-tbody > tr > td { height:44px; }.page-pagination { display:flex; justify-content:flex-end; margin-top:16px; }.metric-grid--three { grid-template-columns:repeat(3,minmax(0,1fr)); }.toolbar-search { width:300px; }.service-groups .ant-collapse-item { border-color:var(--border); }
```

Run:

```bash
cd web && npm test -- src/pages/cluster/presentation.test.ts && npm run typecheck && npm run build
```

Expected: presentation tests, typecheck, and build PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/cluster web/src/styles/pages.css web/src/global.css
git commit -m "feat(web): refresh cluster overview"
```

## Task 6: Migrate KV and configuration resource pages

**Files:**
- Create: `web/src/pages/kv/buildKVTree.test.ts`
- Modify: `web/src/pages/kv/index.tsx`
- Modify: `web/src/pages/kv/KVTreeView.tsx`
- Modify: `web/src/pages/config/index.tsx`
- Modify: `web/src/styles/pages.css`

- [ ] **Step 1: Add a regression test for tree construction**

Create `buildKVTree.test.ts`:

```ts
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
```

Run `cd web && npm test -- src/pages/kv/buildKVTree.test.ts` and confirm it passes before visual edits.

- [ ] **Step 2: Refactor KV page structure**

Preserve every handler and use:

```tsx
<PageHeader eyebrow="Key Value Store" title="KV 管理" description="浏览、检索和维护当前集群中的键值数据" extra={isAdmin ? <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建键值</Button> : undefined} />
<PageToolbar trailing={<Segmented value={viewMode} onChange={(value) => setViewMode(value as 'list' | 'tree')} options={[{ value: 'list', label: '列表', icon: <UnorderedListOutlined /> }, { value: 'tree', label: '树形', icon: <ApartmentOutlined /> }]} />}>
  <Input className="toolbar-search" prefix={<SearchOutlined />} value={prefix} onChange={(event) => setPrefix(event.target.value)} onPressEnter={() => fetchData()} />
  <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
</PageToolbar>
<SectionCard className="resource-card">{viewMode === 'list' ? <Table className="data-table" rowKey="key" columns={columns} dataSource={items} loading={loading} pagination={false} size="middle" /> : <KVTreeView treeData={buildKVTree(items)} isAdmin={isAdmin} onEdit={openEdit} onDelete={handleDelete} />}</SectionCard>
```

Change `KVTreeView` to a `.resource-split` CSS grid with a `.resource-tree` section and `.resource-detail` section. Use `CopyableCode` for the selected key and `EmptyState` for no selection. Keep Monaco and edit/delete callbacks unchanged.

- [ ] **Step 3: Refactor configuration page structure**

Use `PageHeader` for the title and primary “新建配置” action, `PageToolbar` for key filtering/refresh/import/export, `SectionCard` for the main table, `CopyableCode` for keys and truncated values, and `EmptyState` when no environment is selected. Keep history Drawer, import/export, editor, revision pagination, and rollback APIs unchanged.

Rollback confirmation must include the key and target revision time in `description`; delete confirmation must include the key and current environment.

- [ ] **Step 4: Add resource-page styles**

In `pages.css`, define a `380px minmax(0,1fr)` tree/detail grid, 500px minimum content height, monospaced value preview, toolbar search widths, and drawer pagination alignment. Remove replaced inline `display`, `gap`, `margin`, `fontFamily`, and color styles.

```css
.resource-split { min-height:500px; display:grid; grid-template-columns:380px minmax(0,1fr); }.resource-tree { overflow:auto; padding:16px; border-right:1px solid var(--border); background:var(--surface-subtle); }.resource-detail { min-width:0; padding:18px; }.resource-value-preview { overflow:hidden; font-family:var(--font-mono); text-overflow:ellipsis; white-space:nowrap; }.drawer-pagination { display:flex; justify-content:flex-end; margin-top:16px; }
```

- [ ] **Step 5: Verify tree behavior and build**

Run:

```bash
cd web && npm test -- src/pages/kv/buildKVTree.test.ts && npm run lint && npm run typecheck && npm run build
```

Expected: tree regression test, lint, typecheck, and build PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/kv web/src/pages/config/index.tsx web/src/styles/pages.css
git commit -m "feat(web): redesign resource management pages"
```

## Task 7: Migrate gateway and gRPC service monitoring

**Files:**
- Create: `web/src/pages/services/presentation.ts`
- Create: `web/src/pages/services/presentation.test.ts`
- Modify: `web/src/pages/gateway/index.tsx`
- Modify: `web/src/pages/grpc/index.tsx`
- Modify: `web/src/styles/pages.css`

- [ ] **Step 1: Write failing shared-summary tests**

Create `presentation.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { buildServiceSummary } from './presentation'

describe('buildServiceSummary', () => {
  it('summarizes degraded service groups', () => {
    expect(buildServiceSummary([{ instance_count: 3, healthy_count: 2 }, { instance_count: 2, healthy_count: 2 }])).toEqual({ services: 2, instances: 5, healthy: 4, healthDisplay: '80.0%', tone: 'warning' })
  })
  it('distinguishes fully healthy and empty services', () => {
    expect(buildServiceSummary([{ instance_count: 5, healthy_count: 5 }]).tone).toBe('success')
    expect(buildServiceSummary([])).toEqual({ services: 0, instances: 0, healthy: 0, healthDisplay: '0%', tone: 'default' })
  })
})
```

- [ ] **Step 2: Implement the pure shared helper**

```ts
interface ServiceCounts { instance_count: number; healthy_count: number }
type ServiceTone = 'default' | 'success' | 'warning'
interface ServiceSummary { services: number; instances: number; healthy: number; healthDisplay: string; tone: ServiceTone }
export function buildServiceSummary(groups: ServiceCounts[]): ServiceSummary {
  const instances = groups.reduce((sum, group) => sum + group.instance_count, 0)
  const healthy = groups.reduce((sum, group) => sum + group.healthy_count, 0)
  const percent = instances === 0 ? 0 : (healthy / instances) * 100
  const tone: ServiceTone = instances === 0 ? 'default' : healthy === instances ? 'success' : 'warning'
  return { services: groups.length, instances, healthy, healthDisplay: instances === 0 ? '0%' : `${percent.toFixed(1)}%`, tone }
}
```

- [ ] **Step 3: Apply the shared monitoring pattern to both pages**

Each page must render `PageHeader`, a three-card `metric-grid metric-grid--three`, and `SectionCard` around its Collapse. Replace duplicate statistic calculations with `buildServiceSummary(groups)`. Replace status Tags with `StatusBadge`, addresses and IDs with `CopyableCode`, and empty content with `EmptyState`.

Keep gateway `registered_at` formatting and gRPC `register_time` formatting distinct. Keep each API module, row key, details modal, and online/offline confirmation behavior unchanged.

- [ ] **Step 4: Verify shared metrics and both builds**

Run:

```bash
cd web && npm test -- src/pages/services/presentation.test.ts && npm run lint && npm run typecheck && npm run build
```

Expected: summary tests PASS and no duplicated TypeScript errors between gateway and gRPC.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/services web/src/pages/gateway web/src/pages/grpc web/src/styles/pages.css
git commit -m "feat(web): redesign service monitoring pages"
```

## Task 8: Migrate users, roles, and audit management

**Files:**
- Create: `web/src/pages/roles/permissions.ts`
- Create: `web/src/pages/roles/permissions.test.ts`
- Modify: `web/src/pages/users/index.tsx`
- Modify: `web/src/pages/roles/index.tsx`
- Modify: `web/src/pages/audit/index.tsx`
- Modify: `web/src/styles/pages.css`

- [ ] **Step 1: Write permission coupling tests**

Create `permissions.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { updatePermissionState } from './permissions'

describe('updatePermissionState', () => {
  it('enables read when write is enabled', () => {
    expect(updatePermissionState({}, 'kv', 'can_write').kv).toEqual({ can_read: true, can_write: true })
  })
  it('disables write when read is disabled', () => {
    const state = { kv: { can_read: true, can_write: true } }
    expect(updatePermissionState(state, 'kv', 'can_read').kv).toEqual({ can_read: false, can_write: false })
  })
  it('does not enable write when read is enabled', () => {
    expect(updatePermissionState({}, 'kv', 'can_read').kv).toEqual({ can_read: true, can_write: false })
  })
})
```

- [ ] **Step 2: Implement and adopt the pure permission helper**

```ts
export type PermissionState = Record<string, { can_read: boolean; can_write: boolean }>

export function updatePermissionState(state: PermissionState, module: string, field: 'can_read' | 'can_write'): PermissionState {
  const current = state[module] ?? { can_read: false, can_write: false }
  const updated = { ...current, [field]: !current[field] }
  if (field === 'can_write' && updated.can_write) updated.can_read = true
  if (field === 'can_read' && !updated.can_read) updated.can_write = false
  return { ...state, [module]: updated }
}
```

Replace the inline roles updater with `setPermissions((current) => updatePermissionState(current, module, field))`.

- [ ] **Step 3: Apply the management page pattern**

- Users: `PageHeader` with “新建用户”, `PageToolbar` with refresh and super-admin transfer, `SectionCard` around the table, `StatusBadge` for roles, and `.page-pagination` for pagination.
- Roles: `PageHeader` with “新建角色”, toolbar refresh, SectionCard table, and modal sections named “基本信息”, “授权环境”, and “模块权限”. Preserve role CRUD and user-count warning.
- Audit: `PageHeader`, `PageToolbar` containing action/resource/date filters and search/reset, `SectionCard` table, `CopyableCode` for resource keys and IPs, and semantic `StatusBadge` for actions.

No page may show write controls that existing permission logic hides. The users page remains accessible only through existing module permissions; super-admin-only behavior remains unchanged.

- [ ] **Step 4: Verify permissions and management pages**

Run:

```bash
cd web && npm test -- src/pages/roles/permissions.test.ts src/config/menu.test.ts && npm run lint && npm run typecheck && npm run build
```

Expected: permission coupling and menu permissions PASS; lint, typecheck, and build PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/users web/src/pages/roles web/src/pages/audit web/src/styles/pages.css
git commit -m "feat(web): redesign administration pages"
```

## Task 9: Complete async states, modal polish, and dark-theme consistency

**Files:**
- Modify: `web/src/layouts/components/EnvironmentManager.tsx`
- Modify: `web/src/layouts/components/PasswordModal.tsx`
- Modify: `web/src/layouts/components/SyncRestorePanel.tsx`
- Modify: all migrated page files where loading/empty/error states are still raw
- Modify: `web/src/styles/tokens.css`
- Modify: `web/src/styles/primitives.css`
- Modify: `web/src/styles/layout.css`
- Modify: `web/src/styles/pages.css`

- [ ] **Step 1: Inventory each page state before changing code**

Record one row for login, cluster, KV, config, gateway, gRPC, users, roles, and audit with four columns: initial loading, refresh loading, empty data, and initial error. The required result is:

```text
initial loading -> LoadingState or layout-matching Skeleton
refresh loading -> keep existing content and set button/table loading
empty data -> EmptyState with permission-aware action
initial error -> ErrorState with retry when the page stores an error value
```

- [ ] **Step 2: Normalize state behavior without changing API calls**

Cluster keeps its existing page-level error/retry behavior but renders `ErrorState`. Gateway and gRPC replace centered Spin/Empty with `LoadingState`/`EmptyState`. KV, config, users, roles, and audit add `const [error, setError] = useState<string | null>(null)`, clear it at the start of `fetchData`, and set it in the existing catch block. When `error` is set and no rows have ever loaded, render `ErrorState` with the current fetch function as `onRetry`; when rows exist, preserve the table and keep the existing message. These table pages keep Ant Design table loading and use `EmptyState` through `locale.emptyText`.

Use this exact fetch-state shape in KV:

```tsx
const [error, setError] = useState<string | null>(null)
const hasData = items.length > 0

const fetchData = async (prefixOverride?: string) => {
  setLoading(true)
  setError(null)
  try {
    const data = await kvApi.list(prefixOverride ?? prefix)
    setItems(data ?? [])
  } catch (caught: unknown) {
    const text = caught instanceof Error ? caught.message : '加载失败'
    setError(text)
    if (hasData) message.error(text)
  } finally {
    setLoading(false)
  }
}

if (error && !hasData) return <ErrorState description={error} onRetry={fetchData} />
```

Apply the same state transitions to the existing functions with these concrete empty checks and retries:

```tsx
// config/index.tsx
if (error && items.length === 0) return <ErrorState description={error} onRetry={fetchConfigs} />
// users/index.tsx
if (error && users.length === 0) return <ErrorState description={error} onRetry={() => fetchData(1)} />
// roles/index.tsx
if (error && roles.length === 0) return <ErrorState description={error} onRetry={() => fetchData(1)} />
// audit/index.tsx
if (error && logs.length === 0) return <ErrorState description={error} onRetry={() => fetchData(1)} />
```

In each corresponding fetch function, call `setError(null)` immediately after `setLoading(true)`, set `error` to the caught message, and call `message.error` only when that page's collection already contains rows.

- [ ] **Step 3: Normalize modal and danger-action presentation**

Use `destroyOnHidden`, vertical forms, consistent `okText`, and shared class names for every modal. Ensure delete, rollback, environment restore, instance offline, and super-admin transfer confirmations name the affected resource and lock the primary action while submitting.

- [ ] **Step 4: Remove remaining hard-coded presentation colors**

Run:

```bash
rg -n "#[0-9a-fA-F]{3,8}|fontFamily|backgroundColor" web/src --glob '*.tsx'
```

Expected before edits: matches in migrated files. Replace visual-only matches with token classes; Monaco language colors and data-derived values are exempt. Run the command again; expected matches are limited to theme configuration or documented Monaco behavior.

- [ ] **Step 5: Run the complete automated suite**

```bash
cd web && npm test && npm run lint && npm run typecheck && npm run build
```

Expected: all tests PASS; lint and typecheck emit no errors; Vite produces `web/dist` successfully.

- [ ] **Step 6: Commit**

```bash
git add web/src
git commit -m "feat(web): polish console states and themes"
```

## Task 10: Perform browser regression and visual acceptance

**Files:**
- Modify only files with defects found during this task.

- [ ] **Step 1: Start the application for browser checks**

Run the configured backend and then:

```bash
cd web && npm run dev -- --host 127.0.0.1
```

Expected: Vite reports a local URL and `/api` requests proxy to port 8080.

- [ ] **Step 2: Verify the 1440px light-theme path**

Using the default admin credentials in the README, check login, environment selection, every visible menu item, table scrolling, tree/detail layout, Monaco modals, service collapse panels, role permission modal, and audit filters. Expected: no clipping or horizontal overflow inside the 1440px viewport; primary actions remain in PageHeader.

- [ ] **Step 3: Verify the 1920px light-theme path**

Repeat the overview, KV, configuration, gateway, roles, and audit pages at 1920px. Expected: content expands to available width; tables are not constrained to a narrow centered column; metric grids remain balanced.

- [ ] **Step 4: Verify dark and system themes**

Switch light → dark → system. Expected: page surfaces, sidebar, tables, popovers, drawers, modals, status colors, and Monaco containers remain readable; reloading preserves the selected mode through the existing store.

- [ ] **Step 5: Verify RBAC and destructive actions**

Check a read-only role and a super administrator. Expected: hidden menus and disabled/absent write actions exactly match existing permission rules. Open, but do not confirm, delete/rollback/restore/transfer prompts and verify each prompt names its target and environment.

- [ ] **Step 6: Fix visual defects and rerun automated verification**

For each defect, make the smallest CSS or component change, then run:

```bash
cd web && npm test && npm run lint && npm run typecheck && npm run build
```

Expected: full suite PASS after the final visual correction.

- [ ] **Step 7: Commit final visual corrections**

If files changed:

```bash
git add web/src
git commit -m "fix(web): resolve visual regression issues"
```

If no files changed, do not create an empty commit.

## Completion checklist

- [ ] All ten tasks are completed in order.
- [ ] `npm test`, `npm run lint`, `npm run typecheck`, and `npm run build` pass from `web/`.
- [ ] Light, dark, and system themes are browser-verified at 1440px and 1920px.
- [ ] All existing routes, API calls, environment behavior, and RBAC rules are preserved.
- [ ] Login, global shell, and all eight business modules use the approved cloud-console visual language.
- [ ] Loading, refresh, empty, error, and dangerous-operation states meet the specification.
