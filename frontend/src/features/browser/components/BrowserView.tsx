/**
 * Screen 1b — DIT browser and entry editor: the primary workspace.
 *
 * Sidebar is the tree (filter box, lazy children, an explicit "fetch next"
 * node, and Searches / Bookmarks as sibling roots under the connection); the
 * document area is the entry editor with its own tab set and the entry-info
 * inspector on the right.
 */
import { DitTree } from './DitTree';
import { EntryView } from './EntryView';
import { Workspace } from '@/shell/Workspace';
import { useSession } from '@/app/session';

export function BrowserView() {
  const connection = useSession((s) => s.activeConnection());
  const label = connection?.serverIdentity ?? 'no connection';

  return (
    <div className="view">
      <aside className="sidebar" aria-label="Directory tree">
        <div className="sidebar__header">
          <span>DIT · {label}</span>
          <span className="sidebar__actions mono">
            <button type="button" className="button button--quiet" title="Find" aria-label="Find">
              ⌕
            </button>
            <button
              type="button"
              className="button button--quiet"
              title="Refresh"
              aria-label="Refresh"
            >
              ⟳
            </button>
            <button type="button" className="button button--quiet" title="More" aria-label="More">
              ⋯
            </button>
          </span>
        </div>
        <DitTree />
      </aside>

      <Workspace>
        <EntryView />
      </Workspace>
    </div>
  );
}
