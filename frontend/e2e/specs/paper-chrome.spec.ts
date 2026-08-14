import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { describe, it } from 'node:test'
import { fileURLToPath } from 'node:url'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const src = (...parts: string[]) => path.join(frontendDir, 'src', ...parts)

function readSrc(...parts: string[]): string {
  return fs.readFileSync(src(...parts), 'utf8')
}

function cssVar(css: string, name: string, scope = ':root'): string {
  const blockRe = new RegExp(`${scope.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*\\{([\\s\\S]*?)\\n\\}`)
  const block = blockRe.exec(css)?.[1]
  assert.ok(block, `missing CSS scope ${scope}`)
  const m = new RegExp(`--${name}:\\s*([^;]+);`).exec(block)
  assert.ok(m, `missing --${name} in ${scope}`)
  return m[1].trim()
}

function remToPx(value: string): number {
  const rem = /^([\d.]+)rem$/.exec(value)
  if (rem) return Number(rem[1]) * 16
  const px = /^([\d.]+)px$/.exec(value)
  assert.ok(px, `expected rem or px, got ${value}`)
  return Number(px[1])
}

describe('solarized paper chrome', () => {
  const css = readSrc('styles', 'main.css')
  const topbar = readSrc('components', 'layout', 'Topbar.vue')
  const rail = readSrc('components', 'canvas', 'FlowRail.vue')
  const inspector = readSrc('components', 'canvas', 'CanvasInspector.vue')
  const pathField = readSrc('components', 'forms', 'RemotePathField.vue')
  const settings = readSrc('components', 'flows', 'OperationSettingsPanel.vue')

  it('keeps Solarized paper/base hues and 6–8px radius without offset-neo shadows', () => {
    assert.equal(cssVar(css, 'color-bg'), '#fdf6e3')
    assert.equal(cssVar(css, 'color-bg-secondary'), '#eee8d5')
    assert.equal(cssVar(css, 'color-bg', 'html.dark'), '#002b36')
    assert.equal(cssVar(css, 'color-bg-secondary', 'html.dark'), '#073642')

    const radius = remToPx(cssVar(css, 'radius-md'))
    assert.ok(radius >= 6 && radius <= 8, `radius-md ${radius}px not in 6–8`)

    const paper = cssVar(css, 'shadow-paper')
    const neo = cssVar(css, 'shadow-neo')
    assert.match(paper, /^0\s+\d+px/)
    assert.doesNotMatch(paper, /\d+px\s+\d+px\s+0\b/)
    assert.equal(neo, 'var(--shadow-paper)')
    assert.match(css, /\.neo-card[\s\S]*?border border-border/)
    assert.doesNotMatch(css, /\.btn-primary[\s\S]*?border-2/)
    assert.doesNotMatch(css, /\.field-input[\s\S]*?border-2/)
  })

  it('uses typographic topbar and flow-card headers, not mustard fill bands', () => {
    const header = /<header[\s\S]*?>/.exec(topbar)?.[0] ?? ''
    assert.match(header, /bg-surface/)
    assert.doesNotMatch(header, /bg-accent/)

    assert.match(rail, /bg-surface/)
    assert.doesNotMatch(rail, /bg-accent-strong/)
  })

  it('keeps path + Browse on one row in the inspector', () => {
    const row = /<div class="(flex min-w-0 items-center gap-1\.5)">/.exec(pathField)?.[1] ?? ''
    assert.equal(row, 'flex min-w-0 items-center gap-1.5')
    assert.doesNotMatch(pathField, /flex-wrap items-center gap-1\.5/)
    assert.match(pathField, /shrink-0 whitespace-nowrap/)
    assert.match(inspector, /op-src-\$\{selectedEdge\.id\}/)
    assert.match(inspector, /op-dst-\$\{selectedEdge\.id\}/)
  })

  it('collapses operation settings groups', () => {
    assert.match(settings, /data-testid="op-settings-panel"/)
    const groups = [
      'workspace.opSettings.performance',
      'workspace.opSettings.filtering',
      'workspace.opSettings.safety',
      'workspace.opSettings.comparison',
      'workspace.opSettings.syncOptions',
      'workspace.opSettings.bisyncOptions',
    ]
    for (const key of groups) {
      const idx = settings.indexOf(key)
      assert.ok(idx > 0, `missing settings group ${key}`)
      const before = settings.slice(Math.max(0, idx - 400), idx)
      assert.match(before, /<details/, `${key} should live in a <details>`)
      assert.doesNotMatch(before, /<details[^>]*\sopen[\s>]/, `${key} must start collapsed`)
    }
  })

  it('shows edge run affordances and locks inspector fields while running', () => {
    const page = readSrc('pages', 'WorkspacePage.vue')
    const edge = readSrc('components', 'canvas', 'SyncEdge.vue')
    const inspector = readSrc('components', 'canvas', 'CanvasInspector.vue')
    assert.doesNotMatch(page, /FlowRunBottomBar/)
    assert.match(edge, /data-testid="`canvas-edge-errors-\$\{id\}`"/)
    assert.match(edge, /data-testid="`canvas-edge-processed-\$\{id\}`"/)
    assert.match(edge, /<marker/)
    assert.match(edge, /marker-end="`url\(#\$\{markerId\}\)`"/)
    assert.match(edge, /marker-start="isBidirectional \? `url\(#\$\{markerId\}\)` : undefined"/)
    assert.match(edge, /left: `\$\{sourceX\}px`, top: `\$\{sourceY \+ 28\}px`/)
    assert.match(edge, /left: `\$\{targetX\}px`, top: `\$\{targetY \+ 28\}px`/)
    assert.match(edge, /data-testid="`canvas-edge-files-\$\{id\}`"/)
    assert.match(edge, /class="max-h-52 overflow-auto p-1" @wheel\.stop/)
    assert.match(inspector, /:disabled="running"/)
    assert.match(inspector, /:disabled="!dirty \|\| running"/)
  })

  it('keeps e2e data-testid hooks used by the product specs', () => {
    const specsDir = path.join(frontendDir, 'e2e/specs')
    const specs = fs
      .readdirSync(specsDir)
      .filter((f) => f.endsWith('.spec.ts'))
      .map((f) => fs.readFileSync(path.join(specsDir, f), 'utf8'))
      .join('\n')
    const hooks = [
      'page-unlock',
      'unlock-password',
      'unlock-submit',
      'unlock-error',
      'page-workspace',
      'workspace-remotes',
      'workspace-flows',
      'remotes-add',
      'remotes-add-form',
      'remotes-name',
      'remotes-type',
      'remotes-submit',
      'flows-add',
      'flows-name-inline',
      'flows-run',
      'nav-settings',
      'page-settings',
      'nav-workspace',
      'theme-light',
      'theme-dark',
      'settings-old-password',
      'settings-new-password',
      'settings-change-password',
      'settings-msg',
      'lock-button',
    ]
    const tree = [
      readSrc('pages', 'UnlockPage.vue'),
      readSrc('pages', 'WorkspacePage.vue'),
      readSrc('pages', 'SettingsPage.vue'),
      readSrc('components', 'layout', 'Topbar.vue'),
      readSrc('components', 'workspace', 'WorkspaceRemotesSection.vue'),
      inspector,
      rail,
    ].join('\n')
    for (const hook of hooks) {
      assert.match(specs, new RegExp(hook.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), `spec uses ${hook}`)
      assert.match(tree, new RegExp(hook.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), hook)
    }
    assert.match(inspector, /op-src-\$\{selectedEdge\.id\}/)
    assert.match(inspector, /op-dst-\$\{selectedEdge\.id\}/)
    assert.match(inspector, /flows-save-bottom-\$\{activeFlow\.id\}/)
    assert.match(rail, /flows-delete-\$\{f\.id\}/)
    assert.match(readSrc('components', 'workspace', 'WorkspaceRemotesSection.vue'), /remote-chip-\$\{r\.name\}/)
  })
})
