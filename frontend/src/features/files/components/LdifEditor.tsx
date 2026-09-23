/**
 * The LDIF editor (screen 1e, document area).
 *
 * LDIF is treated as source: line numbers, a validation gutter, and a dry run
 * against the named target that reports a projected outcome per record before
 * anything is executed. The Execute button names the connection it will write
 * to — "Execute" alone is how the wrong window gets written to.
 */
import { useState } from 'react';

import { DryRunPanel } from './DryRunPanel';
import { useSession } from '@/app/session';

type Tab = 'source' | 'diff' | 'report';

const TABS: [Tab, string][] = [
  ['source', 'Source'],
  ['diff', 'Diff vs. server'],
  ['report', 'Import report'],
];

export function LdifEditor() {
  const [tab, setTab] = useState<Tab>('source');
  const [text, setText] = useState('');
  const connection = useSession((s) => s.activeConnection());
  const target = connection?.serverIdentity;

  const lines = text ? text.split('\n') : [];
  const records = lines.filter((line) => line.startsWith('dn:')).length;

  return (
    <>
      <div className="toolbar">
        <span className="mono">
          {records === 0
            ? 'no change records'
            : `${records} change record${records === 1 ? '' : 's'}`}
        </span>
        <span className="toolbar__spacer" />
        <div className="actions">
          <button type="button" className="button">
            Format
          </button>
          <button type="button" className="button">
            Validate
          </button>
          <button type="button" className="button">
            Dry run
          </button>
          <button type="button" className="button button--primary" disabled={!target}>
            {target ? `Execute on ${target}` : 'Execute'}
          </button>
        </div>
      </div>

      <div className="pane-tabs" role="tablist" aria-label="LDIF views">
        {TABS.map(([id, label]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={tab === id}
            className="pane-tab"
            data-active={tab === id || undefined}
            onClick={() => setTab(id)}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="split">
        <div className="split__main" style={{ borderRight: '1px solid var(--border-subtle)' }}>
          {tab === 'source' ? (
            <textarea
              className="source"
              value={text}
              spellCheck={false}
              aria-label="LDIF source"
              placeholder={'dn: cn=example,dc=example,dc=com\nchangetype: add\nobjectClass: top'}
              style={{
                resize: 'none',
                border: 0,
                background: 'transparent',
                padding: 'var(--space-3)',
                borderRadius: 0,
              }}
              onChange={(event) => setText(event.target.value)}
            />
          ) : null}

          {tab === 'diff' ? (
            <div className="pane" style={{ flex: 1 }}>
              <div className="placeholder" style={{ height: 140 }}>
                before / after attribute diff for each record,
                <br />
                read from the target connection without writing to it
              </div>
            </div>
          ) : null}

          {tab === 'report' ? (
            <div className="pane" style={{ flex: 1 }}>
              <div className="placeholder" style={{ height: 140 }}>
                the last run&rsquo;s report — counts per outcome, and the rejects
                <br />
                written to a sibling .ldif so the run can be repaired and repeated
              </div>
            </div>
          ) : null}

          <div className="pane" style={{ flex: 'none' }}>
            <div className="placeholder" style={{ height: 56 }}>
              gutter markers: syntax errors, unknown attributes,
              <br />
              schema violations — with quick-fix suggestions
            </div>
          </div>
        </div>

        <DryRunPanel />
      </div>
    </>
  );
}
