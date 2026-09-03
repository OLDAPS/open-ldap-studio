/**
 * The dry-run result panel (screen 1e, right).
 *
 * A projected outcome per record, and the counts that outcome implies. Each
 * line carries a glyph and a word, never a colour alone, because the three
 * states this panel reports are the difference between running and not.
 */
export function DryRunPanel() {
  const results: { outcome: 'ok' | 'warning' | 'error'; text: string }[] = [];

  const count = (outcome: string) => results.filter((r) => r.outcome === outcome).length;

  return (
    <aside className="inspector inspector--wide" aria-label="Dry-run result">
      <div className="inspector__header">
        <span>Dry-run result</span>
      </div>

      <div className="inspector__body">
        <div className="tag-set">
          <span className="tag">{count('ok')} would succeed</span>
          <span className="tag">{count('warning')} warning</span>
          <span className="tag">{count('error')} blocked</span>
        </div>

        <div className="card card--tight">
          {results.length === 0 ? (
            <span className="dim" style={{ fontSize: 'var(--text-caption)' }}>
              Nothing has been dry-run yet. A dry run reads the target and reports what each record
              would do; it writes nothing.
            </span>
          ) : (
            results.map((result) => (
              <div key={result.text} className="mono">
                {result.outcome === 'ok' ? '✓' : result.outcome === 'warning' ? '⚠' : '✗'}{' '}
                {result.text}
              </div>
            ))
          )}
        </div>

        <div className="placeholder" style={{ flex: 1, minHeight: 90 }}>
          before / after attribute diff for the
          <br />
          record selected in the editor
        </div>

        <div className="actions">
          <button type="button" className="button" disabled={results.length === 0}>
            Fix &amp; re-run
          </button>
          <button type="button" className="button" disabled={results.length === 0}>
            Save report
          </button>
        </div>
      </div>
    </aside>
  );
}
