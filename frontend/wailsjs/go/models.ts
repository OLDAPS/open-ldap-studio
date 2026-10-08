export namespace bridge {
	
	export interface AppInfo {
	    version: string;
	    credentialStore: string;
	    credentialStoreReason: string;
	    updateChecksEnabled: boolean;
	    platform: string;
	}
	export interface Bookmark {
	    id: string;
	    name: string;
	    dn: string;
	}
	export interface CommandSet {
	    platform: string;
	    menus: string[];
	    tree: Record<string, Array<commands.MenuItem>>;
	    commands: commands.Command[];
	    bindings: Record<string, string>;
	}
	export interface SavedSearch {
	    id: string;
	    name: string;
	    def: ldapx.SearchDefinition;
	}
	export interface Template {
	    id: string;
	    name: string;
	    dn: string;
	}
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
	    result: ldapx.Result;
	    message?: string;
	}

}

export namespace changeset {
	
	export interface Warning {
	    severity: string;
	    dn?: string;
	    message: string;
	}
	export interface Operation {
	    dn: string;
	    type: string;
	    attribute?: ldapx.Attribute;
	    attributes?: ldapx.Attribute[];
	    before?: string[];
	    after?: string[];
	    newRdn?: string;
	    newSuperior?: string;
	    keepOldRdn?: boolean;
	}
	export interface ChangeSet {
	    id: string;
	    profileId: string;
	    kind: string;
	    ops: Operation[];
	    affectedCount: number;
	    beforeState?: ldapx.Entry[];
	    warnings?: Warning[];
	}
	export interface Input {
	    profileId: string;
	    kind: string;
	    ops: Operation[];
	}
	
	export interface Token {
	    token: string;
	    changeSetId: string;
	    entryVersions: Record<string, string>;
	    issuedAt: string;
	    expiresAt: string;
	}
	export interface Preview {
	    changeSet: ChangeSet;
	    profileName: string;
	    serverIdentity: string;
	    readOnly: boolean;
	    production: boolean;
	    token: Token;
	    reversible: boolean;
	}
	

}

export namespace commands {
	
	export interface Command {
	    id: string;
	    label: string;
	    menu?: string;
	    path?: string[];
	    group?: string;
	    order: number;
	    scope: string;
	    enablement: string;
	    bindings?: Record<string, string>;
	    destructive?: boolean;
	}
	export interface Conflict {
	    chord: string;
	    scope: string;
	    commandIds: string[];
	}
	export interface Keymap {
	    preset: string;
	    platform: string;
	    bindings: Record<string, string>;
	}
	export interface MenuItem {
	    commandId?: string;
	    label: string;
	    chord?: string;
	    group?: string;
	    enablement?: string;
	    destructive?: boolean;
	    items?: MenuItem[];
	}

}

export namespace connections {
	
	export interface ConnState {
	    profileId: string;
	    state: string;
	    boundDn: string;
	    serverIdentity: string;
	    tlsVerified: boolean;
	    encrypted: boolean;
	    readOnly: boolean;
	    production: boolean;
	    writesRequireConfirmation: boolean;
	    message?: string;
	}

}

export namespace credentials {
	
	export interface Credential {
	    id: string;
	    name: string;
	    kind: string;
	    bindDn?: string;
	    realm?: string;
	    secretRef?: string;
	    createdAt: string;
	    rotatedAt: string;
	    rotateAfterDays?: number;
	    storeAvailable: boolean;
	    storeReason?: string;
	}

}

export namespace history {
	
	export interface SearchRecord {
	    id: string;
	    profileId: string;
	    filter: string;
	    baseDN: string;
	    timestamp: number;
	}

}

export namespace jobs {
	
	export interface Job {
	    id: string;
	    kind: string;
	    mode: string;
	    state: string;
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
	    status: string;
	    result?: ldapx.Result;
	}

}

export namespace ldapx {
	
	export interface Attribute {
	    type: string;
	    options?: string[];
	    values: string[];
	    isOperational: boolean;
	}
	export interface Control {
	    oid: string;
	    critical: boolean;
	    value?: string;
	}
	export interface Entry {
	    dn: string;
	    attributes: Attribute[];
	    hasChildren: string;
	}
	export interface FilterDiagnostic {
	    ok: boolean;
	    position: number;
	    message: string;
	}
	export interface Result {
	    code: number;
	    matchedDn: string;
	    diagnosticMessage: string;
	    interpretation: string;
	    referrals?: string[];
	    controls?: Control[];
	}
	export interface Page {
	    entries: Entry[];
	    cookie?: string;
	    loadedCount: number;
	    serverLimit: number;
	    truncatedByServer: boolean;
	    result: Result;
	}
	export interface PageRequest {
	    size: number;
	    cookie?: string;
	    includeOperational: boolean;
	}
	export interface ReadOptions {
	    IncludeOperational: boolean;
	    Attributes: string[];
	    ManageDsaIT: boolean;
	}
	export interface ReferralTarget {
	    Host: string;
	    Port: number;
	    BaseDN: string;
	    Scope: string;
	    Filter: string;
	    Extension: string;
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
	    raw: Record<string, Array<string>>;
	}
	export interface SearchDefinition {
	    baseDN: string;
	    filter: string;
	    scope: number;
	    attributes: string[];
	    includeOperational: boolean;
	    sizeLimit: number;
	    timeLimit: number;
	    aliases: string;
	    manageDsaIT: boolean;
	}

}

export namespace profiles {
	
	export interface Folder {
	    id: string;
	    name: string;
	    parentId?: string;
	}
	export interface Limits {
	    sizeLimit: number;
	    timeLimit: number;
	    pageSize: number;
	}
	export interface Timeouts {
	    connectMs: number;
	    readMs: number;
	}
	export interface TLSPolicy {
	    verifyCertificate: boolean;
	    verifyHostname: boolean;
	    clientCertRef?: string;
	}
	export interface Profile {
	    id: string;
	    name: string;
	    folderId?: string;
	    host: string;
	    port: number;
	    encryption: string;
	    tls: TLSPolicy;
	    bindMethod: string;
	    bindDn: string;
	    credentialId?: string;
	    readOnly: boolean;
	    tags?: string[];
	    baseDn?: string;
	    timeouts: Timeouts;
	    limits: Limits;
	    aliases: string;
	    referrals: string;
	    schemaVersion: number;
	}
	export interface Summary {
	    id: string;
	    name: string;
	    folderId?: string;
	    host: string;
	    port: number;
	    encryption: string;
	    readOnly: boolean;
	    tags?: string[];
	}
	

}

export namespace trust {
	
	export interface Decision {
	    host: string;
	    port: number;
	    fingerprint: string;
	    scope: string;
	    chain?: number[][];
	    acceptedAt: string;
	    reason: string;
	}

}

