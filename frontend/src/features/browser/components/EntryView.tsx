import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
/**
 * The entry editor (screen 1b, document area).
 *
 * Four tabs over one entry — Attributes, LDIF view, Table editor, Object class
 * — a toolbar carrying the DN, and the entry-info inspector on the right.
 *
 * Save is a button, not a consequence of losing focus (deviation D3). It stays
 * disabled here because the write path is preview-then-commit and nothing in
 * this editor has an edit to preview yet; a button that looked ready and did
 * nothing would be worse than one that admits it.
 */
import { useState } from 'react';

import { AttributeTable } from './AttributeTable';
import { EntryInfoPanel } from './EntryInfoPanel';
import { useEntryViewModel } from '../useEntryViewModel';
import { displayValue } from '../value';
import type { Attribute } from '@/bridge/types';
import { useSession } from '@/app/session';

type Tab = 'attributes' | 'ldif' | 'table' | 'objectClass';

const TABS: [Tab, string][] = [
  ['attributes', 'Attributes'],
  ['ldif', 'LDIF view'],
  ['table', 'Table editor'],
  ['objectClass', 'Object class'],
];

export function EntryView() {
  const [tab, setTab] = useState<Tab>('attributes');
  const [showOperational, setShowOperational] = useState(false);

  const activeTabId = useSession((s) => s.activeTabId);
  const document = useSession((s) => s.tabs.find((t) => t.id === activeTabId));
  const profileId = useSession((s) => s.activeProfileId);

  const dn = document?.dn;
  const { entry, loading, error, reload } = useEntryViewModel(
    document?.profileId ?? profileId,
    dn,
    showOperational,
  );

  return (
    <>
      <div className="toolbar">
        <span className="mono" title={dn}>
          {dn ? `dn: ${dn}` : 'no entry selected'}
        </span>
        <span className="toolbar__spacer" />
        <div className="actions">
          <Button type="button" variant="outline" disabled={!entry}>
            New attribute
          </Button>
          <Button type="button" variant="outline" disabled={!dn || loading} onClick={reload}>
            {loading ? 'Reading…' : 'Refresh'}
          </Button>
          <Button
            type="button"
            variant="default"
            disabled
            title="Editing reaches the server through preview-then-commit, which is not wired yet"
          >
            Save (⌘S)
          </Button>
        </div>
      </div>

      <div className="split">
        <Tabs
          className="split__main gap-0"
          value={tab}
          onValueChange={(value) => setTab(value as Tab)}
        >
          <TabsList className="pane-tabs" variant="line" aria-label="Entry views">
            {TABS.map(([id, label]) => (
              <TabsTrigger key={id} value={id} className="pane-tab">
                {label}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="attributes">
            <AttributeTable
              entry={entry}
              loading={loading}
              error={error}
              showOperational={showOperational}
              onShowOperational={setShowOperational}
            />
          </TabsContent>

          <TabsContent value="ldif">
            <LdifTab dn={dn} attributes={entry?.attributes} />
          </TabsContent>

          <TabsContent value="table">
            <div className="pane" style={{ flex: 1 }}>
              <div className="placeholder" style={{ height: 120 }}>
                table editor — one row per child entry, columns chosen from the
                <br />
                parent&rsquo;s object classes
              </div>
            </div>
          </TabsContent>

          <TabsContent value="objectClass">
            <ObjectClassTab attributes={entry?.attributes} />
          </TabsContent>
        </Tabs>

        <EntryInfoPanel entry={entry} loading={loading} />
      </div>
    </>
  );
}

/**
 * The entry as LDIF.
 *
 * A value that is not printable text is written base64 with `::`, which is the
 * LDIF rule and also the only honest way to show bytes as text.
 */
function LdifTab({ dn, attributes }: { dn?: string; attributes?: Attribute[] }) {
  if (!dn) {
    return (
      <div className="pane" style={{ flex: 1 }}>
        <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
          Select an entry to see it as LDIF.
        </p>
      </div>
    );
  }

  const lines: string[] = [`dn: ${dn}`];
  for (const attribute of attributes ?? []) {
    const name = attribute.options?.length
      ? `${attribute.type};${attribute.options.join(';')}`
      : attribute.type;
    for (const raw of attribute.values ?? []) {
      const value = displayValue(raw);
      lines.push(value.binary ? `${name}:: ${raw}` : `${name}: ${value.text}`);
    }
  }

  return (
    <div className="source">
      {lines.map((line, index) => (
        <div className="source__line" key={index}>
          <span className="source__gutter">{index + 1}</span>
          <span style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{line}</span>
        </div>
      ))}
    </div>
  );
}

/**
 * The entry's object classes.
 *
 * Which of them is structural is a schema question, and the schema browser is
 * not wired yet — so the list is shown in the order the server returned it and
 * makes no claim it cannot support.
 */
function ObjectClassTab({ attributes }: { attributes?: Attribute[] }) {
  const objectClass = attributes?.find((a) => a.type.toLowerCase() === 'objectclass');
  const classes = (objectClass?.values ?? []).map((v) => displayValue(v).text);

  return (
    <div className="pane" style={{ flex: 1 }}>
      <Card className="card card--tight">
        <span className="card__label">Object classes</span>
        {classes.length === 0 ? (
          <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
            No entry loaded.
          </p>
        ) : (
          <div className="tag-set">
            {classes.map((name) => (
              <span key={name} className="tag">
                {name}
              </span>
            ))}
          </div>
        )}
      </Card>

      <div className="placeholder" style={{ height: 110 }}>
        adding or removing an auxiliary class recomputes the must / may set
        <br />
        against the server schema before commit
      </div>
    </div>
  );
}
