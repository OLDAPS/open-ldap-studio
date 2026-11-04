import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, it } from 'vitest';

it.each([
  'features/connections/components/ConnectionsView.tsx',
  'features/connections/components/ConnectionWizard.tsx',
  'features/browser/useDitTreeViewModel.ts',
  'features/browser/useEntryViewModel.ts',
  'features/browser/components/EntryView.tsx',
  'features/browser/components/BrowserView.tsx',
  'features/browser/components/DitTree.tsx',
  'shell/Shell.tsx',
  'shell/BottomPanel.tsx',
  'app/useShellViewModel.ts',
  'app/useJobsViewModel.ts',
  'app/CommandProvider.tsx',
])('%s stays behind the application-client and ViewModel boundary', (file) => {
  const source = readFileSync(
    resolve(dirname(fileURLToPath(import.meta.url)), '../../src', file),
    'utf8',
  );
  expect(source).not.toMatch(/from ['"]@\/bridge\/(client|events)['"]/);
  expect(source).not.toMatch(/window\.(go|runtime)/);
});
