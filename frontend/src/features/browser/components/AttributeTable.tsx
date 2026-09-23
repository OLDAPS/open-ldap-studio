/**
 * The attribute table (screen 1b, Attributes tab).
 *
 * Three columns, as drawn: attribute, value, and the kind — because "may I
 * delete this row" is the first question anyone asks of it.
 *
 * Two rules the rendering follows:
 *   - A value that is not printable text is described, not printed. A JPEG
 *     shown as text is a screenful of noise hiding the one useful fact.
 *   - Multi-valued attributes fold after three values, with the count of what
 *     is hidden. Nothing is silently dropped (FR-027).
 */
import { useState } from 'react';

import { displayValue } from '../value';
import type { Attribute, Entry } from '@/bridge/types';

const COLS = '220px minmax(0, 1fr) 90px';
const FOLD_AFTER = 3;

export function AttributeTable({
  entry,
  loading,
  error,
  showOperational,
  onShowOperational,
}: {
  entry?: Entry;
  loading: boolean;
  error?: string;
  showOperational: boolean;
  onShowOperational: (next: boolean) => void;
}) {
  const [selected, setSelected] = useState<string>();
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const attributes = entry?.attributes ?? [];
  const hiddenOperational = showOperational
    ? 0
    : attributes.filter((a) => a.isOperational).length;
  const visible = showOperational ? attributes : attributes.filter((a) => !a.isOperational);

  return (
    <>
      <div className="toolbar">
        <label className="field-group" style={{ flex: 'none' }}>
          <input
            type="checkbox"
            checked={showOperational}
            onChange={(e) => onShowOperational(e.target.checked)}
          />
          <span>Operational attributes</span>
        </label>
        <span className="toolbar__spacer" />
        <span className="dim mono">
          {loading ? 'reading…' : `${visible.length} attribute${visible.length === 1 ? '' : 's'}`}
        </span>
      </div>

      <div className="dgrid dgrid--flush" style={{ flex: 1 }}>
        <div className="dgrid__head" style={{ '--cols': COLS } as React.CSSProperties}>
          <div>Attribute</div>
          <div>Value</div>
          <div>Type</div>
        </div>

        {visible.map((attribute) => (
          <AttributeRow
            key={describe(attribute)}
            attribute={attribute}
            selected={selected === describe(attribute)}
            expanded={expanded.has(describe(attribute))}
            onSelect={() => setSelected(describe(attribute))}
            onExpand={() =>
              setExpanded((current) => {
                const next = new Set(current);
                const key = describe(attribute);
                if (next.has(key)) next.delete(key);
                else next.add(key);
                return next;
              })
            }
          />
        ))}

        {/* An absence is stated rather than left to be inferred. */}
        {hiddenOperational > 0 ? (
          <div className="dgrid__row" data-muted style={{ '--cols': COLS } as React.CSSProperties}>
            <div className="mono">operational</div>
            <div className="mono">
              {hiddenOperational} attribute{hiddenOperational === 1 ? '' : 's'} hidden
            </div>
            <div>op</div>
          </div>
        ) : null}

        {error ? (
          <div className="strip strip--danger" style={{ margin: 'var(--space-3)' }}>
            <span className="strip__title">Could not read the entry</span>
            <span className="mono">{error}</span>
          </div>
        ) : null}

        {!loading && !error && !entry ? (
          <p className="dim" style={{ padding: 'var(--space-4)', fontSize: 'var(--text-caption)' }}>
            No entry is loaded. Select one in the tree, or open a search result.
          </p>
        ) : null}
      </div>
    </>
  );
}

function describe(a: Attribute): string {
  return a.options?.length ? `${a.type};${a.options.join(';')}` : a.type;
}

function AttributeRow({
  attribute,
  selected,
  expanded,
  onSelect,
  onExpand,
}: {
  attribute: Attribute;
  selected: boolean;
  expanded: boolean;
  onSelect: () => void;
  onExpand: () => void;
}) {
  const values = attribute.values ?? [];
  const shown = expanded ? values : values.slice(0, FOLD_AFTER);
  const hidden = values.length - shown.length;

  return (
    <div
      className="dgrid__row"
      role="button"
      tabIndex={0}
      data-selected={selected || undefined}
      style={{ '--cols': COLS } as React.CSSProperties}
      onClick={onSelect}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onSelect();
      }}
    >
      <div className="mono" title={describe(attribute)}>
        {describe(attribute)}
      </div>
      <div className="mono" style={{ whiteSpace: 'normal' }}>
        {shown.map((raw, index) => {
          const value = displayValue(raw);
          return (
            <div key={index} className={value.binary ? 'dim' : undefined}>
              {value.text || <span className="dim">(empty)</span>}
            </div>
          );
        })}
        {hidden > 0 ? (
          <button
            type="button"
            className="button button--quiet"
            style={{ padding: 0, color: 'var(--text-link)' }}
            onClick={(e) => {
              e.stopPropagation();
              onExpand();
            }}
          >
            +{hidden} more
          </button>
        ) : null}
        {expanded && values.length > FOLD_AFTER ? (
          <button
            type="button"
            className="button button--quiet"
            style={{ padding: 0, color: 'var(--text-link)' }}
            onClick={(e) => {
              e.stopPropagation();
              onExpand();
            }}
          >
            show fewer
          </button>
        ) : null}
      </div>
      <div className="dim">{attribute.isOperational ? 'op' : '—'}</div>
    </div>
  );
}
