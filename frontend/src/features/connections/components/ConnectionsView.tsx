import { Field } from '@/components/ui/field';
import { Card } from '@/components/ui/card';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
/**
 * Screen 1a — Connections: the list, and the wizard that adds to it.
 *
 * The connection list is the workspace tree root, not a gallery of cards: it
 * is the same shape as every other perspective's sidebar, because a connection
 * is navigated exactly like a DIT branch is.
 *
 * Connection state comes from the session store rather than local state, so
 * the list, the status bar and the browser perspective cannot disagree about
 * whether a server is open — they are all reading the same `conn:state` events
 * the Go side emits.
 */
import { useCallback, useEffect, useState } from 'react';

import { ConnectionWizard } from './ConnectionWizard';
import { bridge, isEmbedded } from '@/bridge/client';
import type { ConnState, ProfileSummary } from '@/bridge/types';
import { Workspace } from '@/shell/Workspace';
import { useSession } from '@/app/session';

export function ConnectionsView() {
  const [profiles, setProfiles] = useState<ProfileSummary[]>([]);
  const [selected, setSelected] = useState<string | undefined>();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [busy, setBusy] = useState<string | undefined>();
  const [error, setError] = useState<string | undefined>();

  const connections = useSession((s) => s.connections);
  const setActiveProfile = useSession((s) => s.setActiveProfile);
  const setPerspective = useSession((s) => s.setPerspective);

  const refresh = useCallback(async () => {
    if (!isEmbedded()) return;
    try {
      setProfiles(await bridge.listProfiles());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const stateOf = (id: string): ConnState['state'] => connections[id]?.state ?? 'disconnected';

  const connect = async (id: string) => {
    setError(undefined);
    setBusy(id);
    setActiveProfile(id);
    try {
      // Returns a job id; the connected state arrives as a conn:state event,
      // which the shell already routes into the store.
      await bridge.connect(id);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(undefined);
    }
  };

  const disconnect = async (id: string) => {
    setError(undefined);
    setBusy(id);
    try {
      await bridge.disconnect(id);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(undefined);
    }
  };

  const remove = async (id: string, name: string) => {
    // A connection is cheap to recreate but its stored secret is not
    // recoverable, so the confirmation names both.
    if (!window.confirm(`Delete the connection "${name}" and forget its stored secret?`)) return;
    setError(undefined);
    try {
      await bridge.deleteProfile(id);
      if (selected === id) setSelected(undefined);
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    }
  };

  const active = profiles.find((p) => p.id === selected);

  return (
    <div className="view">
      <aside className="sidebar" aria-label="Connections">
        <div className="sidebar__header">
          <span>Connections</span>
          <span className="sidebar__actions mono">
            <Button
              type="button"
              variant="ghost"
              size="xs"
              title="New connection"
              aria-label="New connection"
              onClick={() => setWizardOpen(true)}
            >
              +
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              title="Refresh"
              aria-label="Refresh"
              onClick={() => void refresh()}
            >
              ⟳
            </Button>
          </span>
        </div>

        <div className="sidebar__body">
          <div className="sidebar__section">LDAP servers</div>

          {profiles.length === 0 ? (
            <p className="sidebar__note">
              {isEmbedded() ? 'no connections yet — use + to add one' : 'no backend'}
            </p>
          ) : (
            profiles.map((profile) => {
              const state = stateOf(profile.id);
              return (
                <Button
                  variant="ghost"
                  size="xs"
                  key={profile.id}
                  type="button"
                  className="tree-row"
                  data-selected={selected === profile.id || undefined}
                  onClick={() => {
                    setSelected(profile.id);
                    setActiveProfile(profile.id);
                  }}
                  onDoubleClick={() => void connect(profile.id)}
                  title={`${profile.host}:${profile.port}`}
                >
                  <span className="tree-row__twisty" aria-hidden="true" />
                  <StateDot state={state} />
                  <span>{profile.name}</span>
                  {profile.readOnly ? <span className="tree-row__count">read-only</span> : null}
                </Button>
              );
            })
          )}
        </div>
      </aside>

      <Workspace>
        {wizardOpen ? (
          <ConnectionWizard
            onClose={() => setWizardOpen(false)}
            onSaved={(id) => {
              void refresh();
              setSelected(id);
              setActiveProfile(id);
            }}
          />
        ) : (
          <ConnectionDetail
            profile={active}
            state={active ? stateOf(active.id) : 'disconnected'}
            message={active ? connections[active.id]?.message : undefined}
            busy={busy === active?.id}
            error={error}
            onNew={() => setWizardOpen(true)}
            onConnect={() => active && void connect(active.id)}
            onDisconnect={() => active && void disconnect(active.id)}
            onDelete={() => active && void remove(active.id, active.name)}
            onBrowse={() => setPerspective('browser')}
          />
        )}
      </Workspace>
    </div>
  );
}

function StateDot({ state }: { state: ConnState['state'] }) {
  const colour =
    state === 'connected'
      ? 'var(--success)'
      : state === 'connecting'
        ? 'var(--warning)'
        : state === 'lost'
          ? 'var(--danger)'
          : 'var(--border-strong)';
  return (
    <span
      aria-hidden="true"
      className="tree-row__glyph tree-row__glyph--entry"
      style={{ background: colour, borderColor: colour }}
    />
  );
}

function ConnectionDetail({
  profile,
  state,
  message,
  busy,
  error,
  onNew,
  onConnect,
  onDisconnect,
  onDelete,
  onBrowse,
}: {
  profile?: ProfileSummary;
  state: ConnState['state'];
  message?: string;
  busy: boolean;
  error?: string;
  onNew: () => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onDelete: () => void;
  onBrowse: () => void;
}) {
  if (!profile) {
    return (
      <div className="pane pane--scroll" style={{ flex: 1 }}>
        <p className="view-title">Connections</p>
        <Card className="card" style={{ maxWidth: '72ch' }}>
          <p style={{ margin: 0, color: 'var(--text-secondary)' }}>
            A connection records where a directory is and how to bind to it. The secret itself is
            held by the platform credential store, never in the connection file.
          </p>
          <div className="actions">
            <Button type="button" variant="default" onClick={onNew}>
              New connection
            </Button>
          </div>
        </Card>
        {error ? (
          <Alert className="strip strip--danger">
            <span className="strip__title">Failed</span>
            <span className="mono">{error}</span>
          </Alert>
        ) : null}
      </div>
    );
  }

  const connected = state === 'connected';

  return (
    <>
      <div className="toolbar">
        <span style={{ color: 'var(--text-primary)' }}>{profile.name}</span>
        <span className="dim mono">
          {profile.host}:{profile.port}
        </span>
        <span className="toolbar__spacer" />
        <div className="actions">
          {connected ? (
            <>
              <Button type="button" variant="outline" onClick={onBrowse}>
                Browse
              </Button>
              <Button type="button" variant="outline" onClick={onDisconnect} disabled={busy}>
                Disconnect
              </Button>
            </>
          ) : (
            <Button
              type="button"
              variant="default"
              onClick={onConnect}
              disabled={busy || state === 'connecting'}
            >
              {busy || state === 'connecting' ? 'Connecting…' : 'Connect'}
            </Button>
          )}
          <Button type="button" variant="destructive" onClick={onDelete} disabled={busy}>
            Delete
          </Button>
        </div>
      </div>

      <div className="pane pane--scroll" style={{ flex: 1 }}>
        <Card className="card card--tight" style={{ maxWidth: '72ch' }}>
          <span className="card__label">Connection</span>
          <Fact label="State" value={state} />
          <Fact label="Host" value={`${profile.host}:${profile.port}`} />
          <Fact label="Encryption" value={profile.encryption} />
          <Fact label="Mode" value={profile.readOnly ? 'read-only' : 'read / write'} />
          {profile.tags?.length ? <Fact label="Tags" value={profile.tags.join(', ')} /> : null}
        </Card>

        {/* The server's own words about a failure, not a paraphrase. */}
        {message ? (
          <div className={connected ? 'strip' : 'strip strip--danger'} style={{ maxWidth: '72ch' }}>
            <span className="strip__title">{connected ? 'Note' : 'Last attempt'}</span>
            <span className="mono">{message}</span>
          </div>
        ) : null}

        {error ? (
          <Alert className="strip strip--danger" style={{ maxWidth: '72ch' }}>
            <span className="strip__title">Failed</span>
            <span className="mono">{error}</span>
          </Alert>
        ) : null}

        {connected ? (
          <Alert className="strip strip--success" style={{ maxWidth: '72ch' }}>
            <span className="strip__title">Open</span>
            <span>The DIT browser can read through this connection now.</span>
          </Alert>
        ) : null}
      </div>
    </>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <Field orientation="horizontal" className="field-row">
      <span className="field-row__label">{label}</span>
      <span className="mono">{value}</span>
    </Field>
  );
}
