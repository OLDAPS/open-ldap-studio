import type {
  bridge as generatedBridge,
  connections,
  commands,
  changeset,
  jobs,
  ldapx,
  profiles,
} from '../../wailsjs/go/models';

/**
 * The shapes the Go bridge speaks.
 *
 * Transport models are generated from Go. Narrow string unions below describe
 * frontend presentation states. Two rules from specs/001-open-ldap-studio/contracts/
 * remain visible in the types:
 *
 *  - `Result` is returned on success as well as failure, and always carries
 *    the server's own `diagnosticMessage`. `interpretation` is additive; a UI
 *    that renders only one of the two is a defect (SC-006).
 *  - Attribute values are byte arrays, base64-encoded across the bridge. They
 *    are never coerced to text outside a presentation layer that can turn them
 *    back into the same bytes (SC-007).
 */

export type Result = ldapx.Result;

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

export type Attribute = ldapx.Attribute;

export type Children = 'yes' | 'no' | 'unknown';

export type Entry = Omit<ldapx.Entry, 'hasChildren'> & { hasChildren: Children };

export type Page = Omit<ldapx.Page, 'entries'> & { entries: Entry[] };

export type PageRequest = ldapx.PageRequest;

export type Encryption = 'none' | 'startTLS' | 'ldaps';
export type BindMethod = 'anonymous' | 'simple' | 'external' | 'gssapi' | 'digestMD5' | 'cramMD5';

export type ProfileSummary = Omit<profiles.Summary, 'encryption'> & { encryption: Encryption };

export type Profile = Omit<
  profiles.Profile,
  'encryption' | 'bindMethod' | 'aliases' | 'referrals'
> & {
  encryption: Encryption;
  bindMethod: BindMethod;
  aliases: 'never' | 'search' | 'find' | 'always';
  referrals: 'follow' | 'ignore' | 'ask';
};

/**
 * What a connection test reports.
 *
 * It carries what the server said rather than a verdict: `reachable` and
 * `bound` are separate because "the host answered but rejected me" and "the
 * host never answered" are different problems with different fixes, and
 * `result` keeps the server's own words for the second.
 */
export type TestResult = generatedBridge.TestResult;

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'lost';

export type ConnState = Omit<connections.ConnState, 'state'> & { state: ConnectionState };

export type JobState = 'running' | 'succeeded' | 'failed' | 'cancelled' | 'partiallyComplete';

export type Job = Omit<jobs.Job, 'mode' | 'state'> & {
  mode: 'execute' | 'dryRun';
  state: JobState;
};

export type Outcome = Omit<jobs.Outcome, 'status'> & { status: 'succeeded' | 'failed' | 'skipped' };

export type OpType =
  | 'addAttr'
  | 'deleteAttr'
  | 'replaceAttr'
  | 'addValue'
  | 'deleteValue'
  | 'addEntry'
  | 'deleteEntry'
  | 'rename';

export type Operation = Omit<changeset.Operation, 'type'> & { type: OpType };

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

export type Warning = Omit<changeset.Warning, 'severity'> & { severity: 'warning' | 'error' };

export type ChangeSetInput = Omit<changeset.Input, 'kind' | 'ops'> & {
  kind: ChangeKind;
  ops: Operation[];
};

export type ChangeSet = Omit<changeset.ChangeSet, 'kind' | 'ops' | 'beforeState' | 'warnings'> & {
  kind: ChangeKind;
  ops: Operation[];
  beforeState?: Entry[];
  warnings?: Warning[];
};

export type PreviewToken = changeset.Token;

export type ChangeSetPreview = Omit<changeset.Preview, 'changeSet'> & { changeSet: ChangeSet };

export type FilterDiagnostic = ldapx.FilterDiagnostic;

export type RootDSE = ldapx.RootDSE;

export type Platform = 'linux' | 'darwin' | 'windows';
export type MenuName =
  'File' | 'Edit' | 'Search' | 'LDAP' | 'Schema' | 'Credentials' | 'Preferences' | 'Help';

export type MenuItem = commands.MenuItem;

export type Command = Omit<commands.Command, 'menu' | 'scope' | 'bindings'> & {
  menu?: MenuName;
  scope: 'global' | 'tree' | 'grid' | 'textEditor';
  bindings?: Partial<Record<Platform, string>>;
};

export type CommandSet = Omit<
  generatedBridge.CommandSet,
  'platform' | 'menus' | 'tree' | 'commands'
> & {
  platform: Platform;
  menus: MenuName[];
  tree: Record<MenuName, MenuItem[]>;
  commands: Command[];
};

export type AppInfo = Omit<generatedBridge.AppInfo, 'platform'> & { platform: Platform };

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
