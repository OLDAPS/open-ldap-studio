/**
 * Screen 3c — wizard step 2: the bind identity.
 *
 * Two things on this step are safety features rather than decoration:
 *   - the keychain line, because a user is entitled to know where a secret
 *     they type is about to be written;
 *   - the clear-text strip, because a simple bind over an unencrypted
 *     transport puts the password on the wire, and that fact belongs next to
 *     the field rather than in a manual. It is driven by the transport chosen
 *     on step 1, so it is quiet when there is nothing to warn about.
 *
 * "Retry anonymously" is deliberately absent from the failure actions: a
 * silent privilege downgrade makes the next screen a lie (deviation D7). A
 * rejected bind is reported with the server's own result code and left for the
 * user to fix.
 */
import type { ConnectionDraft } from './draft';
import type { BindMethod, TestResult } from '@/bridge/types';

const METHODS: [BindMethod, string][] = [
  ['simple', 'Simple'],
  ['digestMD5', 'SASL DIGEST-MD5'],
  ['gssapi', 'GSSAPI'],
  ['anonymous', 'Anonymous'],
];

export function AuthStep({
  draft,
  patch,
  onCheck,
  busy,
  result,
}: {
  draft: ConnectionDraft;
  patch: (changes: Partial<ConnectionDraft>) => void;
  onCheck: () => void;
  busy: boolean;
  result?: TestResult;
}) {
  const encrypted = draft.encryption !== 'none';
  const anonymous = draft.bindMethod === 'anonymous';

  return (
    <>
      <span className="card__label">Authentication</span>

      <div className="field-row field-row--wrap">
        <span className="field-row__label">Method</span>
        <span className="tag-set">
          {METHODS.map(([id, label]) => (
            <button
              key={id}
              type="button"
              className="tag"
              data-selected={draft.bindMethod === id || undefined}
              onClick={() => patch({ bindMethod: id })}
            >
              {label}
            </button>
          ))}
        </span>
      </div>

      {anonymous ? (
        <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
          The connection binds anonymously. Most directories return a much smaller tree in this
          state; that is the server&rsquo;s answer, not a failure to read it.
        </p>
      ) : (
        <>
          <div className="field-row">
            <label className="field-row__label" htmlFor="auth-dn">
              Bind DN
            </label>
            <input
              id="auth-dn"
              className="field field--mono"
              placeholder="cn=admin,dc=example,dc=org"
              autoComplete="off"
              value={draft.bindDn}
              onChange={(e) => patch({ bindDn: e.target.value })}
            />
          </div>

          <div className="field-row">
            <label className="field-row__label" htmlFor="auth-secret">
              Password
            </label>
            <input
              id="auth-secret"
              className="field"
              type="password"
              placeholder="••••••••"
              autoComplete="off"
              value={draft.secret}
              onChange={(e) => patch({ secret: e.target.value })}
            />
          </div>

          <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
            On Finish the secret goes to the platform credential store; the connection file keeps
            only a reference to it. Leave it empty to be prompted at each bind instead.
          </p>
        </>
      )}

      {!anonymous && !encrypted ? (
        <div className="strip strip--danger">
          <span className="strip__title">Clear text</span>
          <span>
            This credential would cross an unencrypted transport. Choose StartTLS or LDAPS on step
            1, or bind anonymously.
          </span>
        </div>
      ) : null}

      <div className="actions">
        <button
          type="button"
          className="button"
          onClick={onCheck}
          disabled={busy || !draft.host || (!anonymous && !draft.bindDn)}
        >
          {busy ? 'Checking…' : 'Check authentication'}
        </button>
        <BindOutcome result={result} busy={busy} />
      </div>
    </>
  );
}

/**
 * The bind's outcome, in the server's own words.
 *
 * A rejected bind shows the LDAP result code and the server's diagnostic
 * verbatim, with any plain-language reading added beside it — never instead of
 * it (FR-013).
 */
function BindOutcome({ result, busy }: { result?: TestResult; busy: boolean }) {
  if (busy || !result) return null;

  if (!result.reachable) {
    return (
      <span className="mono" style={{ color: 'var(--danger)' }}>
        could not reach the server
      </span>
    );
  }

  if (result.bound) {
    return (
      <span className="mono" style={{ color: 'var(--success)' }}>
        ✓ bound as {result.boundDn}
      </span>
    );
  }

  const code = result.result?.code;
  const diagnostic = result.result?.diagnosticMessage || result.message;
  return (
    <span className="mono" style={{ color: 'var(--danger)' }}>
      ✗ {code !== undefined ? `result ${code}` : 'rejected'}
      {diagnostic ? ` — ${diagnostic}` : ''}
      {result.result?.interpretation ? ` (${result.result.interpretation})` : ''}
    </span>
  );
}
