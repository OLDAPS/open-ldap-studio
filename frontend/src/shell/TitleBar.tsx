/**
 * The window's own title bar: traffic lights, the menu bar, and a context
 * strip showing what the window is currently pointed at.
 *
 * The window is frameless (main.go), so this bar is also the drag handle.
 */
import { MenuBar } from './MenuBar';
import { windowControls } from '@/bridge/client';
import { useSession } from '@/app/session';

export function TitleBar() {
  const connection = useSession((s) => s.activeConnection());
  const tabs = useSession((s) => s.tabs);
  const activeTabId = useSession((s) => s.activeTabId);
  const activeTab = tabs.find((t) => t.id === activeTabId);

  const context = activeTab?.dn ?? connection?.serverIdentity ?? 'no connection';

  return (
    <div className="titlebar" style={{ '--wails-draggable': 'drag' } as React.CSSProperties}>
      <div
        className="titlebar__dots"
        style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
      >
        <button
          type="button"
          className="titlebar__dot titlebar__dot--close"
          aria-label="Close window"
          onClick={() => windowControls.quit()}
        />
        <button
          type="button"
          className="titlebar__dot titlebar__dot--minimise"
          aria-label="Minimise window"
          onClick={() => windowControls.minimise()}
        />
        <button
          type="button"
          className="titlebar__dot titlebar__dot--maximise"
          aria-label="Maximise window"
          onClick={() => windowControls.toggleMaximise()}
        />
      </div>

      <div style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}>
        <MenuBar />
      </div>

      <div className="titlebar__context mono" title={context}>
        {context}
      </div>
    </div>
  );
}
