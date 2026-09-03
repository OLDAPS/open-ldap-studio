/**
 * Screen 5g — Keyboard shortcuts.
 *
 * FR-102 requires full keyboard operability, which implies a command registry;
 * the rebinding table is that registry made visible. Conflicts are shown as
 * they are recorded, because a shortcut bound twice fails silently at the
 * moment it is needed.
 */
import { useState } from 'react';

import { useCommands } from '@/commands/CommandProvider';

const COLS = 'minmax(0, 1fr) 130px 130px 96px';

export function ShortcutsPane() {
  const [filter, setFilter] = useState('');
  const { commands, chordFor } = useCommands();

  const rows = commands.filter((command) =>
    filter.trim() ? command.label.toLowerCase().includes(filter.toLowerCase()) : true,
  );

  return (
    <>
      <div className="field-row field-row--wrap">
        <span className="field-row__label">Keymap preset</span>
        <span className="tag-set">
          <button type="button" className="tag" data-selected>
            Default
          </button>
          <button type="button" className="tag">
            Custom
          </button>
        </span>
        <span className="actions actions--end">
          <input
            className="field"
            type="search"
            value={filter}
            placeholder="search command…"
            aria-label="Search commands"
            onChange={(e) => setFilter(e.target.value)}
            style={{ width: 190 }}
          />
          <button type="button" className="button">
            Export
          </button>
        </span>
      </div>

      <div className="card">
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
      </div>

      <div className="placeholder" style={{ height: 52 }}>
        recording field: press the new combination —
        <br />
        conflicts and reserved OS shortcuts are flagged as you type
      </div>
    </>
  );
}
