import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
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
    <Tabs value={tab} onValueChange={(value) => setTab(value as Tab)} className="gap-0">
      <div className="toolbar">
        <span className="mono">
          {records === 0
            ? 'no change records'
            : `${records} change record${records === 1 ? '' : 's'}`}
        </span>
        <span className="toolbar__spacer" />
        <div className="actions">
          <Button type="button" variant="outline">
            Format
          </Button>
          <Button type="button" variant="outline">
            Validate
          </Button>
          <Button type="button" variant="outline">
            Dry run
          </Button>
          <Button type="button" variant="default" disabled={!target}>
            {target ? `Execute on ${target}` : 'Execute'}
          </Button>
        </div>
      </div>

      <TabsList className="pane-tabs" variant="line" aria-label="LDIF views">
        {TABS.map(([id, label]) => (
          <TabsTrigger key={id} value={id} className="pane-tab">
            {label}
          </TabsTrigger>
        ))}
      </TabsList>

      <div className="split">
        <div className="split__main" style={{ borderRight: '1px solid var(--border-subtle)' }}>
          <TabsContent value="source">
            <Textarea
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
          </TabsContent>

          <TabsContent value="diff">
            <div className="pane" style={{ flex: 1 }}>
              <div className="placeholder" style={{ height: 140 }}>
                before / after attribute diff for each record,
                <br />
                read from the target connection without writing to it
              </div>
            </div>
          </TabsContent>

          <TabsContent value="report">
            <div className="pane" style={{ flex: 1 }}>
              <div className="placeholder" style={{ height: 140 }}>
                the last run&rsquo;s report — counts per outcome, and the rejects
                <br />
                written to a sibling .ldif so the run can be repaired and repeated
              </div>
            </div>
          </TabsContent>

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
    </Tabs>
  );
}
