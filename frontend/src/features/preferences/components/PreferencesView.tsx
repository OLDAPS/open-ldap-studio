import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
/**
 * Screen 3a — Preferences: a settings tree, a pane, and a live preview.
 *
 * Preferences open as a perspective rather than a modal window (deviation D4),
 * so a setting can be changed while the thing it affects is still on screen.
 */
import { useState } from 'react';

import { AppearancePane } from '@/features/preferences/components/AppearancePane';
import { BrowserPane } from '@/features/preferences/components/BrowserPane';
import { ConnectionsPane } from '@/features/preferences/components/ConnectionsPane';
import { EntryEditorPane } from '@/features/preferences/components/EntryEditorPane';
import { LdifPane } from '@/features/preferences/components/LdifPane';
import { SecurityPane } from '@/features/preferences/components/SecurityPane';
import { ShortcutsPane } from '@/features/preferences/components/ShortcutsPane';
import { UpdatesPane } from '@/features/preferences/components/UpdatesPane';
import { ValueEditorsPane } from '@/features/preferences/components/ValueEditorsPane';
import { Workspace } from '@/shell/Workspace';

type PaneId =
  | 'appearance'
  | 'browser'
  | 'entryEditor'
  | 'valueEditors'
  | 'ldif'
  | 'connections'
  | 'security'
  | 'shortcuts'
  | 'updates';

const PANES: { id: PaneId; label: string; screen: string }[] = [
  { id: 'appearance', label: 'Appearance', screen: '3a' },
  { id: 'browser', label: 'Browser & tree', screen: '5a' },
  { id: 'entryEditor', label: 'Entry editor', screen: '5b' },
  { id: 'valueEditors', label: 'Value editors', screen: '5c' },
  { id: 'ldif', label: 'LDIF & text editors', screen: '5d' },
  { id: 'connections', label: 'Connections & timeouts', screen: '5e' },
  { id: 'security', label: 'Credentials & security', screen: '5f' },
  { id: 'shortcuts', label: 'Keyboard shortcuts', screen: '5g' },
  { id: 'updates', label: 'Updates & about', screen: '5h' },
];

export function PreferencesView() {
  const [pane, setPane] = useState<PaneId>('appearance');
  const [filter, setFilter] = useState('');

  const visible = filter.trim()
    ? PANES.filter((p) => p.label.toLowerCase().includes(filter.toLowerCase()))
    : PANES;
  const current = PANES.find((p) => p.id === pane);

  return (
    <div className="view">
      <aside className="sidebar" aria-label="Preferences">
        <div className="sidebar__header">
          <span>Preferences</span>
          <span className="sidebar__actions mono">⌕</span>
        </div>

        <Input
          className="field sidebar__filter"
          type="search"
          value={filter}
          placeholder="search settings…"
          aria-label="Search settings"
          onChange={(e) => setFilter(e.target.value)}
        />

        <div className="sidebar__body">
          {visible.map((entry) => (
            <Button
              variant="ghost"
              size="xs"
              key={entry.id}
              type="button"
              className="tree-row"
              data-selected={entry.id === pane || undefined}
              onClick={() => setPane(entry.id)}
            >
              <span className="tree-row__twisty" aria-hidden="true" />
              <span className="tree-row__glyph" aria-hidden="true" />
              <span>{entry.label}</span>
            </Button>
          ))}
          {visible.length === 0 ? <p className="sidebar__note">no matching settings</p> : null}
        </div>
      </aside>

      <Workspace>
        <div className="toolbar">
          <span>Preferences › {current?.label}</span>
          <span className="toolbar__spacer" />
          <div className="actions">
            <Button type="button" variant="outline">
              Restore defaults
            </Button>
            <Button type="button" variant="outline">
              Export…
            </Button>
            <Button type="button" variant="default">
              Apply
            </Button>
          </div>
        </div>

        <div className="split">
          <div className="pane pane--scroll split__main">
            {pane === 'appearance' ? <AppearancePane /> : null}
            {pane === 'browser' ? <BrowserPane /> : null}
            {pane === 'entryEditor' ? <EntryEditorPane /> : null}
            {pane === 'valueEditors' ? <ValueEditorsPane /> : null}
            {pane === 'ldif' ? <LdifPane /> : null}
            {pane === 'connections' ? <ConnectionsPane /> : null}
            {pane === 'security' ? <SecurityPane /> : null}
            {pane === 'shortcuts' ? <ShortcutsPane /> : null}
            {pane === 'updates' ? <UpdatesPane /> : null}
          </div>

          <LivePreview />
        </div>
      </Workspace>
    </div>
  );
}

/**
 * The live preview (screen 3a, right). It renders with the tokens the pane is
 * changing, so theme, font and density are seen before Apply rather than after.
 */
function LivePreview() {
  return (
    <aside className="inspector" aria-label="Live preview">
      <div className="inspector__header">
        <span>Live preview</span>
      </div>
      <div className="inspector__body">
        <Card className="card card--tight">
          <div className="mono">dn: cn=jrivera,ou=people</div>
          <div className="mono dim">objectClass: inetOrgPerson</div>
          <div className="mono" style={{ color: 'var(--accent-quiet)' }}>
            mail: j.rivera@example.com
          </div>
        </Card>
        <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
          The preview reflects theme, fonts and density before Apply. Nothing on these panes changes
          directory data.
        </p>
      </div>
    </aside>
  );
}
