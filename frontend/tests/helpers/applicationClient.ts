import type { ApplicationClient } from '@/application/client';

const unexpected = async (): Promise<never> => {
  throw new Error('Unexpected application request');
};

/** A small in-memory presentation port, without Wails globals or module mocks. */
export function fakeApplicationClient(
  overrides: Partial<ApplicationClient> = {},
): ApplicationClient {
  return {
    isAvailable: () => false,
    getCommands: unexpected,
    listProfiles: unexpected,
    getProfile: unexpected,
    saveProfile: unexpected,
    deleteProfile: unexpected,
    storeProfileSecret: unexpected,
    testConnection: unexpected,
    connect: unexpected,
    disconnect: unexpected,
    connectionStates: unexpected,
    rootDSE: unexpected,
    listChildren: unexpected,
    readEntry: unexpected,
    cancel: unexpected,
    subscribe: () => () => {},
    ...overrides,
  };
}
