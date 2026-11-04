import { useCallback, useEffect, useRef, useState } from 'react';
import { useApplicationClient } from '@/app/ApplicationClientProvider';
import type { Encryption, TestResult } from '@/bridge/types';
import { DEFAULT_PORT, draftProblems, emptyDraft, toProfile } from './model';
import type { ConnectionDraft } from './model';

export interface WizardCallbacks {
  onClose(): void;
  onSaved?(profileId: string): void;
}

export function useConnectionWizardViewModel({ onClose, onSaved }: WizardCallbacks) {
  const client = useApplicationClient();
  const [draft, setDraft] = useState<ConnectionDraft>(emptyDraft);
  const [test, setTest] = useState<TestResult>();
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const generation = useRef(0);
  const portEdited = useRef(false);
  useEffect(
    () => () => {
      generation.current += 1;
    },
    [],
  );

  const patch = useCallback((changes: Partial<ConnectionDraft>) => {
    generation.current += 1;
    setTest(undefined);
    setTesting(false);
    setError(undefined);
    setDraft((current) => ({ ...current, ...changes }));
  }, []);

  const chooseEncryption = (encryption: Encryption) => {
    patch(portEdited.current ? { encryption } : { encryption, port: DEFAULT_PORT[encryption] });
  };
  const changePort = (port: string) => {
    portEdited.current = true;
    patch({ port });
  };

  const runTest = async (withBind: boolean) => {
    const mine = ++generation.current;
    setError(undefined);
    setTesting(true);
    setTest(undefined);
    try {
      const profile = toProfile(draft);
      const response = await client.testConnection(
        withBind ? profile : { ...profile, bindMethod: 'anonymous', bindDn: '' },
        withBind ? draft.secret : '',
      );
      if (generation.current !== mine) return;
      setTest(response);
      const firstContext = response.namingContexts?.[0];
      if (response.reachable && !draft.baseDn && firstContext) {
        setDraft((current) => ({ ...current, baseDn: firstContext }));
      }
    } catch (cause) {
      if (generation.current === mine)
        setError(String(cause instanceof Error ? cause.message : cause));
    } finally {
      if (generation.current === mine) setTesting(false);
    }
  };

  const finish = async () => {
    generation.current += 1;
    setTesting(false);
    setError(undefined);
    setSaving(true);
    try {
      const saved = await client.saveProfile(toProfile(draft));
      // Creation has already succeeded. Keep its identity even if the
      // credential store fails so Finish retries update the same profile.
      setDraft((current) => ({ ...current, id: saved.id }));
      if (draft.bindMethod !== 'anonymous' && draft.secret) {
        await client.storeProfileSecret(saved.id, draft.secret);
      }
      setDraft((current) => ({ ...current, secret: '' }));
      onSaved?.(saved.id);
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setSaving(false);
    }
  };

  const problems = draftProblems(draft);
  return {
    draft,
    test,
    testing,
    saving,
    error,
    problems,
    patch,
    chooseEncryption,
    changePort,
    runTest,
    finish,
    available: client.isAvailable(),
    canFinish: problems.length === 0 && !saving && client.isAvailable(),
  };
}

export type ConnectionWizardViewModel = ReturnType<typeof useConnectionWizardViewModel>;
