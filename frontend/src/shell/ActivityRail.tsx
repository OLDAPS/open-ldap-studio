/**
 * The activity rail: six perspectives.
 *
 * Six, not seven. The LDAP Servers perspective the wireframe drew is dropped
 * with local server management (deviation D1).
 */
import type { Perspective } from '@/app/session';
import { useSession } from '@/app/session';

interface RailEntry {
  id: Perspective;
  glyph: string;
  label: string;
}

const TOP: RailEntry[] = [
  { id: 'connections', glyph: '⇄', label: 'Connections' },
  { id: 'browser', glyph: '⊞', label: 'DIT browser' },
  { id: 'searches', glyph: '⌕', label: 'Searches' },
  { id: 'schema', glyph: '◈', label: 'Schema' },
  { id: 'files', glyph: '▤', label: 'LDIF & files' },
];

const BOTTOM: RailEntry[] = [{ id: 'preferences', glyph: '⚙', label: 'Preferences' }];

export function ActivityRail() {
  const perspective = useSession((s) => s.perspective);
  const setPerspective = useSession((s) => s.setPerspective);

  const button = (entry: RailEntry) => (
    <button
      key={entry.id}
      type="button"
      className="rail__icon"
      data-active={perspective === entry.id || undefined}
      title={entry.label}
      aria-label={entry.label}
      aria-current={perspective === entry.id ? 'page' : undefined}
      onClick={() => setPerspective(entry.id)}
    >
      <span aria-hidden="true">{entry.glyph}</span>
    </button>
  );

  return (
    <nav className="rail" aria-label="Perspectives">
      {TOP.map(button)}
      <div className="rail__spacer" />
      {BOTTOM.map(button)}
    </nav>
  );
}
