import { CommandProvider } from '@/app/CommandProvider';
import { Shell } from '@/shell/Shell';
import '@/styles/global.css';
import '@/styles/shell.css';
import '@/styles/views.css';

export default function App() {
  return (
    <CommandProvider>
      <Shell />
    </CommandProvider>
  );
}
