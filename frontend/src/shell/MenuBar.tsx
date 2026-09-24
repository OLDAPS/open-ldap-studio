/**
 * The menu bar: eight menus, rendered from the command registry.
 *
 * Eight, not nine. There is no Window menu, and File › New offers no server
 * instance — local server management is out of scope for v1 (deviations D1,
 * screens.md § Shell).
 */
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Fragment } from 'react';

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

  const renderItem = (item: MenuItem, key: string) => {
    if (item.items?.length) {
      return (
        <DropdownMenuSub key={key}>
          <DropdownMenuSubTrigger>{item.label}</DropdownMenuSubTrigger>
          <DropdownMenuSubContent>
            <DropdownMenuGroup>
              {item.items.map((child, index) => renderItem(child, `${key}-${index}`))}
            </DropdownMenuGroup>
          </DropdownMenuSubContent>
        </DropdownMenuSub>
      );
    }

    const enabled = item.commandId ? isCommandEnabled(item.commandId) : false;
    return (
      <DropdownMenuItem
        key={key}
        variant={item.destructive ? 'destructive' : 'default'}
        disabled={!enabled}
        onClick={() => {
          if (!item.commandId) return;
          setOpenMenu(undefined);
          run(item.commandId);
        }}
      >
        <span>{item.label}</span>
        {item.commandId && chordFor(item.commandId) ? (
          <DropdownMenuShortcut className="mono">{chordFor(item.commandId)}</DropdownMenuShortcut>
        ) : null}
      </DropdownMenuItem>
    );
  };

  return (
    <div className="menubar" role="menubar" aria-label="Application menu">
      {MENUS.map((name) => {
        const items = menu(name);
        const isOpen = openMenu === name;
        return (
          <DropdownMenu
            key={name}
            open={isOpen}
            onOpenChange={(open) => setOpenMenu(open ? name : undefined)}
          >
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="xs"
                  className="menubar__button"
                  onMouseEnter={() => openMenu && setOpenMenu(name)}
                />
              }
            >
              {name}
            </DropdownMenuTrigger>
            {items.length > 0 ? (
              <DropdownMenuContent className="menu__dropdown">
                <DropdownMenuGroup>
                  {items.map((item, index) => {
                    const previous = items[index - 1];
                    const separator = previous && previous.group !== item.group;
                    return (
                      <Fragment key={`${name}-${index}`}>
                        {separator ? <DropdownMenuSeparator /> : null}
                        {renderItem(item, `${name}-${index}`)}
                      </Fragment>
                    );
                  })}
                </DropdownMenuGroup>
              </DropdownMenuContent>
            ) : null}
          </DropdownMenu>
        );
      })}
    </div>
  );
}
