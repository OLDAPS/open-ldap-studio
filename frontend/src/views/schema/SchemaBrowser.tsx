/**
 * The schema element view (screen 1d, document area).
 *
 * Definition on the left, raw form and usage on the right. The raw definition
 * is shown verbatim rather than reformatted: it is what the server said, and
 * it is what a bug report needs to quote.
 *
 * "Used by N entries · run as search" is the one place the schema browser
 * reaches into the rest of the application — it hands a SearchDefinition to
 * the search editor and changes nothing on the server.
 */
import { useState } from 'react';

import { HierarchyDiagram } from './HierarchyDiagram';
import { UsedBySearch } from './UsedBySearch';

interface SchemaElement {
  name: string;
  kind: string;
  oid: string;
  origin: string;
}

export function SchemaBrowser() {
  const [element] = useState<SchemaElement | undefined>();

  if (!element) {
    return (
      <div className="pane" style={{ flex: 1 }}>
        <p className="view-title">Schema</p>
        <p className="dim" style={{ margin: 0, maxWidth: '68ch' }}>
          Open a connection and pick an element on the left. The schema shown is the one this server
          publishes, not a built-in copy — which is the only version that can tell you whether a
          change will be accepted.
        </p>
      </div>
    );
  }

  return (
    <div className="pane" style={{ flex: 1, minHeight: 0 }}>
      <div className="view-title-row">
        <p className="view-title">{element.name}</p>
        <span className="tag">{element.kind}</span>
        <span className="tag mono">{element.oid}</span>
        <span className="dim">{element.origin}</span>
      </div>

      <div className="split" style={{ gap: 'var(--space-4)' }}>
        <div className="card" style={{ flex: 1, minWidth: 0 }}>
          <span className="card__label">Definition</span>
          <div className="field-row">
            <span className="field-row__label">Superior</span>
            <span className="mono">—</span>
          </div>
          <div className="field-row">
            <span className="field-row__label">Must contain</span>
            <span className="mono">—</span>
          </div>
          <div className="field-row" style={{ alignItems: 'flex-start' }}>
            <span className="field-row__label">May contain</span>
            <span className="mono">—</span>
          </div>
          <HierarchyDiagram />
        </div>

        <div className="inspector inspector--wide" style={{ border: 0, background: 'none' }}>
          <div className="inspector__body" style={{ padding: 0 }}>
            <div className="card card--tight">
              <span className="card__label">Raw definition</span>
              <code className="mono dim">—</code>
            </div>
            <div className="card card--tight">
              <span className="card__label">Used by</span>
              <UsedBySearch />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
