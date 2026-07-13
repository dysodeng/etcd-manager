# Frontend Quality and Performance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add repeatable frontend lint/tests and split route/Monaco code so the production entry chunk is no longer a single approximately 1.42 MB file.

**Architecture:** ESLint 9 and Vitest establish a minimal quality gate around existing TypeScript modules. React Router elements become lazy route boundaries, Monaco is dynamically imported inside its wrapper, and Vite uses stable vendor groups for cacheable output.

**Tech Stack:** React 18, TypeScript 5.9, Vite 8, ESLint 9 flat config, typescript-eslint, React Hooks lint rules, Vitest.

---

## File Structure

- Modify `web/package.json`, `web/package-lock.json`: add lint/typecheck/test scripts and dev dependencies.
- Create `web/eslint.config.js`: browser/TypeScript/React Hooks flat lint configuration.
- Create `web/src/config/menu.test.ts`: initial pure module test suite.
- Modify `web/src/App.tsx`: lazy-load layout and every page with one Suspense boundary.
- Modify `web/src/components/MonacoEditor.tsx`: runtime dynamic import of `@monaco-editor/react` while retaining existing Props.
- Modify `web/vite.config.ts`: Vitest configuration and stable React/Ant Design/Monaco vendor groups.

Token storage files (`web/src/stores/auth.ts`, `web/src/api/client.ts`) are explicitly not modified.

### Task 1: Install and Configure Frontend Quality Gates

**Files:**
- Modify: `web/package.json`
- Modify: `web/package-lock.json`
- Create: `web/eslint.config.js`
- Create: `web/src/config/menu.test.ts`
- Modify: `web/vite.config.ts`

- [ ] **Step 1: Verify the missing-script baseline**

Run from `web/`:

~~~bash
npm run lint
npm test
~~~

Expected: both commands fail because the scripts do not exist.

- [ ] **Step 2: Install exact tool categories**

Run from `web/`:

~~~bash
npm install --save-dev eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh globals vitest
~~~

Expected: `package.json` and `package-lock.json` include the new dev dependencies; installation exits 0.

- [ ] **Step 3: Add package scripts**

Set `scripts` in `web/package.json` to include:

~~~json
{
  "dev": "vite",
  "build": "tsc -b && vite build",
  "lint": "eslint .",
  "typecheck": "tsc --noEmit --incremental false",
  "test": "vitest run",
  "test:watch": "vitest",
  "preview": "vite preview"
}
~~~

- [ ] **Step 4: Create ESLint flat config**

Create `web/eslint.config.js`:

~~~js
import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'coverage', '*.tsbuildinfo'] },
  {
    files: ['**/*.{ts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser, ...globals.node },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactRefresh.configs.vite.rules,
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
)
~~~

- [ ] **Step 5: Write the initial menu behavior tests**

Create `web/src/config/menu.test.ts`:

~~~ts
import { describe, expect, it } from 'vitest'
import type { UserProfile } from '@/types'
import { getDefaultRoute, getVisibleMenuKeys } from './menu'

const roleUser: UserProfile = {
  user_id: 'user-1',
  username: 'reader',
  is_super: false,
  role: {
    id: 'role-1',
    name: 'reader',
    permissions: [{ module: 'config', can_read: true, can_write: false }],
    environment_ids: [],
  },
}

describe('menu permissions', () => {
  it('shows only modules granted to a role user', () => {
    expect(getVisibleMenuKeys(roleUser)).toEqual(['/config'])
    expect(getDefaultRoute(roleUser)).toBe('/config')
  })

  it('shows every menu to a super admin', () => {
    const superUser: UserProfile = { ...roleUser, is_super: true, role: null }
    expect(getVisibleMenuKeys(superUser)).toHaveLength(8)
  })

  it('uses cluster as the empty-permission fallback', () => {
    expect(getDefaultRoute(null)).toBe('/cluster')
  })
})
~~~

- [ ] **Step 6: Add Vitest configuration to Vite**

Import `defineConfig` from `vitest/config` instead of `vite`, keep the existing React plugin/alias/proxy, and add:

~~~ts
test: {
  environment: 'node',
  include: ['src/**/*.test.ts'],
},
~~~

- [ ] **Step 7: Run the new gates and resolve expected lint findings**

Run from `web/`:

~~~bash
npm run lint
npm run typecheck
npm test
~~~

Expected: all commands exit 0. Existing dependency omissions may be reported as `react-hooks/exhaustive-deps` warnings; rules-of-hooks violations and TypeScript/ESLint correctness errors must remain fatal.

- [ ] **Step 8: Commit quality baseline**

~~~bash
git add web/package.json web/package-lock.json web/eslint.config.js web/src/config/menu.test.ts web/vite.config.ts
git commit -m "test: add frontend quality gates"
~~~

### Task 2: Lazy-Load Routes

**Files:**
- Modify: `web/src/App.tsx`

- [ ] **Step 1: Record the pre-change build output**

Run from `web/`:

~~~bash
rm -rf dist
npm run build
ls -lh dist/assets/*.js
~~~

Expected: build succeeds and reports a single main JS asset around 1.42 MB.

- [ ] **Step 2: Replace eager imports with lazy route boundaries**

At the top of `App.tsx` use:

~~~tsx
import { lazy, Suspense, type ReactNode } from 'react'
import { Spin } from 'antd'

const MainLayout = lazy(() => import('@/layouts/MainLayout'))
const LoginPage = lazy(() => import('@/pages/login'))
const KVPage = lazy(() => import('@/pages/kv'))
const ConfigPage = lazy(() => import('@/pages/config'))
const ClusterPage = lazy(() => import('@/pages/cluster'))
const UsersPage = lazy(() => import('@/pages/users'))
const RolesPage = lazy(() => import('@/pages/roles'))
const AuditPage = lazy(() => import('@/pages/audit'))
const GatewayPage = lazy(() => import('@/pages/gateway'))
const GrpcPage = lazy(() => import('@/pages/grpc'))
~~~

Change `ProtectedRoute` to use `ReactNode`, and wrap the complete `<Routes>` element in:

~~~tsx
<Suspense fallback={<Spin fullscreen tip="加载中..." />}>
  <Routes>
    <Route path="/login" element={<LoginPage />} />
    <Route path="/" element={<ProtectedRoute><MainLayout /></ProtectedRoute>}>
      <Route index element={<DefaultRedirect />} />
      <Route path="kv" element={<KVPage />} />
      <Route path="config" element={<ConfigPage />} />
      <Route path="cluster" element={<ClusterPage />} />
      <Route path="users" element={<UsersPage />} />
      <Route path="roles" element={<RolesPage />} />
      <Route path="audit" element={<AuditPage />} />
      <Route path="gateway" element={<GatewayPage />} />
      <Route path="grpc" element={<GrpcPage />} />
    </Route>
  </Routes>
</Suspense>
~~~

- [ ] **Step 3: Verify lazy routes**

Run from `web/`:

~~~bash
npm run lint
npm run typecheck
npm test
npm run build
~~~

Expected: all commands exit 0 and Vite emits named route chunks in addition to the entry asset.

- [ ] **Step 4: Commit route split**

~~~bash
git add web/src/App.tsx
git commit -m "perf: lazy load frontend routes"
~~~

### Task 3: Delay Monaco and Stabilize Vendor Chunks

**Files:**
- Modify: `web/src/components/MonacoEditor.tsx`
- Modify: `web/vite.config.ts`

- [ ] **Step 1: Make the Monaco runtime import lazy**

Replace the eager runtime import with a type-only import plus React lazy component:

~~~tsx
import { lazy, Suspense, useEffect, useRef, useState } from 'react'
import { Spin } from 'antd'
import type { EditorProps, Monaco } from '@monaco-editor/react'

const Editor = lazy(() => import('@monaco-editor/react'))
~~~

Wrap both editor instances with Suspense without changing existing values/options/mount behavior:

~~~tsx
<Suspense fallback={<Spin style={{ display: 'block', margin: '48px auto' }} />}>
  <Editor
    height={height}
    language={lang}
    value={value}
    onChange={(nextValue) => onChange?.(nextValue ?? '')}
    options={options}
    theme="vs-dark"
    onMount={handleMount}
  />
</Suspense>
~~~

Apply the same wrapper to the expanded editor.

- [ ] **Step 2: Add stable vendor groups**

Under `build` in `web/vite.config.ts` add:

~~~ts
rollupOptions: {
  output: {
    manualChunks: {
      react: ['react', 'react-dom', 'react-router-dom', 'zustand'],
      antd: ['antd', '@ant-design/icons', 'dayjs'],
      monaco: ['monaco-editor', '@monaco-editor/react'],
    },
  },
},
~~~

- [ ] **Step 3: Verify Monaco is not in the entry chunk**

Run from `web/`:

~~~bash
rm -rf dist
npm run lint
npm run typecheck
npm test
npm run build
ls -lh dist/assets/*.js
~~~

Expected: commands exit 0; output includes separate route chunks and a Monaco vendor chunk. The main entry asset is no longer approximately 1.42 MB. A size warning for the isolated Monaco or Ant Design vendor chunk is acceptable because neither is the all-code entry bundle.

- [ ] **Step 4: Confirm Token files are untouched**

Run:

~~~bash
git diff --quiet main...HEAD -- web/src/stores/auth.ts web/src/api/client.ts
~~~

Expected: exit 0.

- [ ] **Step 5: Commit Monaco/vendor split**

~~~bash
git add web/src/components/MonacoEditor.tsx web/vite.config.ts
git commit -m "perf: defer Monaco and split vendor bundles"
~~~

### Task 4: Frontend and Deployment Verification

**Files:**
- Verify all files changed by Tasks 1-3.

- [ ] **Step 1: Run frontend gates from a clean build directory**

Run from `web/`:

~~~bash
rm -rf dist
npm ci
npm run lint
npm run typecheck
npm test
npm run build
~~~

Expected: all commands exit 0.

- [ ] **Step 2: Run repository integration checks**

Run from the repository root:

~~~bash
go test -count=1 ./...
helm lint deploy/helm
git diff --check
git status --short --branch
~~~

Expected: Go tests pass, Helm reports 0 failed charts, diff check is clean, and only expected branch commits are present.
