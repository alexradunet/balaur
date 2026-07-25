// Completeness audit for the disposable in-memory index.
import { buildSourceRecord, buildEntityProjection, extractCanvasPlacements, detectDuplicateIds } from "./life-indexer.js";
import { isCanvas } from "./canvas-validate.js";

const json = (v) => JSON.stringify(v);
function expectedRows(record, parsed) {
  if (!parsed || record.parseStatus === "error") return [];
  const p = buildEntityProjection(record, parsed);
  if (!p) return [];
  return p.kind === "entity" ? [{ type: p.entityType, row: p.row }] : p.rows.map((row) => ({ type: "habit-entry", row }));
}
function rowKey(type, row) {
  if (type === "journal") return `${type}:${row.localDate}`;
  if (type === "habit-entry") return `${type}:${row.id}`;
  return `${type}:${row.id}`;
}

export async function auditIndex(vault, index, { canvasIdFromPath = null } = {}) {
  const problems = [], vaultFiles = await vault.list(""), vaultByPath = new Map(vaultFiles.map((f) => [f.path, f]));
  const sources = index.allSourceFiles(), sourceByPath = new Map(sources.map((r) => [r.path, r]));
  for (const path of vaultByPath.keys()) if (!sourceByPath.has(path)) problems.push({ code: "UNINDEXED_FILE", path, message: `Vault file is not indexed: ${path}` });
  for (const path of sourceByPath.keys()) if (!vaultByPath.has(path)) problems.push({ code: "STALE_SOURCE", path, message: `Index source has no vault file: ${path}` });
  for (const [path, rec] of sourceByPath) { const vf = vaultByPath.get(path); if (vf && vf.hash !== rec.contentHash) problems.push({ code: "HASH_MISMATCH", path, message: `Index hash differs from vault: ${path}` }); }

  const parsedByPath = new Map(), expected = new Map();
  for (const meta of vaultFiles) {
    let content; try { content = await vault.read(meta.path); } catch (_) { continue; }
    const built = await buildSourceRecord(meta.path, content, { mediaType: meta.mediaType, modifiedAt: meta.modifiedAt });
    parsedByPath.set(meta.path, built);
    for (const item of expectedRows(built.record, built.parsed)) expected.set(rowKey(item.type, item.row), { ...item, path: meta.path });
  }
  const duplicateDiagnostics = detectDuplicateIds([...parsedByPath.values()].map((x) => x.record));
  for (const d of duplicateDiagnostics) problems.push({ code: d.errorCode, path: d.sourcePath, message: d.message, details: JSON.parse(d.detailsJson) });
  const actual = new Map();
  for (const [type, rows] of [["task", index.allTasks()], ["habit", index.allHabits()], ["journal", index.allJournals()], ["calendar-event", index.allEvents()], ["habit-entry", index.allHabitEntries()]]) for (const row of rows) actual.set(rowKey(type, row), { type, row });
  for (const [key, item] of expected) {
    const conflict = duplicateDiagnostics.some((d) => d.sourcePath === item.path && d.detailsJson && JSON.parse(d.detailsJson).balaurId === item.row.id);
    if (conflict) continue;
    const got = actual.get(key);
    if (!got) problems.push({ code: "MISSING_TYPED_ROW", path: item.path, message: `Missing typed row: ${key}` });
    else if (json(got.row) !== json(item.row)) problems.push({ code: "TYPED_ROW_MISMATCH", path: item.path, message: `Typed row differs from canonical file: ${key}` });
  }
  for (const [key, got] of actual) if (!expected.has(key)) problems.push({ code: "EXTRA_TYPED_ROW", path: got.row.sourcePath || null, message: `Extra typed row: ${key}` });

  const entityMap = new Map();
  for (const [path, built] of parsedByPath) if (built.record.parseStatus === "ok" && built.record.entityType && built.record.entityType !== "canvas" && built.record.entityId) entityMap.set(path, { entityId: built.record.entityId, entityType: built.record.entityType });
  const expectedPlacements = new Map();
  for (const meta of vaultFiles.filter((f) => f.path.endsWith(".canvas"))) {
    const built = parsedByPath.get(meta.path); if (!built?.parsed || !isCanvas(built.parsed.doc)) continue;
    const canvasId = canvasIdFromPath ? canvasIdFromPath(meta.path) : meta.path.split("/").pop().replace(/\.canvas$/, "");
    const { placements } = extractCanvasPlacements(canvasId, built.parsed.doc, entityMap);
    for (const p of placements) expectedPlacements.set(`${p.canvasId}\0${p.nodeId}`, p);
  }
  const actualPlacements = new Map(index.allPlacements().map((p) => [`${p.canvasId}\0${p.nodeId}`, p]));
  for (const [key, p] of expectedPlacements) if (!actualPlacements.has(key)) problems.push({ code: "MISSING_PLACEMENT", path: p.sourcePath, message: `Missing placement: ${key}` });
  const knownEntityIds = new Set([...parsedByPath.values()].filter((x) => x.record.parseStatus === "ok").map((x) => x.record.entityId).filter(Boolean));
  for (const [key, p] of actualPlacements) {
    if (!knownEntityIds.has(p.entityId)) problems.push({ code: "DANGLING_PLACEMENT", path: p.sourcePath, message: `Placement references a missing entity: ${p.entityId}` });
    else if (!expectedPlacements.has(key)) problems.push({ code: "EXTRA_PLACEMENT", path: p.sourcePath, message: `Extra placement: ${key}` });
  }
  for (const d of index.allDiagnostics()) problems.push({ code: "PARSE_DIAGNOSTIC", path: d.sourcePath, message: d.message, details: d });
  for (const d of parsedByPath.values()) if (d.record.parseStatus === "error") problems.push({ code: "PARSE_DIAGNOSTIC", path: d.record.path, message: d.record.parseError });

  return { ok: problems.length === 0, problems, counts: { files: vaultFiles.length, sources: sources.length, placements: index.allPlacements().length } };
}

export async function purgeAndRebuild(vault, index, indexer) { index.clearAll(); await indexer.rebuild(); return auditIndex(vault, index); }
