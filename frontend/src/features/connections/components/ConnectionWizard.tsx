import { Field, FieldLabel } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { Alert } from '@/components/ui/alert';
import { Label } from '@/components/ui/label';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
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
              <Button
                variant="ghost"
                size="xs"
                key={label}
                type="button"
                className="wizard__step"
                data-active={index === step || undefined}
                data-done={index < step || undefined}
                onClick={() => setStep(index)}
              >
                {index + 1} · {label}
              </Button>
            ))}
            <TestSummary result={test} busy={testing} />
          </nav>

          <Card className="card wizard__body">
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
              <Alert className="strip strip--danger">
                <span className="strip__title">Failed</span>
                <span className="mono">{error}</span>
              </Alert>
            ) : null}

            {!isEmbedded() ? (
              <Alert className="strip strip--warning">
                <span className="strip__title">No backend</span>
                <span>
                  This interface is running outside the desktop shell, so nothing can be tested or
                  saved. Start it with <span className="mono">wails dev</span>.
                </span>
              </Alert>
            ) : null}

            <div className="actions actions--end" style={{ marginTop: 'auto' }}>
              {problems.length > 0 ? (
                <span className="dim" style={{ marginRight: 'auto' }}>
                  Still needs {problems.join(', ')}.
                </span>
              ) : null}
              <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
                Cancel
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={step === 0 || saving}
                onClick={() => setStep((s) => Math.max(0, s - 1))}
              >
                Back
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={step === STEPS.length - 1 || saving}
                onClick={() => setStep((s) => Math.min(STEPS.length - 1, s + 1))}
              >
                Next
              </Button>
              <Button
                type="button"
                variant="default"
                disabled={!canFinish}
                onClick={() => void finish()}
              >
                {saving ? 'Saving…' : 'Finish'}
              </Button>
            </div>
          </Card>
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
      <Alert className="strip strip--danger" style={{ marginTop: 'auto' }}>
        <span className="strip__title">Unreachable</span>
        <span className="mono">{result.message}</span>
      </Alert>
    );
  }

  return (
    <Card className="card card--tight" style={{ marginTop: 'auto' }}>
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
    </Card>
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

      <Field orientation="horizontal" className="field-row">
        <FieldLabel className="field-row__label" htmlFor="conn-name">
          Connection name
        </FieldLabel>
        <Input
          id="conn-name"
          className="field"
          placeholder={draft.host || 'local-dev'}
          value={draft.name}
          onChange={(e) => patch({ name: e.target.value })}
        />
      </Field>

      <Field orientation="horizontal" className="field-row">
        <FieldLabel className="field-row__label" htmlFor="conn-host">
          Hostname
        </FieldLabel>
        <Input
          id="conn-host"
          className="field"
          placeholder="localhost"
          value={draft.host}
          onChange={(e) => patch({ host: e.target.value })}
        />
        <FieldLabel className="field-row__label field-row__label--auto" htmlFor="conn-port">
          Port
        </FieldLabel>
        <Input
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
      </Field>

      <Field orientation="horizontal" className="field-row">
        <FieldLabel className="field-row__label" htmlFor="conn-encryption">
          Encryption
        </FieldLabel>
        <NativeSelect
          id="conn-encryption"
          className="field"
          value={draft.encryption}
          onChange={(e) => onEncryptionChange(e.target.value as Encryption)}
        >
          <option value="none">No encryption</option>
          <option value="startTLS">StartTLS</option>
          <option value="ldaps">LDAPS</option>
        </NativeSelect>
      </Field>

      {draft.encryption === 'none' ? (
        <Alert className="strip strip--warning">
          <span className="strip__title">Unencrypted</span>
          <span>
            Everything on this connection — the bind password included — crosses the network in the
            clear.
          </span>
        </Alert>
      ) : (
        <Field orientation="horizontal" className="field-row field-row--wrap">
          <span className="field-row__label">Certificate</span>
          <Label className="field-group" style={{ flex: 'none' }}>
            <Checkbox
              checked={draft.verifyCertificate}
              onCheckedChange={(checked) => patch({ verifyCertificate: checked })}
            />
            <span>Verify chain</span>
          </Label>
          <Label className="field-group" style={{ flex: 'none' }}>
            <Checkbox
              checked={draft.verifyHostname}
              onCheckedChange={(checked) => patch({ verifyHostname: checked })}
            />
            <span>Verify hostname</span>
          </Label>
          {!draft.verifyCertificate || !draft.verifyHostname ? (
            <span className="dim">
              Turning verification off is remembered and badged for as long as the connection is
              open.
            </span>
          ) : null}
        </Field>
      )}

      <div className="actions">
        <Button type="button" variant="outline" onClick={onCheck} disabled={busy || !draft.host}>
          {busy ? 'Checking…' : 'Check network parameter'}
        </Button>
        <span className="dim">Opens a connection and reads the root DSE. Saves nothing.</span>
      </div>
    </>
  );
}

function BrowserOptionsStep({ draft, patch }: StepProps) {
  return (
    <>
      <span className="card__label">Browser options</span>

      <Field orientation="horizontal" className="field-row">
        <FieldLabel className="field-row__label" htmlFor="conn-base">
          Base DN
        </FieldLabel>
        <Input
          id="conn-base"
          className="field field--mono"
          placeholder="read from the root DSE"
          value={draft.baseDn}
          onChange={(e) => patch({ baseDn: e.target.value })}
        />
      </Field>

      <Field orientation="horizontal" className="field-row">
        <FieldLabel className="field-row__label" htmlFor="conn-page">
          Page size
        </FieldLabel>
        <Input
          id="conn-page"
          className="field"
          inputMode="numeric"
          value={draft.pageSize}
          onChange={(e) => patch({ pageSize: e.target.value.replace(/\D/g, '') })}
          style={{ width: 90, flex: 'none' }}
        />
        <FieldLabel className="field-row__label field-row__label--auto" htmlFor="conn-size">
          Size limit
        </FieldLabel>
        <Input
          id="conn-size"
          className="field"
          inputMode="numeric"
          value={draft.sizeLimit}
          onChange={(e) => patch({ sizeLimit: e.target.value.replace(/\D/g, '') })}
          style={{ width: 90, flex: 'none' }}
        />
        <FieldLabel className="field-row__label field-row__label--auto" htmlFor="conn-time">
          Time limit (s)
        </FieldLabel>
        <Input
          id="conn-time"
          className="field"
          inputMode="numeric"
          value={draft.timeLimit}
          onChange={(e) => patch({ timeLimit: e.target.value.replace(/\D/g, '') })}
          style={{ width: 90, flex: 'none' }}
        />
      </Field>

      <Field orientation="horizontal" className="field-row">
        <FieldLabel className="field-row__label" htmlFor="conn-aliases">
          Aliases
        </FieldLabel>
        <NativeSelect
          id="conn-aliases"
          className="field"
          value={draft.aliases}
          onChange={(e) => patch({ aliases: e.target.value as ConnectionDraft['aliases'] })}
        >
          <option value="never">never dereference</option>
          <option value="search">dereference when searching</option>
          <option value="find">dereference when finding</option>
          <option value="always">always dereference</option>
        </NativeSelect>
        <FieldLabel className="field-row__label field-row__label--auto" htmlFor="conn-referrals">
          Referrals
        </FieldLabel>
        <NativeSelect
          id="conn-referrals"
          className="field"
          value={draft.referrals}
          onChange={(e) => patch({ referrals: e.target.value as ConnectionDraft['referrals'] })}
        >
          <option value="follow">follow</option>
          <option value="ignore">ignore</option>
          <option value="ask">ask</option>
        </NativeSelect>
      </Field>
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

      <Field orientation="horizontal" className="field-row">
        <span className="field-row__label">Open read-only</span>
        <Label className="field-group">
          <Checkbox
            style={{ flex: 'none' }}
            checked={draft.readOnly}
            onCheckedChange={(checked) => patch({ readOnly: checked })}
          />
          <span className="dim">
            Refused at the changeset boundary, below any check the UI makes.
          </span>
        </Label>
      </Field>

      <Field orientation="horizontal" className="field-row">
        <span className="field-row__label">Tag as production</span>
        <Label className="field-group">
          <Checkbox
            style={{ flex: 'none' }}
            checked={draft.production}
            onCheckedChange={(checked) => patch({ production: checked })}
          />
          <span className="dim">
            The status bar carries the tag and every write confirms first.
          </span>
        </Label>
      </Field>

      <p className="dim" style={{ margin: 0, fontSize: 'var(--text-caption)' }}>
        Auto-save on focus loss is deliberately not offered: a directory write should be an act, not
        a side effect of moving the cursor (deviation D3).
      </p>
    </>
  );
}
