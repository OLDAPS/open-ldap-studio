import { Input } from '@/components/ui/input';
/**
 * Small pieces of the tree sidebar (screen 1b).
 *
 * The filter box filters the rows already fetched; it does not issue a search.
 * The placeholder says so, because a filter that silently queries the server
 * is a filter that can time out.
 */
export function TreeFilterBox({
  value,
  onChange,
}: {
  value: string;
  onChange: (next: string) => void;
}) {
  return (
    <Input
      className="field sidebar__filter"
      type="search"
      value={value}
      placeholder="filter loaded entries…"
      aria-label="Filter the directory tree"
      onChange={(event) => onChange(event.target.value)}
    />
  );
}
