import type { ApplicationClient, ApplicationEventMap } from '@/application/client';
import { fakeApplicationClient } from './applicationClient';

/** An in-memory transport whose subscriptions have the same lifecycle as Wails events. */
export function eventClient(overrides: Partial<ApplicationClient> = {}) {
  const listeners = new Map<keyof ApplicationEventMap, Set<(event: unknown) => void>>();
  const client = fakeApplicationClient({
    isAvailable: () => true,
    ...overrides,
    subscribe: (name, handler) => {
      let handlers = listeners.get(name);
      if (!handlers) {
        handlers = new Set();
        listeners.set(name, handlers);
      }
      const listener = handler as (event: unknown) => void;
      handlers.add(listener);
      return () => {
        handlers?.delete(listener);
      };
    },
  });
  return {
    client,
    emit<Event extends keyof ApplicationEventMap>(name: Event, event: ApplicationEventMap[Event]) {
      listeners.get(name)?.forEach((listener) => listener(event));
    },
  };
}
