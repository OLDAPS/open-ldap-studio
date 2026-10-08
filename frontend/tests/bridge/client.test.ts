import { afterEach, describe, expect, it, vi } from 'vitest';

import { bridge, BridgeUnavailableError, isEmbedded } from '@/bridge/client';
import type { AppInfo, Entry, Result } from '@/bridge/types';

function bind(methods: Record<string, unknown>) {
  Object.defineProperty(window, 'go', {
    configurable: true,
    value: { bridge: { Bridge: methods } },
  });
}

afterEach(() => {
  Object.defineProperty(window, 'go', { configurable: true, value: undefined });
});

describe('generated desktop bridge', () => {
  it('exposes Go application metadata through the generated wrapper', async () => {
    const info: AppInfo = {
      version: 'test',
      credentialStore: 'session only',
      credentialStoreReason: 'no platform agent',
      updateChecksEnabled: false,
      platform: 'linux',
    };
    const getAppInfo = vi.fn().mockResolvedValue(info);
    bind({ GetAppInfo: getAppInfo });

    expect(isEmbedded()).toBe(true);
    expect(await bridge.getAppInfo()).toBe(info);
    expect(getAppInfo).toHaveBeenCalledExactlyOnceWith();
  });

  it('reports unavailable Go methods when running outside the native window', async () => {
    expect(isEmbedded()).toBe(false);
    await expect(bridge.getAppInfo()).rejects.toBeInstanceOf(BridgeUnavailableError);
    bind({});
    await expect(bridge.listProfiles()).rejects.toThrow('Bridge.ListProfiles is not available');
  });

  it('serializes profiles and keeps a test secret in its separate argument', async () => {
    const profile = { name: 'test', host: 'localhost', credentialId: 'credential-1' };
    const saveProfile = vi.fn().mockResolvedValue(profile);
    const testConnection = vi.fn().mockResolvedValue({});
    bind({ SaveProfile: saveProfile, TestConnection: testConnection });

    await bridge.saveProfile(profile);
    await bridge.testConnection(profile, 'one-off secret');

    expect(saveProfile).toHaveBeenCalledWith(JSON.stringify(profile));
    expect(testConnection).toHaveBeenCalledWith(JSON.stringify(profile), 'one-off secret');
  });

  it('preserves multiple Go results and the server diagnostic verbatim', async () => {
    const entry: Entry = { dn: 'dc=example', attributes: [], hasChildren: 'unknown' };
    const result: Result = {
      code: 32,
      matchedDn: 'dc=example',
      diagnosticMessage: '  server diagnostic\n',
      interpretation: 'No such object',
    };
    const reply = [entry, result];
    const readEntry = vi.fn().mockResolvedValue(reply);
    bind({ ReadEntry: readEntry });

    expect(await bridge.readEntry('profile-1', 'dc=example', true)).toBe(reply);
    expect(readEntry).toHaveBeenCalledWith('profile-1', 'dc=example', {
      IncludeOperational: true,
      Attributes: [],
      ManageDsaIT: false,
    });
  });

  it('preserves backend failures', async () => {
    const error = new Error('profile store is newer than this application');
    bind({ ListProfiles: vi.fn().mockRejectedValue(error) });
    await expect(bridge.listProfiles()).rejects.toBe(error);
  });

  it('passes binary paging cookies and attribute values without text conversion', async () => {
    const entry: Entry = {
      dn: 'dc=example',
      hasChildren: 'unknown',
      attributes: [{ type: 'jpegPhoto', values: ['AP/+'], isOperational: false }],
    };
    const page = { entries: [entry], cookie: 'AP8=', loadedCount: 1 };
    const listChildren = vi.fn().mockResolvedValue(page);
    bind({ ListChildren: listChildren });
    const request = { size: 10, cookie: 'AP8=', includeOperational: false };

    expect(await bridge.listChildren('profile-1', 'dc=example', request)).toBe(page);
    expect(listChildren).toHaveBeenCalledWith('profile-1', 'dc=example', request);
  });
});
