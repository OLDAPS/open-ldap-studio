/**
 * One registry, three surfaces.
 *
 * The menu bar, the context menus and the keyboard all read from the command
 * set the Go side publishes. A command that is invocable one way is invocable
 * every way, which is what FR-102's "fully keyboard operable" actually
 * requires (design gap G4).
 */
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

import { bridge, isEmbedded } from '@/bridge/client';
import type { Command, CommandSet, MenuName, Platform } from '@/bridge/types';

/** The context a command's enablement predicate is evaluated against. */
export interface EnablementContext {
  connectionOpen: boolean;
  connectionReadOnly: boolean;
  profileSelected: boolean;
  entrySelected: boolean;
  editorOpen: boolean;
  editorDirty: boolean;
  canUndo: boolean;
  canRedo: boolean;
  hasResults: boolean;
  schemaProjectOpen: boolean;
  historySelectedIsReversible: boolean;
}

export const emptyEnablement: EnablementContext = {
  connectionOpen: false,
  connectionReadOnly: false,
  profileSelected: false,
  entrySelected: false,
  editorOpen: false,
  editorDirty: false,
  canUndo: false,
  canRedo: false,
  hasResults: false,
  schemaProjectOpen: false,
  historySelectedIsReversible: false,
};

/**
 * Evaluates a predicate such as `connection.open && !connection.readOnly`.
 *
 * The grammar is deliberately tiny — identifiers, `!`, `&&` — so it can be
 * evaluated without a parser and without `eval`. A predicate that needs more
 * than this is a sign the command should carry its own condition.
 */
export function isEnabled(predicate: string, ctx: EnablementContext): boolean {
  if (!predicate || predicate === 'always') return true;

  const lookup: Record<string, boolean> = {
    'connection.open': ctx.connectionOpen,
    'connection.closed': !ctx.connectionOpen,
    'connection.readOnly': ctx.connectionReadOnly,
    'profile.selected': ctx.profileSelected,
    'entry.selected': ctx.entrySelected,
    'editor.open': ctx.editorOpen,
    'editor.dirty': ctx.editorDirty,
    'editor.canUndo': ctx.canUndo,
    'editor.canRedo': ctx.canRedo,
    'search.hasResults': ctx.hasResults,
    'schemaProject.open': ctx.schemaProjectOpen,
    'history.selectedIsReversible': ctx.historySelectedIsReversible,
  };

  return predicate.split('&&').every((term) => {
    const trimmed = term.trim();
    const negated = trimmed.startsWith('!');
    const key = negated ? trimmed.slice(1).trim() : trimmed;
    const value = lookup[key];
    if (value === undefined) {
      // An unknown predicate disables the command rather than enabling it: a
      // command that fires when nobody knows whether it should is worse than
      // one that is greyed out.
      return false;
    }
    return negated ? !value : value;
  });
}

type Handler = () => void;

interface CommandContextValue {
  commandSet?: CommandSet;
  platform: Platform;
  commands: Command[];
  menu: (name: MenuName) => CommandSet['tree'][MenuName];
  chordFor: (commandId: string) => string | undefined;
  register: (commandId: string, handler: Handler) => () => void;
  run: (commandId: string) => void;
  enablement: EnablementContext;
  setEnablement: (partial: Partial<EnablementContext>) => void;
  isCommandEnabled: (commandId: string) => boolean;
}

const CommandContext = createContext<CommandContextValue | undefined>(undefined);

/** Normalises a keyboard event into the chord form the registry uses. */
export function chordFromEvent(event: KeyboardEvent, platform: Platform): string {
  const parts: string[] = [];
  if (event.ctrlKey) parts.push('Ctrl');
  if (event.metaKey) parts.push('Cmd');
  if (event.altKey) parts.push('Alt');
  if (event.shiftKey) parts.push('Shift');

  let key = event.key;
  if (key === ' ') key = 'Space';
  if (key.length === 1) key = key.toUpperCase();
  if (['Control', 'Meta', 'Alt', 'Shift'].includes(key)) return '';

  parts.push(key);
  void platform;
  return parts.join('+');
}

export function CommandProvider({ children }: { children: ReactNode }) {
  const [commandSet, setCommandSet] = useState<CommandSet>();
  const [enablement, setEnablementState] = useState<EnablementContext>(emptyEnablement);
  const handlers = useMemo(() => new Map<string, Handler>(), []);

  useEffect(() => {
    if (!isEmbedded()) return;
    bridge
      .getCommands()
      .then(setCommandSet)
      .catch(() => setCommandSet(undefined));
  }, []);

  const register = useCallback(
    (commandId: string, handler: Handler) => {
      handlers.set(commandId, handler);
      return () => {
        handlers.delete(commandId);
      };
    },
    [handlers],
  );

  const run = useCallback(
    (commandId: string) => {
      handlers.get(commandId)?.();
    },
    [handlers],
  );

  /*
   * Stable identity, and a no-op when nothing actually changed.
   *
   * Shell recomputes enablement in an effect that depends on this function. If
   * it were rebuilt with the context value — which changes whenever enablement
   * does — the effect would re-run, set a fresh object, and rebuild the value
   * again: a render loop that never settles.
   */
  const setEnablement = useCallback((partial: Partial<EnablementContext>) => {
    setEnablementState((current) => {
      const next = { ...current, ...partial };
      const unchanged = (Object.keys(next) as (keyof EnablementContext)[]).every(
        (key) => next[key] === current[key],
      );
      return unchanged ? current : next;
    });
  }, []);

  const isCommandEnabled = useCallback(
    (commandId: string) => {
      const command = commandSet?.commands.find((c) => c.id === commandId);
      return command ? isEnabled(command.enablement, enablement) : false;
    },
    [commandSet, enablement],
  );

  // The keyboard layer. A chord fires its command only when the command is
  // enabled, so a shortcut and a menu item are never out of step.
  useEffect(() => {
    if (!commandSet) return;

    const byChord = new Map<string, string>();
    for (const [commandId, chord] of Object.entries(commandSet.bindings)) {
      byChord.set(chord, commandId);
    }

    const onKeyDown = (event: KeyboardEvent) => {
      const chord = chordFromEvent(event, commandSet.platform);
      const commandId = chord && byChord.get(chord);
      if (!commandId) return;

      const command = commandSet.commands.find((c) => c.id === commandId);
      if (!command || !isEnabled(command.enablement, enablement)) return;
      if (!handlers.has(commandId)) return;

      event.preventDefault();
      handlers.get(commandId)?.();
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [commandSet, enablement, handlers]);

  const value = useMemo<CommandContextValue>(
    () => ({
      commandSet,
      platform: commandSet?.platform ?? 'linux',
      commands: commandSet?.commands ?? [],
      menu: (name) => commandSet?.tree?.[name] ?? [],
      chordFor: (commandId) => commandSet?.bindings?.[commandId],
      register,
      run,
      enablement,
      setEnablement,
      isCommandEnabled,
    }),
    [commandSet, enablement, register, run, setEnablement, isCommandEnabled],
  );

  return <CommandContext.Provider value={value}>{children}</CommandContext.Provider>;
}

export function useCommands(): CommandContextValue {
  const ctx = useContext(CommandContext);
  if (!ctx) {
    throw new Error('useCommands must be used inside a CommandProvider');
  }
  return ctx;
}
