/**
 * Screen 1a — the four-step connection wizard: Network → Authentication →
 * Browser options → Edit options.
 *
 * Two promises this screen makes and now keeps:
 *
 *   - "Check network parameter" and "Check authentication" perform a real
 *     connection and a real bind against the server, and persist nothing.
 *   - Nothing is written until Finish, which saves the profile and then files
 *     the secret in the platform credential store under a reference the
 *     profile points at. The password never travels inside the profile
 *     payload; the Go decoder refuses one outright.
 *
 * The wireframe's "Provider" selector (JNDI / Apache Directory API) is absent:
 * it is a Java artefact with no meaning in a Go client (deviation D5).
 */
import { useState } from 'react';

import { AuthStep } from './AuthStep';
import { DEFAULT_PORT, draftProblems, emptyDraft, toProfile } from '../model';
import type { ConnectionDraft } from '../model';
import { bridge, isEmbedded } from '@/bridge/client';
import type { Encryption, TestResult } from '@/bridge/types';

const STEPS = ['Network', 'Authentication', 'Browser options', 'Edit options'] as const;

export function ConnectionWizard({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved?: (profileId: string) => void;
}) {
  const [step, setStep] = useState(0);
  const [draft, setDraft] = useState<ConnectionDraft>(emptyDraft);
  const [portEdited, setPortEdited] = useState(false);

  const [test, setTest] = useState<TestResult | undefined>();
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | undefined>();

  const patch = (changes: Partial<ConnectionDraft>) => {
    setDraft((current) => ({ ...current, ...changes }));
    // Any edit invalidates the last check: a green tick that describes a
    // hostname the user has since changed is worse than no tick at all.
    setTest(undefined);
  };

  // Changing the transport moves the port with it, until the user names one of
  // their own — after that it is theirs and we stop touching it.
  const chooseEncryption = (encryption: Encryption) => {
    patch(portEdited ? { encryption } : { encryption, port: DEFAULT_PORT[encryption] });
  };

  const runTest = async (withBind: boolean) => {
    setError(undefined);
    setTesting(true);
    setTest(undefined);
    try {
      const profile = toProfile(draft);
      const result = await bridge.testConnection(
        withBind ? profile : { ...profile, bindMethod: 'anonymous', bindDn: '' },
        withBind ? draft.secret : '',
      );
      setTest(result);
      // The server just told us its naming contexts; that is the base DN the
      // next page is about to ask for, so offer it rather than asking twice.
      const firstContext = result.namingContexts?.[0];
      if (!draft.baseDn && firstContext) {
        setDraft((current) => ({ ...current, baseDn: firstContext }));
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setTesting(false);
    }
  };

  const finish = async () => {
    setError(undefined);
    setSaving(true);
    try {
      const saved = await bridge.saveProfile(toProfile(draft));
      if (draft.bindMethod !== 'anonymous' && draft.secret) {
        await bridge.storeProfileSecret(saved.id, draft.secret);
      }
      onSaved?.(saved.id);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setSaving(false);
    }
  };

  const problems = draftProblems(draft);
  const canFinish = problems.length === 0 && !saving && isEmbedded();

  return (
    <>
      <div className="toolbar">
        <span style={{ color: 'var(--text-primary)' }}>New LDAP connection</span>
        <span className="dim">File › New › LDAP Connection</span>
        <span className="toolbar__spacer" />
        <span className="dim mono">nothing is persisted until Finish</span>
      </div>

      <div className="pane pane--scroll" style={{ flex: 1, minHeight: 0 }}>
        <div className="wizard">
          <nav className="wizard__steps" aria-label="Wizard steps">
            {STEPS.map((label, index) => (
              <button
                key={label}
                type="button"
                className="wizard__step"
                data-active={index === step || undefined}
                data-done={index < step || undefined}
                onClick={() => setStep(index)}
              >
                {index + 1} · {label}
              </button>
            ))}
            <TestSummary result={test} busy={testing} />
          </nav>

          <div className="card wizard__body">
            {step === 0 ? (
              <NetworkStep
                draft={draft}
                patch={patch}
                onEncryptionChange={chooseEncryption}
                onPortChange={(port) => {
                  setPortEdited(true);
                  patch({ port });
                }}
                onCheck={() => void runTest(false)}
                busy={testing}
              />
            ) : null}

            {step === 1 ? (
              <AuthStep
                draft={draft}
                patch={patch}
                onCheck={() => void runTest(true)}
                busy={testing}
                result={test}
              />
            ) : null}

            {step === 2 ? <BrowserOptionsStep draft={draft} patch={patch} /> : null}
            {step === 3 ? <EditOptionsStep draft={draft} patch={patch} /> : null}

            {error ? (
              <div className="strip strip--danger">
                <span className="strip__title">Failed</span>
                <span className="mono">{error}</span>
              </div>
            ) : null}

            {!isEmbedded() ? (
              <div className="strip strip--warning">
                <span className="strip__title">No backend</span>
                <span>
                  This interface is running outside the desktop shell, so nothing can be tested or
                  saved. Start it with <span className="mono">wails dev</span>.
                </span>
              </div>
            ) : null}

            <div className="actions actions--end" style={{ marginTop: 'auto' }}>
              {problems.length > 0 ? (
                <span className="dim" style={{ marginRight: 'auto' }}>
                  Still needs {problems.join(', ')}.
                </span>
              ) : null}
              <button type="button" className="button" onClick={onClose} disabled={saving}>
                Cancel
              </button>
              <button
                type="button"
                className="button"
                disabled={step === 0 || saving}
                onClick={() => setStep((s) => Math.max(0, s - 1))}
              >
                Back
              </button>
              <button
                type="button"
                className="button"
                disabled={step === STEPS.length - 1 || saving}
                onClick={() => setStep((s) => Math.min(STEPS.length - 1, s + 1))}
              >
                Next
              </button>
              <button
                type="button"
                className="button button--primary"
                disabled={!canFinish}
                onClick={() => void finish()}
              >
                {saving ? 'Saving…' : 'Finish'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

/**
 * What the server said about itself, in the panel the wireframe reserved for a
 * root DSE summary. Reachability and the bind are reported separately, because
 * "the host answered but rejected me" and "the host never answered" are
 * different problems with different fixes.
 */
function TestSummary({ result, busy }: { result?: TestResult; busy: boolean }) {
  if (busy) {
    return (
      <div className="placeholder" style={{ marginTop: 'auto', height: 70 }}>
        checking…
      </div>
    );
  }

  if (!result) {
    return (
      <div className="placeholder" style={{ marginTop: 'auto', height: 70 }}>
        root DSE summary appears here
        <br />
        once the server answers
      </div>
    );
  }

  if (!result.reachable) {
    return (
      <div className="strip strip--danger" style={{ marginTop: 'auto' }}>
        <span className="strip__title">Unreachable</span>
        <span className="mono">{result.message}</span>
      </div>
    );
  }

  return (
    <div className="card card--tight" style={{ marginTop: 'auto' }}>
      <span className="card__label">Server</span>
      <div className="mono" style={{ color: 'var(--success)' }}>
        ✓ reachable{result.encrypted ? ' · encrypted' : ''}
        {result.encrypted && !result.tlsVerified ? ' (unverified)' : ''}
      </div>
      {result.vendorName ? (
        <div className="mono dim">
          {result.vendorName} {result.vendorVersion}
        </div>
      ) : null}
      {result.namingContexts?.length ? (
        <div className="mono dim">{result.namingContexts.join(', ')}</div>
      ) : null}
      {result.bound ? (
        <div className="mono" style={{ color: 'var(--success)' }}>
          ✓ bound as {result.boundDn}
        </div>
      ) : null}
    </div>
  );
}

interface StepProps {
  draft: ConnectionDraft;
  patch: (changes: Partial<ConnectionDraft>) => void;
}

function NetworkStep({
  draft,
  patch,
  onEncryptionChange,
  onPortChange,
  onCheck,
  busy,
}: StepProps & {
  onEncryptionChange: (next: Encryption) => void;
  onPortChange: (next: string) => void;
  onCheck: () => void;
  busy: boolean;
}) {
  return (
    <>
      <span className="card__label">Network</span>

      <div className="field-row">
        <label className="field-row__label" htmlFor="conn-name">
          Connection name
        </label>
        <input
          id="conn-name"
          className="field"
          placeholder={draft.host || 'local-dev'}
          value={draft.name}
          onChange={(e) => patch({ name: e.target.value })}
        />
      </div>

      <div className="field-row">
        <label className="field-row__label" htmlFor="conn-host">
          Hostname
        </label>
        <input
          id="conn-host"
          className="field"
          placeholder="localhost"
          value={draft.host}
          onChange={(e) => patch({ host: e.target.value })}
        />
        <label className="field-row__label field-row__label--auto" htmlFor="conn-port">
          Port
        </label>
        <input
          id="conn-port"
          className="field"
          /*
           * Typed, not nudged. A number input carries spinner buttons and
           * increments on the mouse wheel, and neither is a sensible way to
           * reach a port: 636 is not 389 stepped 247 times, and a stray scroll
           * over a focused field silently retargets the connection.
           */
          type="text"
          inputMode="numeric"
          autoComplete="off"
          maxLength={5}
          value={draft.port}
          onChange={(e) => onPortChange(e.target.value.replace(/\D/g, ''))}
          style={{ width: 80, flex: 'none' }}
        />
      </div>

      <div className="field-row">
        <label className="field-row__label" htmlFor="conn-encryption">
          Encryption
        </label>
        <select
          id="conn-encryption"
          className="field"
          value={draft.encryption}
          onChange={(e) => onEncryptionChange(e.target.value as Encryption)}
        >
          <option value="none">No encryption</option>
          <option value="startTLS">StartTLS</option>
          <option value="ldaps">LDAPS</option>
        </select>
      </div>

      {draft.encryption === 'none' ? (
        <div className="strip strip--warning">
          <span className="strip__title">Unencrypted</span>
          <span>
            Everything on this connection — the bind password included — crosses the network in the
            clear.
          </span>
        </div>
      ) : (
        <div className="field-row field-row--wrap">
          <span className="field-row__label">Certificate</span>
          <label className="field-group" style={{ flex: 'none' }}>
            <input
              type="checkbox"
              checked={draft.verifyCertificate}
              onChange={(e) => patch({ verifyCertificate: e.target.checked })}
            />
            <span>Verify chain</span>
          </label>
          <label className="field-group" style={{ flex: 'none' }}>
            <input
              type="checkbox"
              checked={draft.verifyHostname}
              onChange={(e) => patch({ verifyHostname: e.target.checked })}
            />
            <span>Verify hostname</span>
          </label>
          {!draft.verifyCertificate || !draft.verifyHostname ? (
            <span className="dim">
              Turning verification off is remembered and badged for as long as the connection is
              open.
            </span>
          ) : null}
        </div>
      )}

      <div className="actions">
        <button type="button" className="button" onClick={onCheck} disabled={busy || !draft.host}>
          {busy ? 'Checking…' : 'Check network parameter'}
        </button>
        <span className="dim">Opens a connection and reads the root DSE. Saves nothing.</span>
      </div>
    </>
  );
}

function BrowserOptionsStep({ draft, patch }: StepProps) {
  return (
    <>
      <span className="card__label">Browser options</span>

      <div className="field-row">
        <label className="field-row__label" htmlFor="conn-base">
          Base DN
        </label>
        <input
          id="conn-base"
          className="field field--mono"
          placeholder="read from the root DSE"
          value={draft.baseDn}
          onChange={(e) => patch({ baseDn: e.target.value })}
        />
      </div>

      <div className="field-row">
        <label className="field-row__label" htmlFor="conn-page">
          Page size
        </label>
        <input
          id="conn-page"
          className="field"
          inputMode="numeric"
          value={draft.pageSize}
          onChange={(e) => patch({ pageSize: e.target.value.replace(/\D/g, '') })}
          style={{ width: 90, flex: 'none' }}
        />
        <label className="field-row__label field-row__label--auto" htmlFor="conn-size">
          Size limit
        </label>
        <input
          id="conn-size"
          className="field"
          inputMode="numeric"
          value={draft.sizeLimit}
          onChange={(e) => patch({ sizeLimit: e.target.value.replace(/\D/g, '') })}
          style={{ width: 90, flex: 'none' }}
        />
        <label className="field-row__label field-row__label--auto" htmlFor="conn-time">
          Time limit (s)
        </label>
        <input
          id="conn-time"
          className="field"
          inputMode="numeric"
          value={draft.timeLimit}
          onChange={(e) => patch({ timeLimit: e.target.value.replace(/\D/g, '') })}
          style={{ width: 90, flex: 'none' }}
        />
      </div>

      <div className="field-row">
        <label className="field-row__label" htmlFor="conn-aliases">
          Aliases
        </label>
        <select
          id="conn-aliases"
          className="field"
          value={draft.aliases}
          onChange={(e) => patch({ aliases: e.target.value as ConnectionDraft['aliases'] })}
        >
          <option value="never">never dereference</option>
          <option value="search">dereference when searching</option>
          <option value="find">dereference when finding</option>
          <option value="always">always dereference</option>
        </select>
        <label className="field-row__label field-row__label--auto" htmlFor="conn-referrals">
          Referrals
        </label>
        <select
          id="conn-referrals"
          className="field"
          value={draft.referrals}
          onChange={(e) => patch({ referrals: e.target.value as ConnectionDraft['referrals'] })}
        >
          <option value="follow">follow</option>
          <option value="ignore">ignore</option>
          <option value="ask">ask</option>
        </select>
      </div>
    </>
  );
}

/**
 * Step 4 carries the two protections design gap G3 asked for. They are here
 * rather than buried in preferences because the moment a connection is
 * described is the moment the user knows which server it points at.
 */
function EditOptionsStep({ draft, patch }: StepProps) {
  return (
    <>
      <span className="card__label">Edit options</span>

      <div className="field-row">
        <span className="field-row__label">Open read-only</span>
        <label className="field-group">
          <input
            type="checkbox"
            style={{ flex: 'none' }}
            checked={draft.readOnly}
            onChange={(e) => patch({ readOnly: e.target.checked })}
          />
          <span className="dim">
            Refused at the changeset boundary, below any check the UI makes.
          </span>
        </label>
      </div>

      <div className="field-row">
        <span className="field-row__label">Tag as production</span>
        <label className="field-group">
          <input
            type="checkbox"
            style={{ flex: 'none' }}
            checked={draft.production}
            onChange={(e) => patch({ production: e.target.checked })}
          />
          <span className="dim">
            The status bar carries the tag and every write confirms first.
          </span>
        </label>
      </div>

      <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
        Auto-save on focus loss is deliberately not offered: a directory write should be an act, not
        a side effect of moving the cursor (deviation D3).
      </p>
    </>
  );
}
