import { Card } from '@/components/ui/card';
import { Alert } from '@/components/ui/alert';
/**
 * The import report (screen 1e, "Import report" tab; flow 7e terminal state).
 *
 * A run reports counts per outcome and names every record it skipped. There is
 * no partial-silent state: if the report cannot name a DN, the run is not
 * finished being reported.
 */
export function ImportReport({
  added = 0,
  modified = 0,
  deleted = 0,
  rejected = 0,
}: {
  added?: number;
  modified?: number;
  deleted?: number;
  rejected?: number;
}) {
  return (
    <div className="pane" style={{ flex: 1 }}>
      <Card className="card card--tight">
        <span className="card__label">Result</span>
        <div className="tag-set">
          <span className="tag">{added} added</span>
          <span className="tag">{modified} modified</span>
          <span className="tag">{deleted} deleted</span>
          <span className="tag" data-selected={rejected > 0 || undefined}>
            {rejected} rejected
          </span>
        </div>
      </Card>

      {rejected > 0 ? (
        <Alert className="strip strip--warning">
          <span className="strip__title">Rejects saved</span>
          <span>
            The failed records were written to a sibling .ldif with their result codes, so the run
            can be repaired and repeated.
          </span>
        </Alert>
      ) : null}
    </div>
  );
}
