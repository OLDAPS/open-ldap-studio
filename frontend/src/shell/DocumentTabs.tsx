import { Button } from '@/components/ui/button';
/**
 * Editors are documents, not modals.
 *
 * A search and an LDIF file are tabs you can leave open and come back to,
 * which is the difference between a tool you work in and a tool you visit
 * (screens.md § Shell).
 */
import { useSession } from '@/app/session';

const GLYPH: Record<string, string> = {
  entry: '◍',
  search: '⌕',
  ldif: '▤',
  schema: '◈',
  preferences: '⚙',
  connections: '⇄',
};

export function DocumentTabs() {
  const tabs = useSession((s) => s.tabs);
  const activeTabId = useSession((s) => s.activeTabId);
  const activateTab = useSession((s) => s.activateTab);
  const closeTab = useSession((s) => s.closeTab);

  if (tabs.length === 0) {
    return (
      <div className="tabstrip tabstrip--empty">
        <span className="dim">No documents open</span>
      </div>
    );
  }

  return (
    <div className="tabstrip" role="tablist">
      {tabs.map((tab) => (
        <div
          key={tab.id}
          role="tab"
          tabIndex={0}
          aria-selected={tab.id === activeTabId}
          className="tab"
          data-active={tab.id === activeTabId || undefined}
          onClick={() => activateTab(tab.id)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') activateTab(tab.id);
          }}
        >
          <span className="tab__glyph" aria-hidden="true">
            {GLYPH[tab.kind] ?? '◍'}
          </span>
          <span className="tab__title">{tab.title}</span>
          {tab.dirty ? (
            <span className="tab__dirty" title="Unsaved changes" aria-label="Unsaved changes">
              ●
            </span>
          ) : null}
          <Button
            variant="ghost"
            size="icon-sm"
            type="button"
            className="tab__close"
            aria-label={`Close ${tab.title}`}
            onClick={(e) => {
              e.stopPropagation();
              closeTab(tab.id);
            }}
          >
            ×
          </Button>
        </div>
      ))}
    </div>
  );
}
