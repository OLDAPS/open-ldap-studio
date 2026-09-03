/**
 * Screen 1e — LDIF and files.
 *
 * Sidebar carries the workspace files, the outline of change records in the
 * open file, and the target connection with its dry-run switch. The target and
 * the dry-run state sit in the sidebar rather than in a dialog because they
 * are the two facts that decide what Execute will do, and they must be on
 * screen while the file is being edited, not remembered from a wizard.
 */
import { LdifEditor } from './ldif/LdifEditor';
import { Workspace } from '@/shell/Workspace';
import { useSession } from '@/store/session';

export function FilesView() {
  const connection = useSession((s) => s.activeConnection());

  return (
    <div className="view">
      <aside className="sidebar" aria-label="Workspace files">
        <div className="sidebar__header">
          <span>Workspace files</span>
          <span className="sidebar__actions mono">
            <button
              type="button"
              className="button button--quiet"
              title="New LDIF file"
              aria-label="New LDIF file"
            >
              +
            </button>
            <button type="button" className="button button--quiet" title="More" aria-label="More">
              ⋯
            </button>
          </span>
        </div>

        <div className="sidebar__body">
          <p className="sidebar__note">no files open</p>

          <div className="sidebar__section">Outline</div>
          <p className="sidebar__note">change records appear here</p>

          <div className="sidebar__section">Target</div>
          <div className="tree-row">
            <span className="tree-row__glyph tree-row__glyph--entry" aria-hidden="true" />
            <span>{connection?.serverIdentity ?? 'no connection'}</span>
          </div>
          <label className="tree-row" style={{ cursor: 'pointer' }}>
            <input type="checkbox" defaultChecked />
            <span className="dim">dry run</span>
          </label>
          <label className="tree-row" style={{ cursor: 'pointer' }}>
            <input type="checkbox" />
            <span className="dim">continue on error</span>
          </label>
        </div>
      </aside>

      <Workspace>
        <LdifEditor />
      </Workspace>
    </div>
  );
}
