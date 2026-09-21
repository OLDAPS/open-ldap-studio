/**
 * Screen 1c — search and filter builder with result grid.
 *
 * The search editor is a document tab rather than a modal, which is the whole
 * point of the screen: a filter you can leave open, re-run, save and hand to
 * somebody else is a different tool from a dialog you dismiss.
 */
import { AttributePalette } from './AttributePalette';
import { SavedSearches } from './SavedSearches';
import { SearchEditor } from './SearchEditor';
import { Workspace } from '@/shell/Workspace';

export function SearchesView() {
  return (
    <div className="view">
      <aside className="sidebar" aria-label="Searches">
        <div className="sidebar__header">
          <span>Searches</span>
          <span className="sidebar__actions mono">
            <button
              type="button"
              className="button button--quiet"
              title="New search"
              aria-label="New search"
            >
              +
            </button>
            <button type="button" className="button button--quiet" title="More" aria-label="More">
              ⋯
            </button>
          </span>
        </div>

        <div className="sidebar__body">
          <SavedSearches />
          <AttributePalette />
        </div>
      </aside>

      <Workspace>
        <SearchEditor />
      </Workspace>
    </div>
  );
}
