/**
 * Session state: what the shell needs to render itself.
 *
 * Server data does not live here. Entries, pages and search results stay with
 * the views that fetched them, because a store that caches directory data is a
 * store that shows a stale entry after somebody else changed it (FR-105).
 */
import { create } from 'zustand';

import type { CapabilityUnavailable, ConnState, Job, MenuName } from '@/bridge/types';

/** The six perspectives on the activity rail. There is no seventh: local
 *  server management is out of scope for v1 (deviation D1). */
export type Perspective =
  'connections' | 'browser' | 'searches' | 'schema' | 'files' | 'preferences';

export type PanelTab = 'progress' | 'modifications' | 'searches' | 'errors' | 'console';

export type Theme = 'dark' | 'light' | 'high-contrast' | 'system';
export type Density = 'compact' | 'comfortable';

export interface DocumentTab {
  id: string;
  title: string;
  kind: 'entry' | 'search' | 'ldif' | 'schema' | 'preferences' | 'connections';
  profileId?: string;
  dn?: string;
  dirty?: boolean;
}

export interface Notice {
  id: string;
  level: 'info' | 'warning' | 'error';
  message: string;
  detail?: string;
  at: string;
}

interface SessionState {
  perspective: Perspective;
  setPerspective: (p: Perspective) => void;

  theme: Theme;
  density: Density;
  setTheme: (t: Theme) => void;
  setDensity: (d: Density) => void;

  /** Connection state per profile, as the status bar reads it. */
  connections: Record<string, ConnState>;
  activeProfileId?: string;
  setConnState: (state: ConnState) => void;
  setActiveProfile: (profileId?: string) => void;
  activeConnection: () => ConnState | undefined;

  jobs: Record<string, Job>;
  upsertJob: (job: Job) => void;
  runningJobs: () => Job[];

  tabs: DocumentTab[];
  activeTabId?: string;
  openTab: (tab: DocumentTab) => void;
  closeTab: (id: string) => void;
  activateTab: (id: string) => void;

  panelTab: PanelTab;
  panelOpen: boolean;
  setPanelTab: (tab: PanelTab) => void;
  togglePanel: () => void;

  notices: Notice[];
  addNotice: (notice: Omit<Notice, 'id' | 'at'>) => void;
  dismissNotice: (id: string) => void;
  capabilityNotices: CapabilityUnavailable[];
  addCapabilityNotice: (notice: CapabilityUnavailable) => void;

  openMenu?: MenuName;
  setOpenMenu: (menu?: MenuName) => void;
}

let noticeCounter = 0;

export const useSession = create<SessionState>((set, get) => ({
  perspective: 'connections',
  setPerspective: (perspective) => set({ perspective }),

  theme: 'dark',
  density: 'compact',
  setTheme: (theme) => set({ theme }),
  setDensity: (density) => set({ density }),

  connections: {},
  setConnState: (state) =>
    set((s) => ({
      connections: { ...s.connections, [state.profileId]: state },
      activeProfileId:
        state.state === 'connected' && !s.activeProfileId ? state.profileId : s.activeProfileId,
    })),
  setActiveProfile: (activeProfileId) => set({ activeProfileId }),
  activeConnection: () => {
    const { activeProfileId, connections } = get();
    return activeProfileId ? connections[activeProfileId] : undefined;
  },

  jobs: {},
  upsertJob: (job) => set((s) => ({ jobs: { ...s.jobs, [job.id]: job } })),
  runningJobs: () => Object.values(get().jobs).filter((j) => j.state === 'running'),

  tabs: [],
  openTab: (tab) =>
    set((s) => ({
      tabs: s.tabs.some((t) => t.id === tab.id) ? s.tabs : [...s.tabs, tab],
      activeTabId: tab.id,
    })),
  closeTab: (id) =>
    set((s) => {
      const tabs = s.tabs.filter((t) => t.id !== id);
      const activeTabId = s.activeTabId === id ? tabs[tabs.length - 1]?.id : s.activeTabId;
      return { tabs, activeTabId };
    }),
  activateTab: (activeTabId) => set({ activeTabId }),

  panelTab: 'progress',
  panelOpen: true,
  setPanelTab: (panelTab) => set({ panelTab, panelOpen: true }),
  togglePanel: () => set((s) => ({ panelOpen: !s.panelOpen })),

  notices: [],
  addNotice: (notice) =>
    set((s) => ({
      notices: [
        ...s.notices,
        { ...notice, id: `notice-${++noticeCounter}`, at: new Date().toISOString() },
      ],
    })),
  dismissNotice: (id) => set((s) => ({ notices: s.notices.filter((n) => n.id !== id) })),

  capabilityNotices: [],
  addCapabilityNotice: (notice) =>
    set((s) => ({
      capabilityNotices: s.capabilityNotices.some(
        (n) => n.profileId === notice.profileId && n.capability === notice.capability,
      )
        ? s.capabilityNotices
        : [...s.capabilityNotices, notice],
    })),

  openMenu: undefined,
  setOpenMenu: (openMenu) => set({ openMenu }),
}));
