import { createContext, useContext } from 'react';
import type { ReactNode } from 'react';
import type { ApplicationClient } from '@/application/client';
import { applicationClient } from '@/bridge/client';

// Only this composition boundary chooses the desktop adapter. ViewModels and
// component tests can instead supply an in-memory ApplicationClient.
const ClientContext = createContext<ApplicationClient>(applicationClient);

export function ApplicationClientProvider({
  client,
  children,
}: {
  client: ApplicationClient;
  children: ReactNode;
}) {
  return <ClientContext.Provider value={client}>{children}</ClientContext.Provider>;
}

export function useApplicationClient(): ApplicationClient {
  return useContext(ClientContext);
}
