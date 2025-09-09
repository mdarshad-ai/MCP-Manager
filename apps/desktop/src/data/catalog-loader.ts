import { CATALOG, type CatalogItem } from "./mcp-catalog";

function mergeCatalog(base: CatalogItem[], override: CatalogItem[]): CatalogItem[] {
  const map = new Map(base.map((i) => [i.slug, i] as const));
  for (const item of override) {
    const existing = map.get(item.slug);
    map.set(item.slug, { ...(existing || {} as CatalogItem), ...item });
  }
  return Array.from(map.values());
}

export async function loadCatalog(): Promise<CatalogItem[]> {
  const url = (import.meta as any)?.env?.VITE_MCP_CATALOG_URL as string | undefined;
  if (!url) return CATALOG;
  try {
    const r = await fetch(url, { cache: "no-store" });
    if (!r.ok) throw new Error(`HTTP ${r.status}`);
    const remote = (await r.json()) as CatalogItem[];
    if (!Array.isArray(remote)) return CATALOG;
    return mergeCatalog(CATALOG, remote);
  } catch {
    return CATALOG;
  }
}

