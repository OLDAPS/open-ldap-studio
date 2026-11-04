import type {
  CapabilityUnavailable,
  CommandSet,
  ConnState,
  Entry,
  Page,
  PageRequest,
  Profile,
  ProfileSummary,
  Result,
  RootDSE,
  TestResult,
} from '@/bridge/types';
import type { JobStarted, JobProgress, JobFinished, EntryChanged } from '@/bridge/events';

export interface ApplicationEventMap {
  'conn:state': ConnState;
  'conn:lost': { profileId: string; reason: string };
  'job:started': JobStarted;
  'job:progress': JobProgress;
  'job:finished': JobFinished;
  'entry:changed': EntryChanged;
  'capability:unavailable': CapabilityUnavailable;
  'credential:storeUnavailable': { reason: string };
}

/** The presentation boundary; implementations own transport lookup and decoding. */
export interface ApplicationClient {
  isAvailable(): boolean;
  getCommands(): Promise<CommandSet>;
  listProfiles(): Promise<ProfileSummary[]>;
  getProfile(id: string): Promise<Profile>;
  saveProfile(profile: Partial<Profile>): Promise<Profile>;
  deleteProfile(id: string): Promise<void>;
  storeProfileSecret(id: string, secret: string): Promise<void>;
  testConnection(profile: Partial<Profile>, secret: string): Promise<TestResult>;
  connect(id: string): Promise<string>;
  disconnect(id: string): Promise<void>;
  connectionStates(): Promise<ConnState[]>;
  rootDSE(id: string): Promise<[RootDSE, Result]>;
  listChildren(id: string, dn: string, page: PageRequest): Promise<Page>;
  readEntry(id: string, dn: string, includeOperational?: boolean): Promise<[Entry, Result]>;
  cancel(id: string): Promise<void>;
  subscribe<Event extends keyof ApplicationEventMap>(
    name: Event,
    handler: (event: ApplicationEventMap[Event]) => void,
  ): () => void;
}
