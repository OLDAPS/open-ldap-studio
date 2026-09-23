/**
 * The menu bar: eight menus, rendered from the command registry.
 *
 * Eight, not nine. There is no Window menu, and File › New offers no server
 * instance — local server management is out of scope for v1 (deviations D1,
 * screens.md § Shell).
 */
import { useEffect, useRef } from 'react';

import { useCommands } from '@/app/CommandProvider';
import type { MenuItem, MenuName } from '@/bridge/types';
import { useSession } from '@/app/session';

const MENUS: MenuName[] = [
  'File',
  'Edit',
  'Search',
  'LDAP',
  'Schema',
  'Credentials',
  'Preferences',
  'Help',
];

export function MenuBar() {
  const { menu, run, isCommandEnabled, chordFor } = useCommands();
  const openMenu = useSession((s) => s.openMenu);
  const setOpenMenu = useSession((s) => s.setOpenMenu);
  const barRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!openMenu) return;
    const close = (event: MouseEvent) => {
      if (!barRef.current?.contains(event.target as Node)) setOpenMenu(undefined);
    };
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpenMenu(undefined);
    };
    document.addEventListener('mousedown', close);
    document.addEventListener('keydown', escape);
    return () => {
      document.removeEventListener('mousedown', close);
      document.removeEventListener('keydown', escape);
    };
  }, [openMenu, setOpenMenu]);

  const renderItem = (item: MenuItem, key: string) => {
    if (item.items?.length) {
      return (
        <div className="menu__submenu" key={key}>
          <div className="menu__item menu__item--parent">
            <span>{item.label}</span>
            <span className="menu__arrow">›</span>
          </div>
          <div className="menu__dropdown menu__dropdown--nested">
            {item.items.map((child, index) => renderItem(child, `${key}-${index}`))}
          </div>
        </div>
      );
    }

    const enabled = item.commandId ? isCommandEnabled(item.commandId) : false;
    return (
      <button
        type="button"
        key={key}
        className="menu__item"
        data-destructive={item.destructive || undefined}
        disabled={!enabled}
        onClick={() => {
          if (!item.commandId) return;
          setOpenMenu(undefined);
          run(item.commandId);
        }}
      >
        <span>{item.label}</span>
        {item.commandId && chordFor(item.commandId) ? (
          <span className="menu__chord mono">{chordFor(item.commandId)}</span>
        ) : null}
      </button>
    );
  };

  return (
    <div className="menubar" ref={barRef} role="menubar" aria-label="Application menu">
      {MENUS.map((name) => {
        const items = menu(name);
        const isOpen = openMenu === name;
        return (
          <div className="menubar__menu" key={name}>
            <button
              type="button"
              className="menubar__button"
              data-open={isOpen || undefined}
              aria-haspopup="menu"
              aria-expanded={isOpen}
              onClick={() => setOpenMenu(isOpen ? undefined : name)}
              onMouseEnter={() => openMenu && setOpenMenu(name)}
            >
              {name}
            </button>
            {isOpen && items.length > 0 ? (
              <div className="menu__dropdown" role="menu">
                {items.map((item, index) => {
                  const previous = items[index - 1];
                  const separator = previous && previous.group !== item.group;
                  return (
                    <div key={`${name}-${index}`}>
                      {separator ? <div className="menu__separator" /> : null}
                      {renderItem(item, `${name}-${index}`)}
                    </div>
                  );
                })}
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
