// Shared JSON Canvas 1.0 structural validator (Phase 4, ADR-0001).
//
// Extracted verbatim from app.js so the file-canonical vault layer
// (storage/workspace-vault.js) can validate .canvas documents at the storage
// boundary without importing the browser-only app module. Phase 4b switches
// app.js to import this, removing its inline copy. Keep the two in sync until
// then; the canonical copy lives here.

export function isCanvas(data) {
  if (!data || typeof data!=="object" || data.nodes&&!Array.isArray(data.nodes) || data.edges&&!Array.isArray(data.edges)) return false;
  const nodes=data.nodes||[],edges=data.edges||[],nodeIds=new Set(),itemIds=new Set(),sides=new Set(["top","right","bottom","left"]),ends=new Set(["none","arrow"]);
  const validColor=color=>color===undefined || typeof color==="string" && (/^[1-6]$/.test(color)||/^#[0-9a-f]{6}$/i.test(color));
  for(const node of nodes){
    if(!node||typeof node.id!=="string"||!node.id||itemIds.has(node.id)||!["text","file","link","group"].includes(node.type)||![node.x,node.y,node.width,node.height].every(Number.isInteger)||node.width<=0||node.height<=0||!validColor(node.color))return false;
    if(node.type==="text"&&typeof node.text!=="string"||node.type==="file"&&typeof node.file!=="string"||node.type==="link"&&typeof node.url!=="string"||node.type==="group"&&node.backgroundStyle!==undefined&&!["cover","ratio","repeat"].includes(node.backgroundStyle))return false;
    nodeIds.add(node.id);itemIds.add(node.id);
  }
  for(const edge of edges){
    if(!edge||typeof edge.id!=="string"||!edge.id||itemIds.has(edge.id)||typeof edge.fromNode!=="string"||typeof edge.toNode!=="string"||!nodeIds.has(edge.fromNode)||!nodeIds.has(edge.toNode)||edge.fromSide!==undefined&&!sides.has(edge.fromSide)||edge.toSide!==undefined&&!sides.has(edge.toSide)||edge.fromEnd!==undefined&&!ends.has(edge.fromEnd)||edge.toEnd!==undefined&&!ends.has(edge.toEnd)||!validColor(edge.color))return false;
    itemIds.add(edge.id);
  }
  return true;
}
