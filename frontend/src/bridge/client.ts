/**
 * The typed client for the Go bridge.
 *
 * Wails exposes bound methods on `window.go.<package>.<struct>.<Method>`. This
 * module wraps that one lookup so the rest of the frontend never touches the
 * global, and so a build can run — and the tests can run — without a webview
 * behind it.
 *
 * The write surface is deliberately small: `preview` then `commit(token)`.
 * There is no `modify`, `add`, `delete` or `rename` here because there is none
 * on the Go side either (bridge-api.md Rule 1).
 */
import type {
  AppInfo,
  ChangeSet,
  ChangeSetInput,
  ChangeSetPreview,
  CommandSet,
  ConnState,
  Entry,
  FilterDiagnostic,
  Job,
  Outcome,
  Page,
  PageRequest,
  Profile,
  ProfileSummary,
  Result,
  RootDSE,
  TestResult,
} from './types';

type BoundMethod = (...args: unknown[]) => Promise<unknown>;

interface WailsWindow {
  go?: Record<string, Record<string, Record<string, BoundMethod>>>;
  runtime?: {
    EventsOn: (name: string, handler: (...data: unknown[]) => void) => () => void;
    EventsOff: (name: string) => void;
    WindowMinimise?: () => void;
    WindowToggleMaximise?: () => void;
    Quit?: () => void;
  };
}

const BOUND_PACKAGE = 'bridge';
const BOUND_STRUCT = 'Bridge';

/** True when the app is running inside the Wails webview. */
export function isEmbedded(): boolean {
  const w = window as unknown as WailsWindow;
  return Boolean(w.go?.[BOUND_PACKAGE]?.[BOUND_STRUCT]);
}

export class BridgeUnavailableError extends Error {
  constructor(method: string) {
    super(
      `Bridge.${method} is not available: the interface is running outside the desktop shell. ` +
        `Start it with "wails dev" rather than "vite dev".`,
    );
    this.name = 'BridgeUnavailableError';
  }
}

async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  const w = window as unknown as WailsWindow;
  const bound = w.go?.[BOUND_PACKAGE]?.[BOUND_STRUCT]?.[method];
  if (!bound) {
    throw new BridgeUnavailableError(method);
  }
  return (await bound(...args)) as T;
}

export const bridge = {
  // --- Application ---
  getAppInfo: () => call<AppInfo>('GetAppInfo'),
  credentialStoreStatus: () => call<[boolean, string]>('CredentialStoreStatus'),

  // --- Commands, menus and keymaps: one registry, three views ---
  getCommands: () => call<CommandSet>('GetCommands'),
  searchCommands: (query: string) => call<CommandSet['commands']>('SearchCommands', query),

  // --- Connections ---
  listProfiles: () => call<ProfileSummary[]>('ListProfiles'),
  getProfile: (id: string) => call<Profile>('GetProfile', id),
  /**
   * The payload is serialised here rather than passed as an object because the
   * Go side decodes it strictly: a profile carrying a password field is
   * refused outright rather than silently stripped (SC-008).
   */
  saveProfile: (profile: Partial<Profile>) => call<Profile>('SaveProfile', JSON.stringify(profile)),
  deleteProfile: (id: string) => call<void>('DeleteProfile', id),
  duplicateProfile: (id: string) => call<Profile>('DuplicateProfile', id),

  /**
   * Dials a profile that has not been saved, and binds it if it names a bind
   * method. Persists nothing — no profile, no secret, no open connection — so
   * the wizard can check its work on every page without committing to it.
   *
   * The secret travels as its own argument rather than inside the payload,
   * because the payload is decoded strictly and a profile has nowhere to put
   * one.
   */
  testConnection: (profile: Partial<Profile>, secret: string) =>
    call<TestResult>('TestConnection', JSON.stringify(profile), secret),

  /** Binds an already-saved profile with a one-off secret, then closes. */
  testBind: (profileId: string, secret: string) => call<TestResult>('TestBind', profileId, secret),

  /**
   * Files a bind secret in the platform credential store and points the
   * profile at it. An empty secret removes the stored one, which is how a
   * connection is moved back to prompting.
   */
  storeProfileSecret: (profileId: string, secret: string) =>
    call<void>('StoreProfileSecret', profileId, secret),
  /** Whether a secret is on file. Never reveals anything about it. */
  profileHasSecret: (profileId: string) => call<boolean>('ProfileHasSecret', profileId),
  forgetProfileSecret: (profileId: string) => call<void>('ForgetProfileSecret', profileId),

  /** Returns a job id: binding may raise the platform credential prompt. */
  connect: (profileId: string) => call<string>('Connect', profileId),
  disconnect: (profileId: string) => call<void>('Disconnect', profileId),
  connectionState: (profileId: string) => call<ConnState>('ConnectionState', profileId),
  connectionStates: () => call<ConnState[]>('ConnectionStates'),
  whoAmI: (profileId: string) => call<[string, Result]>('WhoAmI', profileId),
  rootDSE: (profileId: string) => call<[RootDSE, Result]>('RootDSE', profileId),

  // --- Browsing ---
  listChildren: (profileId: string, dn: string, page: PageRequest) =>
    call<Page>('ListChildren', profileId, dn, page),
  readEntry: (profileId: string, dn: string, includeOperational = false) =>
    call<[Entry, Result]>('ReadEntry', profileId, dn, { includeOperational, attributes: [] }),

  /** Local only. It never contacts a server and never rewrites the filter. */
  validateFilter: (filter: string) => call<FilterDiagnostic>('ValidateFilter', filter),

  // --- The entire write path ---
  preview: (input: ChangeSetInput, confirmedProduction = false) =>
    call<ChangeSetPreview>('Preview', input, confirmedProduction),
  pending: (token: string) => call<ChangeSet>('Pending', token),
  /** The only dispatch path. It takes a token and nothing else. */
  commit: (token: string) => call<Result>('Commit', token),
  commitAsJob: (token: string) => call<string>('CommitAsJob', token),
  /** Drops the token and keeps the draft (FR-040). */
  discard: (token: string) => call<void>('Discard', token),

  // --- Jobs ---
  cancel: (jobId: string) => call<void>('Cancel', jobId),
  jobState: (jobId: string) => call<Job>('JobState', jobId),
  listJobs: () => call<Job[]>('ListJobs'),
  jobOutcomes: (jobId: string) => call<Outcome[]>('JobOutcomes', jobId),
  pauseJob: (jobId: string) => call<void>('PauseJob', jobId),
  resumeJob: (jobId: string) => call<void>('ResumeJob', jobId),

  // --- Trust ---
  listTrustDecisions: () => call<unknown[]>('ListTrustDecisions'),
  decideTrust: (
    host: string,
    port: number,
    fingerprint: string,
    scope: 'session' | 'permanent',
    reason: string,
  ) => call<void>('DecideTrust', host, port, fingerprint, scope, reason),
  revokeTrustDecision: (fingerprint: string) => call<void>('RevokeTrustDecision', fingerprint),
};

/** Window controls, for the frameless title bar the wireframe draws. */
export const windowControls = {
  minimise: () => (window as unknown as WailsWindow).runtime?.WindowMinimise?.(),
  toggleMaximise: () => (window as unknown as WailsWindow).runtime?.WindowToggleMaximise?.(),
  quit: () => (window as unknown as WailsWindow).runtime?.Quit?.(),
};
