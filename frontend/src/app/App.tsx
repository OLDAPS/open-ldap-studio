import { CommandProvider } from '@/app/CommandProvider';
import { TooltipProvider } from '@/components/ui/tooltip';
import { Shell } from '@/shell/Shell';
import '@/styles/global.css';
import '@/styles/shell.css';
import '@/styles/views.css';

export default function App() {
  return (
    <TooltipProvider>
      <CommandProvider>
        <Shell />
      </CommandProvider>
    </TooltipProvider>
  );
}
