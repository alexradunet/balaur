// Index integrity audit and recovery (Phase 10, ADR-0001, plan §10/§21).
//
// The index is derived state: it must always be reconstructible from the
// canonical vault. auditIndex compares the vault against the index and reports
// inconsistencies; purgeAndRebuild discards the index and reconstructs it. Every
// problem auditIndex can find is therefore recoverable, which makes deleting
// index/cache storage a supported recovery action (plan §10 exit criteria).

// Compare the canonical vault against the derived index. Returns
// { ok, problems, counts }. Problem codes:
//   UNINDEXED_FILE     vault file with no index source row
//   STALE_SOURCE       index source row with no vault file
//   HASH_MISMATCH      index hash differs from vault (external edit not reindexed)
//   DUPLICATE_ID       one orbit-id claimed by more than one file
//   DANGLING_PLACEMENT placement references an entity the index does not know
export async function auditIndex(vault, index) {
  const problems = [];
  const vaultFiles = await vault.list("");
  const vaultByPath = new Map(vaultFiles.map((f) => [f.path, f]));
  const sources = index.allSourceFiles();
  const sourceByPath = new Map(sources.map((r) => [r.path, r]));

  for (const path of vaultByPath.keys()) {
    if (!sourceByPath.has(path)) problems.push({ code: "UNINDEXED_FILE", path, message: `Vault file is not indexed: ${path}` });
  }
  for (const path of sourceByPath.keys()) {
    if (!vaultByPath.has(path)) problems.push({ code: "STALE_SOURCE", path, message: `Index source has no vault file: ${path}` });
  }
  for (const [path, rec] of sourceByPath) {
    const vf = vaultByPath.get(path);
    if (vf && vf.hash && rec.contentHash && vf.hash !== rec.contentHash) {
      problems.push({ code: "HASH_MISMATCH", path, message: `Index hash differs from vault (external edit not reindexed): ${path}` });
    }
  }

  const byId = new Map();
  for (const rec of sources) {
    if (!rec.entityId) continue;
    if (!byId.has(rec.entityId)) byId.set(rec.entityId, []);
    byId.get(rec.entityId).push(rec.path);
  }
  for (const [id, paths] of byId) {
    if (paths.length > 1) problems.push({ code: "DUPLICATE_ID", path: paths[0], message: `Duplicate orbit-id "${id}"`, details: { orbitId: id, paths } });
  }

  const entityIds = new Set(sources.map((r) => r.entityId).filter(Boolean));
  for (const p of index.allPlacements()) {
    if (!entityIds.has(p.entityId)) problems.push({ code: "DANGLING_PLACEMENT", path: p.sourcePath, message: `Placement references a missing entity: ${p.entityId}` });
  }

  return {
    ok: problems.length === 0,
    problems,
    counts: { files: vaultFiles.length, sources: sources.length, placements: index.allPlacements().length },
  };
}

// Discard the index and reconstruct it from the vault (plan §10: explicit
// index/cache purge and rebuild). Returns the post-rebuild audit, which must be
// clean for a healthy vault.
export async function purgeAndRebuild(vault, index, indexer) {
  index.clearAll();
  await indexer.rebuild();
  return auditIndex(vault, index);
}
