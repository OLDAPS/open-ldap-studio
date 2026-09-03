/**
 * The shapes the Go bridge speaks.
 *
 * These mirror internal/bridge and the contracts in
 * specs/001-open-ldap-studio/contracts/. Two rules from those contracts are
 * visible in the types themselves, and both are load-bearing:
 *
 *  - `Result` is returned on success as well as failure, and always carries
 *    the server's own `diagnosticMessage`. `interpretation` is additive; a UI
 *    that renders only one of the two is a defect (SC-006).
 *  - Attribute values are byte arrays, base64-encoded across the bridge. They
 *    are never coerced to text outside a presentation layer that can turn them
 *    back into the same bytes (SC-007).
 */

export interface Result {
  code: number;
  matchedDn: string;
  /** Verbatim, byte-for-byte as the server sent it. Never rewritten. */
  diagnosticMessage: string;
  /** Additive plain-language text, shown beside the message, never instead. */
  interpretation: string;
  referrals?: string[];
}

export type ErrorCategory =
  | 'serverError'
  | 'authError'
  | 'transportError'
  | 'trustError'
  | 'credentialStoreError'
  | 'localValidationError'
  | 'capabilityError'
  | 'indeterminate'
  | 'cancelled';

export interface Attribute {
  type: string;
  options?: string[];
  /** Base64 across the bridge; decoded only where it can be re-encoded. */
  values: string[];
  isOperational: boolean;
}

export type Children = 'yes' | 'no' | 'unknown';

export interface Entry {
  dn: string;
  attributes: Attribute[];
  hasChildren: Children;
}

export interface Page {
  entries: Entry[];
  cookie?: string;
  loadedCount: number;
  serverLimit: number;
  /** The server stopped early; the list shown is partial and says so. */
  truncatedByServer: boolean;
  result: Result;
}

export interface PageRequest {
  size: number;
  cookie?: string;
  includeOperational: boolean;
}

export type Encryption = 'none' | 'startTLS' | 'ldaps';
export type BindMethod = 'anonymous' | 'simple' | 'external' | 'gssapi' | 'digestMD5' | 'cramMD5';

export interface ProfileSummary {
  id: string;
  name: string;
  folderId?: string;
  host: string;
  port: number;
  encryption: Encryption;
  readOnly: boolean;
  tags?: string[];
}

export interface Profile extends ProfileSummary {
  tls: { verifyCertificate: boolean; verifyHostname: boolean; clientCertRef?: string };
  bindMethod: BindMethod;
  bindDn: string;
  /** A reference to a credential. Never a secret — there is nowhere to put one. */
  credentialId?: string;
  baseDn?: string;
  timeouts: { connectMs: number; readMs: number };
  limits: { sizeLimit: number; timeLimit: number; pageSize: number };
  aliases: 'never' | 'search' | 'find' | 'always';
  referrals: 'follow' | 'ignore' | 'ask';
  schemaVersion: number;
}

/**
 * What a connection test reports.
 *
 * It carries what the server said rather than a verdict: `reachable` and
 * `bound` are separate because "the host answered but rejected me" and "the
 * host never answered" are different problems with different fixes, and
 * `result` keeps the server's own words for the second.
 */
export interface TestResult {
  reachable: boolean;
  encrypted: boolean;
  tlsVerified: boolean;
  bound: boolean;
  boundDn?: string;
  vendorName?: string;
  vendorVersion?: string;
  namingContexts?: string[];
  saslMechanisms?: string[];
  result: Result;
  /** A failure that happened before the server could answer at all. */
  message?: string;
}

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'lost';

export interface ConnState {
  profileId: string;
  state: ConnectionState;
  boundDn: string;
  serverIdentity: string;
  tlsVerified: boolean;
  encrypted: boolean;
  readOnly: boolean;
  production: boolean;
  writesRequireConfirmation: boolean;
  message?: string;
}

export type JobState = 'running' | 'succeeded' | 'failed' | 'cancelled' | 'partiallyComplete';

export interface Job {
  id: string;
  kind: string;
  mode: 'execute' | 'dryRun';
  state: JobState;
  profileId?: string;
  total: number;
  done: number;
  message: string;
  startedAt: string;
  endedAt?: string;
  summary?: string;
  reportPath?: string;
  error?: string;
  paused: boolean;
}

export interface Outcome {
  dn: string;
  status: 'succeeded' | 'failed' | 'skipped';
  result?: Result;
}

export type OpType =
  | 'addAttr'
  | 'deleteAttr'
  | 'replaceAttr'
  | 'addValue'
  | 'deleteValue'
  | 'addEntry'
  | 'deleteEntry'
  | 'rename';

export interface Operation {
  dn: string;
  type: OpType;
  attribute?: Attribute;
  attributes?: Attribute[];
  before?: string[];
  after?: string[];
  newRdn?: string;
  newSuperior?: string;
  keepOldRdn?: boolean;
}

export type ChangeKind =
  | 'add'
  | 'modify'
  | 'modRDN'
  | 'delete'
  | 'subtreeDelete'
  | 'copy'
  | 'move'
  | 'bulkModify'
  | 'schemaCommit'
  | 'configModify';

export interface Warning {
  severity: 'warning' | 'error';
  dn?: string;
  message: string;
}

export interface ChangeSetInput {
  profileId: string;
  kind: ChangeKind;
  ops: Operation[];
}

export interface ChangeSet extends ChangeSetInput {
  id: string;
  affectedCount: number;
  beforeState?: Entry[];
  warnings?: Warning[];
}

export interface PreviewToken {
  token: string;
  changeSetId: string;
  entryVersions: Record<string, string>;
  issuedAt: string;
  expiresAt: string;
}

export interface ChangeSetPreview {
  changeSet: ChangeSet;
  profileName: string;
  serverIdentity: string;
  readOnly: boolean;
  production: boolean;
  token: PreviewToken;
  reversible: boolean;
}

export interface FilterDiagnostic {
  ok: boolean;
  position: number;
  message: string;
}

export interface RootDSE {
  namingContexts: string[];
  supportedControl: string[];
  supportedExtension: string[];
  supportedSaslMechanisms: string[];
  supportedLdapVersion: string[];
  subschemaSubentry: string;
  vendorName?: string;
  vendorVersion?: string;
  configContext?: string;
  raw: Record<string, string[]>;
}

export type Platform = 'linux' | 'darwin' | 'windows';
export type MenuName =
  'File' | 'Edit' | 'Search' | 'LDAP' | 'Schema' | 'Credentials' | 'Preferences' | 'Help';

export interface MenuItem {
  commandId?: string;
  label: string;
  chord?: string;
  group?: string;
  enablement?: string;
  destructive?: boolean;
  items?: MenuItem[];
}

export interface Command {
  id: string;
  label: string;
  menu?: MenuName;
  path?: string[];
  group?: string;
  order: number;
  scope: 'global' | 'tree' | 'grid' | 'textEditor';
  enablement: string;
  bindings?: Partial<Record<Platform, string>>;
  destructive?: boolean;
}

export interface CommandSet {
  platform: Platform;
  menus: MenuName[];
  tree: Record<MenuName, MenuItem[]>;
  commands: Command[];
  bindings: Record<string, string>;
}

export interface AppInfo {
  version: string;
  credentialStore: string;
  credentialStoreReason: string;
  updateChecksEnabled: boolean;
  platform: Platform;
}

export interface TrustChallenge {
  host: string;
  port: number;
  fingerprint: string;
  chainPem: string[];
  failureReason: string;
  previouslyTrusted: boolean;
}

export interface CapabilityUnavailable {
  profileId: string;
  capability: string;
  reason: string;
}
