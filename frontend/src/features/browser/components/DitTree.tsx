/**
 * The DIT tree (screen 1b, sidebar).
 *
 * Children are fetched a page at a time and the remainder is an explicit row
 * the user clicks — never an infinite scroll that quietly issues searches
 * against a production directory. The row says how many have been loaded and,
 * where the server admitted to one, the limit it enforced.
 *
 * Rows are virtualised because a container with 100 000 children is ordinary;
 * every row is therefore the same height, which is what --row-height is for.
 */
import { useMemo, useRef, useState } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';

import { rdnOf } from '../dn';
import { TreeFilterBox } from './TreeNodes';
import { useDitTreeViewModel } from '../useDitTreeViewModel';
import type { Row } from '../useDitTreeViewModel';
import { useSession } from '@/app/session';

export function DitTree() {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [filter, setFilter] = useState('');

  const connection = useSession((s) => s.activeConnection());
  const profileId = useSession((s) => s.activeProfileId);
  const openTab = useSession((s) => s.openTab);

  const tree = useDitTreeViewModel(profileId, connection?.state === 'connected');

  // The filter narrows what has been fetched. It does not issue a search, so
  // it can never time out — and the placeholder says as much.
  const rows = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    if (!needle) return tree.rows;
    return tree.rows.filter(
      (row) => row.kind === 'more' || row.label.toLowerCase().includes(needle),
    );
  }, [tree.rows, filter]);

  const virtual = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 24,
    overscan: 16,
  });

  if (connection?.state !== 'connected') {
    return (
      <>
        <TreeFilterBox value={filter} onChange={setFilter} />
        <div className="sidebar__body">
          <p className="sidebar__note">
            {connection ? 'not connected' : 'no connection selected'}
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      <TreeFilterBox value={filter} onChange={setFilter} />

      <div className="sidebar__body" ref={scrollRef}>
        {tree.loadingRoots ? <p className="sidebar__note">reading naming contexts…</p> : null}

        {tree.rootError ? (
          <div className="strip strip--danger" style={{ margin: 'var(--space-2) var(--space-3)' }}>
            <span className="mono">{tree.rootError}</span>
          </div>
        ) : null}

        <div style={{ height: virtual.getTotalSize(), position: 'relative' }}>
          {virtual.getVirtualItems().map((item) => {
            const row = rows[item.index];
            if (!row) return null;

            const style: React.CSSProperties = {
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: item.size,
              transform: `translateY(${item.start}px)`,
              paddingLeft: `calc(var(--space-3) + ${row.depth} * var(--space-4))`,
            };

            if (row.kind === 'more') {
              return (
                <button
                  key={item.key}
                  type="button"
                  className="tree-row tree-row--more"
                  style={style}
                  disabled={row.loading}
                  onClick={() => tree.fetchMore(row.parentDn)}
                  title={
                    row.truncated
                      ? 'The server stopped early on a limit it enforced'
                      : 'Fetch the next page of children'
                  }
                >
                  {row.loading
                    ? 'fetching…'
                    : row.truncated
                      ? `▾ ${row.loaded.toLocaleString()} loaded · server stopped at its limit${
                          row.serverLimit ? ` (${row.serverLimit})` : ''
                        }`
                      : `▾ fetch more · ${row.loaded.toLocaleString()} loaded`}
                </button>
              );
            }

            const isLeaf = row.hasChildren === 'no';

            return (
              <button
                key={item.key}
                type="button"
                className="tree-row"
                data-selected={tree.selected === row.dn || undefined}
                style={style}
                title={row.error ?? row.dn}
                onClick={() => {
                  tree.select(row.dn);
                  if (profileId) {
                    openTab({
                      id: row.dn,
                      title: rdnOf(row.dn),
                      kind: 'entry',
                      dn: row.dn,
                      profileId,
                    });
                  }
                }}
                onDoubleClick={() => {
                  if (!isLeaf) tree.toggle(row.dn);
                }}
              >
                <span
                  className="tree-row__twisty"
                  aria-hidden="true"
                  onClick={(event) => {
                    if (isLeaf) return;
                    event.stopPropagation();
                    tree.toggle(row.dn);
                  }}
                >
                  {row.loading ? '·' : isLeaf ? '' : row.expanded ? '▾' : '▸'}
                </span>
                <span
                  className={
                    isLeaf ? 'tree-row__glyph tree-row__glyph--entry' : 'tree-row__glyph'
                  }
                  aria-hidden="true"
                />
                <span>{row.label}</span>
                {row.error ? (
                  <span className="tree-row__count" style={{ color: 'var(--danger)' }}>
                    failed
                  </span>
                ) : null}
              </button>
            );
          })}
        </div>

        {!tree.loadingRoots && !tree.rootError && rows.length === 0 ? (
          <p className="sidebar__note">
            {filter ? 'nothing loaded matches that filter' : 'no naming contexts'}
          </p>
        ) : null}
      </div>
    </>
  );
}

export type { Row };
