import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
/**
 * Screen 1d — schema browser, read-only.
 *
 * Sidebar navigates the four element kinds the subschema subentry publishes;
 * the document area shows one element's definition with its raw form and its
 * usage beside it. Editing schema happens offline in a project (screen 6a),
 * never against this view.
 */
import { SchemaBrowser } from './SchemaBrowser';
import { Workspace } from '@/shell/Workspace';
import { useSession } from '@/app/session';

export function SchemaView() {
  const connection = useSession((s) => s.activeConnection());

  return (
    <div className="view">
      <aside className="sidebar" aria-label="Schema">
        <div className="sidebar__header">
          <span>Schema · {connection?.serverIdentity ?? 'no connection'}</span>
          <span className="sidebar__actions mono">
            <Button type="button" variant="ghost" size="xs" title="Find" aria-label="Find">
              ⌕
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              title="Reload schema"
              aria-label="Reload schema"
            >
              ⟳
            </Button>
          </span>
        </div>

        <Input
          className="field sidebar__filter"
          type="search"
          placeholder="search object class / attribute…"
          aria-label="Search the schema"
        />

        <div className="sidebar__body">
          <div className="sidebar__section">Object classes</div>
          <p className="sidebar__note">read from cn=subschema on connect</p>

          <div className="sidebar__section">Attribute types</div>
          <p className="sidebar__note">—</p>

          <div className="sidebar__section">Matching rules · syntaxes</div>
          <p className="sidebar__note">—</p>
        </div>
      </aside>

      <Workspace>
        <SchemaBrowser />
      </Workspace>
    </div>
  );
}
