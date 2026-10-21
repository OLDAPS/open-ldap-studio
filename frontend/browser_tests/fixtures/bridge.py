"""Browser-side Wails bridge, injected before the application's scripts run.

The init script runs in the page's JavaScript runtime; its seed, controls and
assertions are supplied by the Python fixtures and scenarios.
"""

INSTALL_BRIDGE = r"""
(seed) => {
  const subscribers = new Map();
  const ok = { code: 0, matchedDn: '', diagnosticMessage: '', interpretation: 'Success' };
  const state = {
    calls: [],
    failures: {},
    responses: {},
    seed,
    emit(event, payload) {
      subscribers.get(event)?.forEach((handler) => handler(structuredClone(payload)));
    },
  };
  window.__testBridge = state;

  const requireProfile = (id) => {
    const found = state.seed.profiles.find((profile) => profile.id === id);
    if (!found) throw new Error(`Unknown test profile: ${id}`);
    return found;
  };

  const handlers = {
    GetCommands: () => state.seed.commands,
    ListProfiles: () => state.seed.profiles,
    GetProfile: (id) => requireProfile(id),
    ConnectionStates: () => state.seed.connections,
    SaveProfile: (payload) => {
      const profile = JSON.parse(payload);
      if ('password' in profile || 'secret' in profile)
        throw new Error('A profile cannot contain a secret');
      profile.id ||= `profile-${state.seed.profiles.length + 1}`;
      state.seed.profiles = [...state.seed.profiles.filter((p) => p.id !== profile.id), profile];
      return profile;
    },
    StoreProfileSecret: (id, _secret) => {
      requireProfile(id);
    },
    DeleteProfile: (id) => {
      requireProfile(id);
      state.seed.profiles = state.seed.profiles.filter((profile) => profile.id !== id);
      state.seed.connections = state.seed.connections.filter(
        (connection) => connection.profileId !== id,
      );
    },
    TestConnection: (payload, _secret) => {
      const draft = JSON.parse(payload);
      return {
        reachable: true,
        encrypted: draft.encryption !== 'none',
        tlsVerified: true,
        bound: true,
        boundDn: draft.bindDn,
        namingContexts: state.seed.rootDSE.namingContexts,
        vendorName: state.seed.rootDSE.vendorName,
        result: ok,
      };
    },
    Connect: (id) => {
      const profile = requireProfile(id);
      const connection = {
        profileId: id,
        state: 'connected',
        boundDn: profile.bindDn,
        serverIdentity: `${profile.host}:${profile.port}`,
        encrypted: profile.encryption !== 'none',
        tlsVerified: profile.tls.verifyCertificate && profile.tls.verifyHostname,
        readOnly: profile.readOnly,
        production: profile.tags?.includes('production') ?? false,
        writesRequireConfirmation: false,
      };
      state.seed.connections = [
        ...state.seed.connections.filter((c) => c.profileId !== id),
        connection,
      ];
      state.emit('conn:state', connection);
      return `connect-${id}`;
    },
    Disconnect: (id) => {
      const connection = state.seed.connections.find((c) => c.profileId === id);
      if (!connection) throw new Error(`No test connection: ${id}`);
      connection.state = 'disconnected';
      state.emit('conn:state', connection);
    },
    RootDSE: (_id) => ({ rootDse: state.seed.rootDSE, result: ok }),
    ListChildren: (_id, dn, request) => {
      const pages = state.seed.children[dn];
      const page = pages?.[request.cookie ? 1 : 0];
      if (!page) throw new Error(`No test page for ${dn} (${request.cookie ?? 'first'})`);
      return page;
    },
    ReadEntry: (_id, dn, options) => {
      const entry = state.seed.entries[dn];
      if (!entry) throw new Error(`No test entry: ${dn}`);
      const filtered = {
        ...entry,
        attributes: entry.attributes.filter(
          (attribute) => options.IncludeOperational || !attribute.isOperational,
        ),
      };
      return { entry: filtered, result: ok };
    },
    Cancel: (jobId) =>
      state.emit('job:finished', { jobId, state: 'cancelled', summary: 'Cancelled by user' }),
  };

  const bridge = Object.fromEntries(
    Object.entries(handlers).map(([method, handler]) => [
      method,
      async (...args) => {
        state.calls.push({ method, args: structuredClone(args) });
        const failure = state.failures[method]?.shift();
        if (failure) throw new Error(failure);
        if (state.responses[method]?.length) return structuredClone(state.responses[method].shift());
        return structuredClone(await handler(...args));
      },
    ]),
  );

  Object.assign(window, {
    go: { bridge: { Bridge: bridge } },
    runtime: {
      EventsOn(event, handler) {
        const listeners = subscribers.get(event) ?? new Set();
        listeners.add(handler);
        subscribers.set(event, listeners);
        return () => {
          listeners.delete(handler);
        };
      },
      EventsOff: (event) => {
        subscribers.delete(event);
      },
    },
  });
}
"""
