import { useApplicationClient } from './ApplicationClientProvider';
import { useSession } from './session';

export function useJobsViewModel() {
  const client = useApplicationClient();
  const jobs = useSession((state) => state.jobs);
  const addNotice = useSession((state) => state.addNotice);
  const cancelJob = async (id: string) => {
    try {
      await client.cancel(id);
    } catch (cause) {
      addNotice({
        level: 'error',
        message: 'Could not cancel the operation',
        detail: cause instanceof Error ? cause.message : String(cause),
      });
    }
  };
  return { jobs: Object.values(jobs), cancelJob };
}
