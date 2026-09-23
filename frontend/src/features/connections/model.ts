/**
 * The connection draft the wizard edits.
 *
 * It is deliberately not a `Profile`. A Profile has nowhere to put a secret —
 * that is enforced on the Go side, where a payload carrying one is refused
 * outright — so the draft keeps the secret beside the profile fields rather
 * than inside them, and `toProfile` is the boundary where it is dropped.
 */
import type { BindMethod, Encryption, Profile } from '@/bridge/types';

export interface ConnectionDraft {
  id?: string;
  name: string;
  host: string;
  /** Text, not a number: a half-typed or cleared port must be representable. */
  port: string;
  encryption: Encryption;
  verifyCertificate: boolean;
  verifyHostname: boolean;

  bindMethod: BindMethod;
  bindDn: string;
  /** Never sent inside the profile payload. */
  secret: string;

  baseDn: string;
  pageSize: string;
  sizeLimit: string;
  timeLimit: string;
  aliases: Profile['aliases'];
  referrals: Profile['referrals'];

  readOnly: boolean;
  production: boolean;
}

/** The port each transport is registered on, used until the user names one. */
export const DEFAULT_PORT: Record<Encryption, string> = {
  none: '389',
  startTLS: '389',
  ldaps: '636',
};

export function emptyDraft(): ConnectionDraft {
  return {
    name: '',
    host: '',
    port: DEFAULT_PORT.startTLS,
    encryption: 'startTLS',
    verifyCertificate: true,
    verifyHostname: true,
    bindMethod: 'simple',
    bindDn: '',
    secret: '',
    baseDn: '',
    pageSize: '100',
    sizeLimit: '1000',
    timeLimit: '10',
    aliases: 'find',
    referrals: 'follow',
    readOnly: false,
    production: false,
  };
}

/** Loads an existing profile back into the wizard. The secret is not fetched:
 *  nothing in this application can read one back out of the platform store. */
export function draftFromProfile(profile: Profile): ConnectionDraft {
  return {
    ...emptyDraft(),
    id: profile.id,
    name: profile.name,
    host: profile.host,
    port: String(profile.port ?? ''),
    encryption: profile.encryption,
    verifyCertificate: profile.tls?.verifyCertificate ?? true,
    verifyHostname: profile.tls?.verifyHostname ?? true,
    bindMethod: profile.bindMethod,
    bindDn: profile.bindDn ?? '',
    secret: '',
    baseDn: profile.baseDn ?? '',
    pageSize: String(profile.limits?.pageSize ?? 100),
    sizeLimit: String(profile.limits?.sizeLimit ?? 1000),
    timeLimit: String(profile.limits?.timeLimit ?? 10),
    aliases: profile.aliases ?? 'find',
    referrals: profile.referrals ?? 'follow',
    readOnly: Boolean(profile.readOnly),
    production: (profile.tags ?? []).includes('production'),
  };
}

const number = (text: string, fallback: number) => {
  const parsed = Number.parseInt(text, 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

/**
 * The draft as the Go side will accept it.
 *
 * Every key here is one `profiles.Profile` declares. The decoder rejects
 * anything else, so an extra field added carelessly fails loudly at save time
 * rather than being dropped in silence — which is the point of that decoder.
 */
export function toProfile(draft: ConnectionDraft): Partial<Profile> {
  return {
    ...(draft.id ? { id: draft.id } : {}),
    name: draft.name.trim() || draft.host.trim(),
    host: draft.host.trim(),
    port: number(draft.port, draft.encryption === 'ldaps' ? 636 : 389),
    encryption: draft.encryption,
    tls: {
      verifyCertificate: draft.verifyCertificate,
      verifyHostname: draft.verifyHostname,
    },
    bindMethod: draft.bindMethod,
    bindDn: draft.bindMethod === 'anonymous' ? '' : draft.bindDn.trim(),
    readOnly: draft.readOnly,
    tags: draft.production ? ['production'] : [],
    baseDn: draft.baseDn.trim(),
    timeouts: { connectMs: 10_000, readMs: 30_000 },
    limits: {
      pageSize: number(draft.pageSize, 100),
      sizeLimit: number(draft.sizeLimit, 1000),
      timeLimit: number(draft.timeLimit, 10),
    },
    aliases: draft.aliases,
    referrals: draft.referrals,
    schemaVersion: 1,
  };
}

/** What stops Finish from being pressable, in the order a user would fix it. */
export function draftProblems(draft: ConnectionDraft): string[] {
  const problems: string[] = [];
  if (!draft.host.trim()) problems.push('a hostname');
  if (!draft.port.trim()) problems.push('a port');
  if (draft.bindMethod !== 'anonymous' && !draft.bindDn.trim()) problems.push('a bind DN');
  return problems;
}
