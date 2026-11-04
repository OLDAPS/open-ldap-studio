/**
 * The application shell, built once and shared by every perspective.
 *
 * Its shape is the wireframe's: a frameless title bar carrying the menus, a
 * six-icon activity rail, a sidebar, a document area, the progress-and-logs
 * panel, and a status bar that is never hidden (screens.md § Shell).
 */

import { useShellViewModel } from '@/app/useShellViewModel';

import { ActivityRail } from './ActivityRail';
import { StatusBar } from './StatusBar';
import { TitleBar } from './TitleBar';
import { BrowserView } from '@/features/browser/components/BrowserView';
import { ConnectionsView } from '@/features/connections/components/ConnectionsView';
import { FilesView } from '@/features/files/components/FilesView';
import { PreferencesView } from '@/features/preferences/components/PreferencesView';
import { SchemaView } from '@/features/schema/components/SchemaView';
import { SearchesView } from '@/features/search/components/SearchesView';

export function Shell() {
  const { perspective } = useShellViewModel();

  return (
    <div className="shell">
      <TitleBar />

      <div className="shell__body">
        <ActivityRail />

        {perspective === 'connections' ? <ConnectionsView /> : null}
        {perspective === 'browser' ? <BrowserView /> : null}
        {perspective === 'searches' ? <SearchesView /> : null}
        {perspective === 'schema' ? <SchemaView /> : null}
        {perspective === 'files' ? <FilesView /> : null}
        {perspective === 'preferences' ? <PreferencesView /> : null}
      </div>

      <StatusBar />
    </div>
  );
}
