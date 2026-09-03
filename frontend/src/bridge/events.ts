/**
 * The runtime event stream.
 *
 * Events are the mechanism behind Gate IV: nothing that takes time is a
 * blocking call, so progress, connection state, capability degradations and
 * security challenges all arrive here (contracts/events.md).
 *
 * No event ever carries secret material — a Secret has no marshaller on the Go
 * side, so a payload that tried to would fail to serialise rather than leak.
 */
import type { CapabilityUnavailable, ConnState, Job, Outcome, TrustChallenge } from './types';

type Handler<T> = (payload: T) => void;

interface WailsRuntime {
  EventsOn: (name: string, handler: (...data: unknown[]) => void) => () => void;
}

function runtime(): WailsRuntime | undefined {
  return (window as unknown as { runtime?: WailsRuntime }).runtime;
}

/** Subscribes to one event, returning an unsubscribe function. */
export function on<T>(name: string, handler: Handler<T>): () => void {
  const rt = runtime();
  if (!rt) {
    // Outside the desktop shell there is no event stream. Returning a no-op
    // keeps every caller's cleanup path identical.
    return () => {};
  }
  return rt.EventsOn(name, (...data: unknown[]) => handler(data[0] as T));
}

export interface JobStarted {
  jobId: string;
  kind: string;
  mode: 'execute' | 'dryRun';
  total?: number;
  profileId?: string;
}

export interface JobProgress {
  jobId: string;
  done: number;
  total: number;
  message: string;
}

export interface JobOutcomeEvent extends Outcome {
  jobId: string;
}

export interface JobFinished {
  jobId: string;
  state: Job['state'];
  summary: string;
  reportPath?: string;
  error?: string;
}

export interface EntryChanged {
  profileId: string;
  dn: string;
  source: 'commit' | 'refresh' | 'external';
}

export interface CredentialRequired {
  profileId: string;
  ref?: string;
  reason: string;
}

export const events = {
  jobStarted: (h: Handler<JobStarted>) => on('job:started', h),
  jobProgress: (h: Handler<JobProgress>) => on('job:progress', h),
  jobOutcome: (h: Handler<JobOutcomeEvent>) => on('job:outcome', h),
  jobFinished: (h: Handler<JobFinished>) => on('job:finished', h),

  connState: (h: Handler<ConnState>) => on('conn:state', h),
  connLost: (h: Handler<{ profileId: string; reason: string }>) => on('conn:lost', h),
  connReconnected: (h: Handler<{ profileId: string; writesRequireConfirmation: boolean }>) =>
    on('conn:reconnected', h),

  /** The connection is already refused when this fires (FR-007). */
  trustChallenge: (h: Handler<TrustChallenge>) => on('trust:challenge', h),
  credentialRequired: (h: Handler<CredentialRequired>) => on('credential:required', h),
  credentialStoreUnavailable: (h: Handler<{ reason: string }>) =>
    on('credential:storeUnavailable', h),

  entryChanged: (h: Handler<EntryChanged>) => on('entry:changed', h),
  /** Every degradation is announced with a reason (SC-016). */
  capabilityUnavailable: (h: Handler<CapabilityUnavailable>) => on('capability:unavailable', h),
};
