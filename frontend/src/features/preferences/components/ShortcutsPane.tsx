import { Field } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
/**
 * Screen 5g — Keyboard shortcuts.
 *
 * FR-102 requires full keyboard operability, which implies a command registry;
 * the rebinding table is that registry made visible. Conflicts are shown as
 * they are recorded, because a shortcut bound twice fails silently at the
 * moment it is needed.
 */
import { useState } from 'react';

import { useCommands } from '@/app/CommandProvider';

const COLS = 'minmax(0, 1fr) 130px 130px 96px';

export function ShortcutsPane() {
  const [filter, setFilter] = useState('');
  const { commands, chordFor } = useCommands();

  const rows = commands.filter((command) =>
    filter.trim() ? command.label.toLowerCase().includes(filter.toLowerCase()) : true,
  );

  return (
    <>
      <Field orientation="horizontal" className="field-row field-row--wrap">
        <span className="field-row__label">Keymap preset</span>
        <span className="tag-set">
          <Button variant="outline" size="xs" type="button" className="tag" data-selected>
            Default
          </Button>
          <Button variant="outline" size="xs" type="button" className="tag">
            Custom
          </Button>
        </span>
        <span className="actions actions--end">
          <Input
            className="field"
            type="search"
            value={filter}
            placeholder="search command…"
            aria-label="Search commands"
            onChange={(e) => setFilter(e.target.value)}
            style={{ width: 190 }}
          />
          <Button type="button" variant="outline">
            Export
          </Button>
        </span>
      </Field>

      <Card className="card">
        <div className="dgrid">
          <div className="dgrid__head" style={{ '--cols': COLS } as React.CSSProperties}>
            <div>Command</div>
            <div>Binding</div>
            <div>Scope</div>
            <div>Source</div>
          </div>
          {rows.map((command) => (
            <div
              key={command.id}
              className="dgrid__row"
              style={{ '--cols': COLS } as React.CSSProperties}
            >
              <div>{command.label}</div>
              <div className="mono">{chordFor(command.id) ?? '—'}</div>
              <div className="dim">{command.scope}</div>
              <div className="dim">default</div>
            </div>
          ))}
          {rows.length === 0 ? (
            <p
              className="dim"
              style={{ padding: 'var(--space-4)', fontSize: 'var(--text-caption)' }}
            >
              No command matches that search.
            </p>
          ) : null}
        </div>
      </Card>

      <div className="placeholder" style={{ height: 52 }}>
        recording field: press the new combination —
        <br />
        conflicts and reserved OS shortcuts are flagged as you type
      </div>
    </>
  );
}
