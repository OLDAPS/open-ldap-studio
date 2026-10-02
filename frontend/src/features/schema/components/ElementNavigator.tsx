import { Button } from '@/components/ui/button';
/**
 * Back / forward through visited schema elements (screen 1d).
 *
 * The hierarchy diagram makes it easy to wander three classes deep; without a
 * way back, the only route home is the sidebar search.
 */
export function ElementNavigator({ current }: { current?: string }) {
  return (
    <div className="actions">
      <Button type="button" variant="ghost" size="xs" aria-label="Back">
        ←
      </Button>
      <span className="mono">{current ?? '—'}</span>
      <Button type="button" variant="ghost" size="xs" aria-label="Forward">
        →
      </Button>
    </div>
  );
}
