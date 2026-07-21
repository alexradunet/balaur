/* Orbit is dependency-free. Every canvas document follows JSON Canvas 1.0; hierarchy and cameras live in a local workspace sidecar. */
const COLORS = {
  "1": "#ff7b78", "2": "#efa66a", "3": "#e9d56b",
  "4": "#7ee0a1", "5": "#64cbd0", "6": "#a78bfa"
};

const demoCanvas = {
  nodes: [
    { id:"g-week", type:"group", x:0, y:0, width:690, height:600, label:"This week", color:"4" },
    { id:"g-horizon", type:"group", x:750, y:0, width:590, height:810, label:"On the horizon", color:"6" },
    { id:"n-focus", type:"text", x:35, y:48, width:620, height:145, color:"3", text:"# A calmer, more intentional week\nProtect the mornings, move the important work forward, and leave enough space to notice life.\n\n`WEEK 29`  ·  **3 priorities**" },
    { id:"n-project", type:"text", x:35, y:230, width:295, height:240, color:"6", text:"# Ship the portfolio refresh\nMake the work feel as considered as the work itself.\n\n- [x] Finalize case study copy\n- [ ] Record walkthrough\n- [ ] Publish and share\n\nProgress: 66%" },
    { id:"n-habit", type:"text", x:360, y:230, width:295, height:150, color:"4", text:"# Morning pages\nWrite three pages before opening any inputs.\n\n**5 day streak**  ·  07:00" },
    { id:"n-idea", type:"text", x:360, y:410, width:295, height:145, color:"2", text:"# Sunday without screens\nA small experiment: books, a long walk, cooking, and no glowing rectangles until evening." },
    { id:"n-goal", type:"text", x:785, y:48, width:520, height:190, color:"1", text:"# Run a comfortable 10K\nBuild patiently. Finish feeling like there was a little more in the tank.\n\n- [x] Choose a training plan\n- [ ] Three easy runs / week\n- [ ] Race day · Sep 14\n\nProgress: 35%" },
    { id:"n-reading", type:"text", x:785, y:280, width:250, height:160, color:"5", text:"# Reading next\n- [ ] Four Thousand Weeks\n- [ ] The Creative Act\n- [ ] Braiding Sweetgrass" },
    { id:"n-trip", type:"link", x:1065, y:280, width:240, height:160, color:"6", url:"https://www.openstreetmap.org" },
    { id:"n-orbit", type:"file", x:785, y:475, width:520, height:300, color:"5", file:"widgets/focus-orbit.html" }
  ],
  edges: [
    { id:"e-focus-project", fromNode:"n-focus", fromSide:"bottom", toNode:"n-project", toSide:"top", color:"6", label:"focus" },
    { id:"e-focus-habit", fromNode:"n-focus", fromSide:"bottom", toNode:"n-habit", toSide:"top", color:"4" },
    { id:"e-habit-goal", fromNode:"n-habit", fromSide:"right", toNode:"n-goal", toSide:"left", color:"1", label:"supports" },
    { id:"e-idea-trip", fromNode:"n-idea", fromSide:"right", toNode:"n-trip", toSide:"left", color:"2", toEnd:"arrow" }
  ]
};

const $ = (selector, root=document) => root.querySelector(selector);
const $$ = (selector, root=document) => [...root.querySelectorAll(selector)];
const clone = value => JSON.parse(JSON.stringify(value));
const uid = prefix => `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2,7)}`;

const WORKSPACE_KEY="orbit-workspace-v1",ROOT_CANVAS_ID="canvas-root";
let workspace=loadWorkspace();
let currentCanvasId=workspace.activeId;
let documentData=workspace.canvases[currentCanvasId].document;
let camera=workspace.canvases[currentCanvasId].camera||{x:80,y:55,zoom:.78};
let selected = null;
let currentTool = "select";
let connectSource = null;
let connectSourceSide = null;
let activeFilter = "all";
let spaceDown = false;
let saveTimer;
const aiCardRuntime=new Map();

const canvas = $("#canvas");
const world = $("#world");
const nodeLayer = $("#nodes");
const edgeLayer = $("#edges");
const shell = $(".app-shell");

function loadDocument() {
  try {
    const saved = localStorage.getItem("orbit-canvas-v1");
    if (saved) {
      const parsed = JSON.parse(saved);
      if (isCanvas(parsed)) return parsed;
    }
  } catch (_) {}
  return clone(demoCanvas);
}
function freshWorkspace(document=loadDocument()){
  return {version:1,rootId:ROOT_CANVAS_ID,activeId:ROOT_CANVAS_ID,canvases:{[ROOT_CANVAS_ID]:{id:ROOT_CANVAS_ID,title:localStorage.getItem("orbit-title")||"Life OS — Summer",parentId:null,portalNodeId:null,document,camera:{x:80,y:55,zoom:.78}}}};
}
function loadWorkspace(){
  try{
    const parsed=JSON.parse(localStorage.getItem(WORKSPACE_KEY)||"null"),canvases=parsed?.canvases;
    if(parsed?.version===1&&canvases&&typeof canvases==="object"&&Object.keys(canvases).length&&Object.values(canvases).every(record=>record&&typeof record.id==="string"&&typeof record.title==="string"&&isCanvas(record.document))){
      parsed.rootId=canvases[parsed.rootId]?parsed.rootId:Object.keys(canvases)[0];parsed.activeId=canvases[parsed.activeId]?parsed.activeId:parsed.rootId;return parsed;
    }
  }catch(_){}
  return freshWorkspace();
}

function isCanvas(data) {
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

function saveCurrentCanvasState(){
  const record=workspace.canvases[currentCanvasId];if(!record)return;record.document=documentData;record.camera={...camera};const title=$("#canvasTitle")?.value.trim();if(title)record.title=title;
}
function persistWorkspace(){
  saveCurrentCanvasState();workspace.activeId=currentCanvasId;localStorage.setItem(WORKSPACE_KEY,JSON.stringify(workspace));localStorage.setItem("orbit-canvas-v1",JSON.stringify(workspace.canvases[workspace.rootId].document));localStorage.setItem("orbit-title",workspace.canvases[workspace.rootId].title);
}
function scheduleSave() {
  $("#saveState").innerHTML = "<i></i> Saving…";
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    persistWorkspace();
    $("#saveState").innerHTML = "<i></i> Saved locally";
  }, 350);
}

function colorValue(color) { return COLORS[color] || color || "#737b87"; }
function escapeHTML(value="") {
  return String(value).replace(/[&<>'"]/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"}[c]));
}
function safeURL(value="") {
  try { const url = new URL(value, location.href); return ["http:","https:","mailto:"].includes(url.protocol) ? escapeHTML(value) : "#"; }
  catch (_) { return "#"; }
}
function safeFileURL(value="") {
  const path=String(value).replace(/\\/g,"/");
  if (!path || path.startsWith("/") || path.includes("..") || /^[a-z][a-z0-9+.-]*:/i.test(path)) return "about:blank";
  return encodeURI(path).replace(/#/g,"%23");
}
function subcanvasIdFromNode(node){
  if(node?.type!=="file")return null;const match=node.file.match(/^canvases\/([^/]+)\.canvas$/);return match&&workspace.canvases[match[1]]?match[1]:null;
}
function canvasRecord(id=currentCanvasId){return workspace.canvases[id];}
function canvasDepth(id){
  let depth=0,seen=new Set();while(workspace.canvases[id]?.parentId&&!seen.has(id)){seen.add(id);id=workspace.canvases[id].parentId;depth++;}return depth;
}
function canvasTrail(id=currentCanvasId){
  const trail=[],seen=new Set();while(workspace.canvases[id]&&!seen.has(id)){seen.add(id);trail.unshift(workspace.canvases[id]);id=workspace.canvases[id].parentId;}return trail;
}
function portalPreview(document){
  const nodes=(document.nodes||[]).slice(0,28);if(!nodes.length)return '<span class="portal-empty">Empty canvas · open to begin</span>';
  const minX=Math.min(...nodes.map(node=>node.x)),minY=Math.min(...nodes.map(node=>node.y)),maxX=Math.max(...nodes.map(node=>node.x+node.width)),maxY=Math.max(...nodes.map(node=>node.y+node.height)),width=Math.max(1,maxX-minX),height=Math.max(1,maxY-minY),scale=Math.min(210/width,82/height);
  return nodes.map(node=>`<i class="${node.type==="group"?"group":""}" style="left:${(node.x-minX)*scale}px;top:${(node.y-minY)*scale}px;width:${Math.max(2,node.width*scale)}px;height:${Math.max(2,node.height*scale)}px;${node.type==="group"?"":`background:${colorValue(node.color)}`}" ></i>`).join("");
}
function renderWorkspaceNavigation(){
  const breadcrumbs=$("#canvasBreadcrumbs"),list=$("#canvasList");
  if(breadcrumbs)breadcrumbs.innerHTML=canvasTrail().map((record,index,trail)=>`<button data-canvas-switch="${escapeHTML(record.id)}" ${index===trail.length-1?'aria-current="page"':''}>${escapeHTML(record.title)}</button>${index<trail.length-1?"<span>›</span>":""}`).join("");
  if(list)list.innerHTML=Object.values(workspace.canvases).map(record=>`<button class="nav-item canvas-list-item ${record.id===currentCanvasId?"active":""}" data-canvas-switch="${escapeHTML(record.id)}" style="--canvas-depth:${canvasDepth(record.id)}"><span>${record.id===workspace.rootId?"◫":"↳"}</span><b>${escapeHTML(record.title)}</b><em>${(record.document.nodes||[]).length}</em></button>`).join("");
  $$('[data-canvas-switch]').forEach(button=>button.onclick=()=>switchCanvas(button.dataset.canvasSwitch,{direction:"switch"}));
}
function activateCanvas(id,{focusNodeId=null,fit=false}={}){
  const record=workspace.canvases[id];if(!record)return;currentCanvasId=id;workspace.activeId=id;documentData=record.document;camera=record.camera?{...record.camera}:{x:80,y:55,zoom:1};selected=null;connectSource=null;connectSourceSide=null;activeFilter="all";aiCardRuntime.clear();shell.classList.remove("inspector-open");$$('.nav-item[data-filter]').forEach(button=>button.classList.toggle("active",button.dataset.filter==="all"));$("#canvasTitle").value=record.title;render();
  if(focusNodeId){const node=documentData.nodes.find(item=>item.id===focusNodeId);if(node)focusNode(node,1.05);else fitView();}
  else if(fit||!record.camera)fitView();
}
function switchCanvas(id,{direction="in",focusNodeId=null,fit=false}={}){
  if(!workspace.canvases[id]||id===currentCanvasId)return;saveCurrentCanvasState();document.body.dataset.canvasNavigation=direction;const update=()=>activateCanvas(id,{focusNodeId,fit}),transition=document.startViewTransition?.(update);if(!transition)update();else transition.finished.finally(()=>delete document.body.dataset.canvasNavigation);scheduleSave();
}
function enterSubcanvas(id){if(workspace.canvases[id])switchCanvas(id,{direction:"in",fit:!workspace.canvases[id].camera});}
function leaveSubcanvas(){
  const child=canvasRecord(),parentId=child?.parentId;if(!parentId)return;switchCanvas(parentId,{direction:"out",focusNodeId:child.portalNodeId});
}
function focusNode(node,zoom=1.05){camera.zoom=zoom;camera.x=(canvas.clientWidth-node.width*zoom)/2-node.x*zoom;camera.y=(canvas.clientHeight-node.height*zoom)/2-node.y*zoom;applyCamera();}
function createSubcanvas(point){
  const center=point||canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),id=uid("canvas"),nodeId=uid("node"),siblings=Object.values(workspace.canvases).filter(record=>record.parentId===currentCanvasId).length,title=`New canvas ${siblings+1}`,node={id:nodeId,type:"file",x:Math.round(center.x-180),y:Math.round(center.y-125),width:360,height:250,color:"3",file:`canvases/${id}.canvas`};
  workspace.canvases[id]={id,title,parentId:currentCanvasId,portalNodeId:nodeId,document:{nodes:[],edges:[]},camera:null};documentData.nodes.push(node);selected={kind:"node",id:node.id};shell.classList.add("inspector-open");scheduleSave();render();toast("Sub-canvas created · double-click or zoom into it");return node;
}
function deleteCanvasTree(id){for(const child of Object.values(workspace.canvases).filter(record=>record.parentId===id))deleteCanvasTree(child.id);delete workspace.canvases[id];}

const AI_CARD_MARKER="<!-- orbit:ai-card -->";
function isAICard(node){return node?.type==="text"&&node.text.includes(AI_CARD_MARKER);}
function parseAICard(node){
  const lines=(node.text||"").split(/\r?\n/).filter(line=>line.trim()!==AI_CARD_MARKER),heading=lines.findIndex(line=>line.startsWith("# ")),title=heading>=0?lines[heading].slice(2).trim():"AI operator";
  if(heading>=0)lines.splice(heading,1);return {title,prompt:lines.join("\n").trim()||"Summarize the connected notes."};
}
function buildAICardText(title,prompt){return `${AI_CARD_MARKER}\n# ${title.trim()||"AI operator"}\n${prompt.trim()||"Summarize the connected notes."}`;}
function nodeTitle(node){
  if(isAICard(node))return parseAICard(node).title;if(node.type==="text"){const heading=node.text.match(/^#{1,2}\s+(.+)$/m);return heading?heading[1]:"Text note";}if(node.type==="group")return node.label||"Group";if(node.type==="link")try{return new URL(node.url).hostname;}catch(_){return "Link";}if(node.type==="file"){const subcanvasId=subcanvasIdFromNode(node);return subcanvasId?workspace.canvases[subcanvasId].title:node.file.split("/").pop();}return node.id;
}
function inputNodesForAICard(cardId,data=documentData){const byId=Object.fromEntries((data.nodes||[]).map(node=>[node.id,node]));return (data.edges||[]).filter(edge=>edge.toNode===cardId&&edge.label!=="AI output").map(edge=>byId[edge.fromNode]).filter(Boolean);}
function nodeAIContent(node){if(node.type==="text")return node.text;if(node.type==="link")return node.url;if(node.type==="file")return [node.file,node.subpath].filter(Boolean).join(" ");if(node.type==="group")return node.label||"";return "";}
function aiCardSignature(card,data=documentData){return JSON.stringify([card.text,inputNodesForAICard(card.id,data).map(node=>[node.id,nodeAIContent(node)])]);}
function aiCardSignatures(data=documentData){return new Map((data.nodes||[]).filter(isAICard).map(card=>[card.id,aiCardSignature(card,data)]));}

function textMeta(node) {
  const map = {"1":"GOAL", "2":"IDEA", "3":"NOTE", "4":"HABIT", "5":"RESOURCE", "6":"PROJECT"};
  return map[node.color] || node.type.toUpperCase();
}

function markdownToHTML(source="") {
  const lines = source.split(/\r?\n/);
  let html = "", inList = false;
  const inline = raw => escapeHTML(raw)
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/`(.+?)`/g, "<code>$1</code>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a class="node-link" href="$2" target="_blank" rel="noreferrer">$1 ↗</a>');
  for (const line of lines) {
    const task = line.match(/^\s*- \[([ xX])\]\s+(.+)/);
    const bullet = line.match(/^\s*-\s+(.+)/);
    if (task || bullet) {
      if (!inList) { html += "<ul>"; inList = true; }
      html += `<li class="${task && task[1].toLowerCase()==="x" ? "checked" : ""}">${inline(task ? task[2] : bullet[1])}</li>`;
      continue;
    }
    if (inList) { html += "</ul>"; inList = false; }
    if (!line.trim() || /^<!--\s*orbit:/.test(line.trim())) continue;
    if (line.startsWith("# ")) html += `<h2>${inline(line.slice(2))}</h2>`;
    else if (line.startsWith("## ")) html += `<h3>${inline(line.slice(3))}</h3>`;
    else if (/^Progress:\s*\d+%/i.test(line)) {
      const amount = Math.min(100, parseInt(line.match(/\d+/)[0],10));
      html += `<div class="progress" title="${amount}% complete"><span style="width:${amount}%"></span></div>`;
    } else html += `<p>${inline(line)}</p>`;
  }
  if (inList) html += "</ul>";
  return html;
}

function renderNodes() {
  nodeLayer.innerHTML = "";
  (documentData.nodes || []).forEach(node => {
    const element = $("#nodeTemplate").content.firstElementChild.cloneNode(true);
    element.dataset.id = node.id;
    element.dataset.color = node.color || "";
    element.style.cssText = `left:${node.x}px;top:${node.y}px;width:${node.width}px;height:${node.height}px;`;
    element.classList.toggle("selected", selected?.kind === "node" && selected.id === node.id);
    element.classList.toggle("connect-source", connectSource === node.id);
    element.classList.toggle("filtered", activeFilter !== "all" && node.type !== "group" && node.color !== activeFilter);
    const content = $(".node-content", element);

    if (node.type === "group") {
      element.classList.add("group-node");
      content.innerHTML = `<div class="group-label">${escapeHTML(node.label || "Untitled group")}</div>`;
      $(".node-accent", element).remove();$(".connection-handles",element).remove();
    } else if (node.type === "text") {
      if(isAICard(node)){
        const config=parseAICard(node),inputs=inputNodesForAICard(node.id),runtime=aiCardRuntime.get(node.id)||{status:"Ready"};element.classList.add("ai-card");element.classList.toggle("running",runtime.running===true);
        content.innerHTML=`<div class="node-kicker">AI OPERATOR</div><h3 class="ai-card-title">${escapeHTML(config.title)}</h3><p class="ai-card-prompt">${escapeHTML(config.prompt)}</p><div class="ai-inputs">${inputs.length?inputs.map(input=>`<span class="ai-input-chip">← ${escapeHTML(nodeTitle(input))}</span>`).join(""):"<span class=\"ai-input-chip\">No inputs connected</span>"}</div><div class="ai-run-row"><span class="ai-run-status">${escapeHTML(runtime.status||"Ready")}</span><button class="ai-run-button" data-ai-run ${runtime.running?"disabled":""}>${runtime.running?"Running…":"Run now"}</button></div>`;
      } else content.innerHTML = `<div class="node-kicker">${textMeta(node)}</div>${markdownToHTML(node.text)}`;
    } else if (node.type === "link") {
      let linkTitle = "Saved link";
      try { linkTitle = new URL(node.url).hostname.replace(/^www\./, ""); } catch (_) {}
      content.innerHTML = `<div class="node-kicker">LINK</div><h3>${escapeHTML(linkTitle)}</h3><p>Open this resource in a new tab.</p><a class="node-link" href="${safeURL(node.url)}" target="_blank" rel="noreferrer">${escapeHTML(node.url)} ↗</a>`;
    } else if (node.type === "file") {
      const subcanvasId=subcanvasIdFromNode(node),subcanvas=subcanvasId&&workspace.canvases[subcanvasId];
      if(subcanvas){
        const children=Object.values(workspace.canvases).filter(record=>record.parentId===subcanvasId).length;element.classList.add("subcanvas-node");element.dataset.subcanvasId=subcanvasId;
        content.innerHTML=`<div class="node-kicker">SUB-CANVAS · ZOOM PORTAL</div><h3>${escapeHTML(subcanvas.title)}</h3><p>${subcanvas.document.nodes.length} item${subcanvas.document.nodes.length===1?"":"s"}${children?` · ${children} nested`:""}</p><div class="portal-preview">${portalPreview(subcanvas.document)}</div><div class="portal-actions"><span>Double-click or zoom to 220%</span><button type="button" data-open-subcanvas>Open ↘</button></div>`;
      } else if (/\.html?$/i.test(node.file)) {
        element.classList.add("html-widget");
        content.innerHTML = `<div class="node-kicker">LIVE HTML · SANDBOXED</div><iframe class="widget-frame" src="${safeFileURL(node.file)}" sandbox="allow-scripts" loading="lazy" referrerpolicy="no-referrer" title="${escapeHTML(node.file.split("/").pop())}"></iframe><div class="widget-shield"></div>`;
      } else content.innerHTML = `<div class="node-kicker">FILE</div><div class="file-preview">▧</div><h3>${escapeHTML(node.file.split("/").pop())}</h3><p>${escapeHTML(node.subpath || node.file)}</p>`;
    }
    element.addEventListener("pointerdown", event => nodePointerDown(event, node));
    $$("[data-connection-side]",element).forEach(handle=>{handle.addEventListener("pointerdown",event=>startConnectionDrag(event,node,handle.dataset.connectionSide));handle.addEventListener("keydown",event=>{if(event.key==="Enter"||event.key===" "){event.preventDefault();event.stopPropagation();connectSource=node.id;connectSourceSide=handle.dataset.connectionSide;setTool("connect");toast("Choose a destination node");}});});
    const aiRun=$("[data-ai-run]",element);if(aiRun){aiRun.addEventListener("pointerdown",event=>event.stopPropagation());aiRun.addEventListener("click",event=>{event.stopPropagation();runAICard(node.id,{manual:true});});}
    const portalButton=$("[data-open-subcanvas]",element);if(portalButton){portalButton.addEventListener("pointerdown",event=>event.stopPropagation());portalButton.addEventListener("click",event=>{event.stopPropagation();enterSubcanvas(element.dataset.subcanvasId);});element.addEventListener("dblclick",event=>{if(event.target.closest("button"))return;event.preventDefault();event.stopPropagation();enterSubcanvas(element.dataset.subcanvasId);});}
    element.addEventListener("click", event => {
      const anchor = event.target.closest("a");
      if (anchor) event.stopPropagation();
    });
    nodeLayer.appendChild(element);
  });
  updateCounts();
  renderMinimap();
}

function getPoint(node, side, other) {
  if (!side) {
    const dx = (other.x + other.width/2) - (node.x + node.width/2);
    const dy = (other.y + other.height/2) - (node.y + node.height/2);
    side = Math.abs(dx) > Math.abs(dy) ? (dx > 0 ? "right" : "left") : (dy > 0 ? "bottom" : "top");
  }
  const points = {
    top:[node.x+node.width/2,node.y], right:[node.x+node.width,node.y+node.height/2],
    bottom:[node.x+node.width/2,node.y+node.height], left:[node.x,node.y+node.height/2]
  };
  return { point:points[side], side };
}
function edgePath(from, to, fromSide, toSide) {
  const a = getPoint(from, fromSide, to), b = getPoint(to, toSide, from);
  const [x1,y1] = a.point, [x2,y2] = b.point;
  const distance = Math.max(45, Math.min(180, Math.hypot(x2-x1,y2-y1) * .38));
  const vectors = { top:[0,-1], right:[1,0], bottom:[0,1], left:[-1,0] };
  const av=vectors[a.side], bv=vectors[b.side];
  return { d:`M ${x1} ${y1} C ${x1+av[0]*distance} ${y1+av[1]*distance}, ${x2+bv[0]*distance} ${y2+bv[1]*distance}, ${x2} ${y2}`, mid:[(x1+x2)/2,(y1+y2)/2] };
}
function pointerEdgePath(node,side,point){
  const [x1,y1]=getPoint(node,side,{x:point.x,y:point.y,width:0,height:0}).point,vectors={top:[0,-1],right:[1,0],bottom:[0,1],left:[-1,0]},vector=vectors[side],distance=Math.max(45,Math.min(180,Math.hypot(point.x-x1,point.y-y1)*.38));
  return `M ${x1} ${y1} C ${x1+vector[0]*distance} ${y1+vector[1]*distance}, ${point.x} ${point.y}, ${point.x} ${point.y}`;
}
function startConnectionDrag(event,node,fromSide){
  if(event.button!==0)return;event.preventDefault();event.stopPropagation();
  const pointerId=event.pointerId,group=document.createElementNS("http://www.w3.org/2000/svg","g"),path=document.createElementNS("http://www.w3.org/2000/svg","path"),sourceElement=$(`.canvas-node[data-id="${CSS.escape(node.id)}"]`);group.classList.add("connection-preview");path.setAttribute("vector-effect","non-scaling-stroke");group.appendChild(path);edgeLayer.appendChild(group);sourceElement?.classList.add("connection-drag-source");document.body.classList.add("connection-dragging");
  let targetNode=null,targetElement=null,toSide=null;
  const clearTarget=()=>{targetElement?.classList.remove("connection-target");targetElement=null;targetNode=null;toSide=null;};
  const move=moveEvent=>{
    if(moveEvent.pointerId!==pointerId)return;const point=canvasPoint(moveEvent.clientX,moveEvent.clientY),candidateElement=document.elementFromPoint(moveEvent.clientX,moveEvent.clientY)?.closest?.(".canvas-node"),candidate=candidateElement&&candidateElement.dataset.id!==node.id?documentData.nodes.find(item=>item.id===candidateElement.dataset.id&&item.type!=="group"):null;
    if(candidateElement!==targetElement){clearTarget();if(candidate){targetElement=candidateElement;targetNode=candidate;targetElement.classList.add("connection-target");}}
    if(targetNode){toSide=getPoint(targetNode,undefined,node).side;path.setAttribute("d",edgePath(node,targetNode,fromSide,toSide).d);}else path.setAttribute("d",pointerEdgePath(node,fromSide,point));
  };
  const cleanup=()=>{clearTarget();group.remove();sourceElement?.classList.remove("connection-drag-source");document.body.classList.remove("connection-dragging");window.removeEventListener("pointermove",move);window.removeEventListener("pointerup",up);window.removeEventListener("pointercancel",cancel);window.removeEventListener("keydown",key);};
  const finish=(target,side)=>{if(!target)return;const before=aiCardSignatures();documentData.edges ||= [];documentData.edges.push({id:uid("edge"),fromNode:node.id,fromSide,toNode:target.id,toSide:side,toEnd:"arrow"});scheduleSave();scheduleChangedAICards(before);selected=null;shell.classList.remove("inspector-open");render();toast(`Connected to ${nodeTitle(target)}`);};
  const up=upEvent=>{if(upEvent.pointerId!==pointerId)return;move(upEvent);const completedTarget=targetNode,completedSide=toSide;cleanup();finish(completedTarget,completedSide);};
  const cancel=cancelEvent=>{if(cancelEvent.pointerId===pointerId)cleanup();};
  const key=keyEvent=>{if(keyEvent.key==="Escape"){keyEvent.preventDefault();cleanup();}};
  move(event);window.addEventListener("pointermove",move);window.addEventListener("pointerup",up);window.addEventListener("pointercancel",cancel);window.addEventListener("keydown",key);
}

function renderEdges() {
  edgeLayer.innerHTML = `<defs><marker id="arrow" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="context-stroke"/></marker></defs>`;
  const byId = Object.fromEntries((documentData.nodes || []).map(n => [n.id,n]));
  (documentData.edges || []).forEach(edge => {
    const from=byId[edge.fromNode], to=byId[edge.toNode];
    if (!from || !to) return;
    const path = edgePath(from,to,edge.fromSide,edge.toSide);
    const group = document.createElementNS("http://www.w3.org/2000/svg","g");
    group.classList.add("edge");
    if (selected?.kind === "edge" && selected.id === edge.id) group.classList.add("selected");
    const startMarker = edge.fromEnd === "arrow" ? 'marker-start="url(#arrow)"' : "";
    const endMarker = edge.toEnd === "none" ? "" : 'marker-end="url(#arrow)"';
    const color = colorValue(edge.color);
    group.innerHTML = `<path class="edge-hit" d="${path.d}" fill="none" stroke="transparent" stroke-width="14" vector-effect="non-scaling-stroke"/><path class="edge-line" d="${path.d}" style="stroke:${color}" ${startMarker} ${endMarker}/>`;
    if (edge.label) {
      const width = Math.max(35, edge.label.length*6+14);
      group.innerHTML += `<rect class="edge-label-bg" x="${path.mid[0]-width/2}" y="${path.mid[1]-10}" width="${width}" height="20" rx="5"/><text class="edge-label" x="${path.mid[0]}" y="${path.mid[1]}">${escapeHTML(edge.label)}</text>`;
    }
    $(".edge-hit",group).addEventListener("pointerdown", event => { event.stopPropagation(); selectItem("edge",edge.id); });
    edgeLayer.appendChild(group);
  });
}

function render() {
  applyCamera(); renderEdges(); renderNodes(); renderInspector(); renderWorkspaceNavigation(); updateAssistantContext();
}
function applyCamera() {
  world.style.transform = `translate(${camera.x}px,${camera.y}px) scale(${camera.zoom})`;
  canvas.style.backgroundSize = `${24*camera.zoom}px ${24*camera.zoom}px`;
  canvas.style.backgroundPosition = `${camera.x}px ${camera.y}px`;
  $("#zoomLabel").textContent = `${Math.round(camera.zoom*100)}%`;
  renderMinimap();
}

function canvasPoint(clientX,clientY) {
  const box=canvas.getBoundingClientRect();
  return { x:(clientX-box.left-camera.x)/camera.zoom, y:(clientY-box.top-camera.y)/camera.zoom };
}
function nodePointerDown(event,node) {
  if (event.button !== 0 || event.target.closest("a,button")) return;
  event.stopPropagation();
  if (currentTool === "connect") {
    if (!connectSource) { connectSource=node.id;connectSourceSide=null;toast("Now choose a destination"); }
    else if (connectSource !== node.id) {
      const before=aiCardSignatures();documentData.edges ||= [];
      const source=documentData.nodes.find(item=>item.id===connectSource),toSide=source?getPoint(node,undefined,source).side:undefined;
      documentData.edges.push({id:uid("edge"),fromNode:connectSource,...(connectSourceSide?{fromSide:connectSourceSide}:{}),toNode:node.id,toSide,toEnd:"arrow"});
      connectSource=null;connectSourceSide=null;setTool("select");scheduleSave();scheduleChangedAICards(before);toast("Nodes connected");
    }
    render(); return;
  }
  selectItem("node",node.id);
  const resizing = event.target.classList.contains("resize-handle");
  const start={x:event.clientX,y:event.clientY,nx:node.x,ny:node.y,w:node.width,h:node.height};
  const move = e => {
    if (resizing) {
      node.width=Math.max(120,Math.round(start.w+(e.clientX-start.x)/camera.zoom));
      node.height=Math.max(70,Math.round(start.h+(e.clientY-start.y)/camera.zoom));
    } else {
      node.x=Math.round(start.nx+(e.clientX-start.x)/camera.zoom);
      node.y=Math.round(start.ny+(e.clientY-start.y)/camera.zoom);
    }
    const el=$(`.canvas-node[data-id="${CSS.escape(node.id)}"]`);
    el.style.left=node.x+"px"; el.style.top=node.y+"px"; el.style.width=node.width+"px"; el.style.height=node.height+"px";
    renderEdges(); renderMinimap();
  };
  const up = () => { window.removeEventListener("pointermove",move); scheduleSave(); renderInspector(); };
  window.addEventListener("pointermove",move); window.addEventListener("pointerup",up,{once:true});
}

canvas.addEventListener("pointerdown", event => {
  if (event.button === 1 || event.button === 0 && (spaceDown || currentTool === "pan")) {
    event.preventDefault(); canvas.classList.add("panning");
    const start={x:event.clientX,y:event.clientY,cx:camera.x,cy:camera.y};
    const move=e=>{camera.x=start.cx+e.clientX-start.x;camera.y=start.cy+e.clientY-start.y;applyCamera();};
    const up=()=>{canvas.classList.remove("panning");window.removeEventListener("pointermove",move);scheduleSave()};
    window.addEventListener("pointermove",move);window.addEventListener("pointerup",up,{once:true}); return;
  }
  if (event.target === canvas || event.target === world || event.target === nodeLayer) {
    selected=null;connectSource=null;connectSourceSide=null;shell.classList.remove("inspector-open");render();
    if (currentTool === "note") { const p=canvasPoint(event.clientX,event.clientY); addNode("note",p); setTool("select"); }
  }
});
canvas.addEventListener("dblclick", event => {
  if (event.target===canvas || event.target===world || event.target===nodeLayer) addNode("note",canvasPoint(event.clientX,event.clientY));
});
canvas.addEventListener("wheel", event => {
  event.preventDefault();
  const portal=event.target.closest?.(".subcanvas-node"),portalId=portal?.dataset.subcanvasId;
  if(event.deltaY>0&&camera.zoom<=.205&&canvasRecord().parentId){leaveSubcanvas();return;}
  const rect=canvas.getBoundingClientRect(), sx=event.clientX-rect.left, sy=event.clientY-rect.top;
  const worldX=(sx-camera.x)/camera.zoom, worldY=(sy-camera.y)/camera.zoom;
  const factor=Math.exp(-event.deltaY*.0012), next=Math.max(.2,Math.min(2.5,camera.zoom*factor));
  camera.x=sx-worldX*next; camera.y=sy-worldY*next; camera.zoom=next; applyCamera();scheduleSave();
  if(event.deltaY<0&&portalId&&next>=2.2)enterSubcanvas(portalId);
},{passive:false});

function selectItem(kind,id) {
  selected={kind,id}; shell.classList.add("inspector-open"); render();
}
function setTool(tool) {
  currentTool=tool;if(tool!=="connect"){connectSource=null;connectSourceSide=null;}
  $$(".tool").forEach(b=>b.classList.toggle("active",b.dataset.tool===tool));
  canvas.classList.toggle("tool-pan",tool==="pan"); renderNodes();
}

function addNode(kind, point) {
  if(kind==="subcanvas")return createSubcanvas(point);
  const center = point || canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2);
  const presets={
    note:{type:"text",color:"2",width:260,height:150,text:"# New thought\nStart writing here…"},
    goal:{type:"text",color:"1",width:300,height:190,text:"# A meaningful goal\nWhat would make this worth doing?\n\n- [ ] Define the first step\n\nProgress: 0%"},
    habit:{type:"text",color:"4",width:280,height:145,text:"# New daily practice\nMake it small enough to begin today."},
    project:{type:"text",color:"6",width:300,height:210,text:"# Untitled project\nDescribe the outcome, not just the activity.\n\n- [ ] First milestone\n- [ ] Next milestone\n\nProgress: 0%"},
    ai:{type:"text",color:"5",width:330,height:210,text:`${AI_CARD_MARKER}\n# Weekly synthesis\nSummarize the connected notes. Highlight progress, blockers, and the most useful next action.`},
    widget:{type:"file",color:"5",width:480,height:290,file:"widgets/focus-orbit.html"},
    group:{type:"group",color:"5",width:620,height:430,label:"New area"}
  };
  const preset=presets[kind]||presets.note;
  const node={id:uid("node"),...preset,x:Math.round(center.x-preset.width/2),y:Math.round(center.y-preset.height/2)};
  documentData.nodes ||= [];
  if (kind==="group") documentData.nodes.unshift(node); else documentData.nodes.push(node);
  selected={kind:"node",id:node.id}; shell.classList.add("inspector-open"); scheduleSave(); render();
  return node;
}

function renderInspector() {
  const panel=$("#inspector");
  if (!selected) { panel.innerHTML='<div class="inspector-empty"><span>↖</span><h3>Nothing selected</h3><p>Select a card or connection to edit its details.</p></div>'; return; }
  const item = selected.kind==="node" ? documentData.nodes.find(n=>n.id===selected.id) : documentData.edges.find(e=>e.id===selected.id);
  if (!item) { selected=null; shell.classList.remove("inspector-open"); renderInspector(); return; }
  const colorButtons=Object.entries(COLORS).map(([key,value])=>`<button type="button" class="color-choice ${item.color===key?"active":""}" data-color="${key}" style="background:${value}" aria-label="Color ${key}"></button>`).join("");
  if (selected.kind==="node") {
    let contentField="";
    if (item.type==="text"&&isAICard(item)){const config=parseAICard(item);contentField=`<label class="field"><span>Operator name</span><input data-key="aiTitle" value="${escapeHTML(config.title)}"></label><label class="field"><span>AI instructions</span><textarea data-key="aiPrompt">${escapeHTML(config.prompt)}</textarea></label><div class="field-hint">Incoming connections become context. The generated note updates automatically when that context changes.</div>`;}
    else if (item.type==="text") contentField=`<label class="field"><span>Markdown</span><textarea data-key="text">${escapeHTML(item.text)}</textarea></label>`;
    if (item.type==="link") contentField=`<label class="field"><span>URL</span><input data-key="url" value="${escapeHTML(item.url)}"></label>`;
    if (item.type==="file") {const subcanvasId=subcanvasIdFromNode(item),subcanvas=subcanvasId&&workspace.canvases[subcanvasId];contentField=subcanvas?`<label class="field"><span>Canvas title</span><input data-canvas-title="${escapeHTML(subcanvasId)}" value="${escapeHTML(subcanvas.title)}"></label><div class="field-hint">This portal is a standard JSON Canvas file node. Double-click it or zoom in to enter the nested canvas.</div><button type="button" class="button open-subcanvas-inspector" data-open-canvas="${escapeHTML(subcanvasId)}">Open sub-canvas ↘</button>`:`<label class="field"><span>File path</span><input data-key="file" value="${escapeHTML(item.file)}"></label><label class="field"><span>Subpath</span><input data-key="subpath" value="${escapeHTML(item.subpath||"")}"></label>`;}
    if (item.type==="group") contentField=`<label class="field"><span>Label</span><input data-key="label" value="${escapeHTML(item.label||"")}"></label><label class="field"><span>Background path</span><input data-key="background" value="${escapeHTML(item.background||"")}"></label>`;
    panel.innerHTML=`<div class="inspector-head"><h3>${item.type[0].toUpperCase()+item.type.slice(1)} node</h3><button class="close-inspector">×</button></div><form class="inspector-form">${contentField}<div class="field-row"><label class="field"><span>X</span><input type="number" data-key="x" value="${item.x}"></label><label class="field"><span>Y</span><input type="number" data-key="y" value="${item.y}"></label></div><div class="field-row"><label class="field"><span>Width</span><input type="number" data-key="width" value="${item.width}"></label><label class="field"><span>Height</span><input type="number" data-key="height" value="${item.height}"></label></div><label class="field"><span>Color preset</span><div class="color-list">${colorButtons}</div></label><button type="button" class="danger-btn">Delete node</button></form>`;
  } else {
    panel.innerHTML=`<div class="inspector-head"><h3>Connection</h3><button class="close-inspector">×</button></div><form class="inspector-form"><label class="field"><span>Label</span><input data-key="label" value="${escapeHTML(item.label||"")}"></label><div class="field-row"><label class="field"><span>From side</span><select data-key="fromSide">${sideOptions(item.fromSide)}</select></label><label class="field"><span>To side</span><select data-key="toSide">${sideOptions(item.toSide)}</select></label></div><label class="field"><span>Color preset</span><div class="color-list">${colorButtons}</div></label><button type="button" class="danger-btn">Delete connection</button></form>`;
  }
  $(".close-inspector",panel).onclick=()=>{selected=null;shell.classList.remove("inspector-open");render();};
  $$("[data-key]",panel).forEach(input=>input.addEventListener("input",()=>{
    const before=aiCardSignatures(),key=input.dataset.key;
    if(key==="aiTitle"||key==="aiPrompt"){const config=parseAICard(item);item.text=buildAICardText(key==="aiTitle"?input.value:config.title,key==="aiPrompt"?input.value:config.prompt);}
    else item[key]=input.type==="number"?Math.round(Number(input.value)):input.value;
    if (input.tagName==="SELECT" && !input.value) delete item[key];
    scheduleSave(); renderNodes(); renderEdges(); renderMinimap(); scheduleChangedAICards(before);
  }));
  const canvasTitleField=$("[data-canvas-title]",panel);if(canvasTitleField)canvasTitleField.addEventListener("input",()=>{const record=workspace.canvases[canvasTitleField.dataset.canvasTitle];if(!record)return;record.title=canvasTitleField.value||"Untitled canvas";scheduleSave();renderNodes();renderWorkspaceNavigation();});
  const openCanvas=$("[data-open-canvas]",panel);if(openCanvas)openCanvas.onclick=()=>enterSubcanvas(openCanvas.dataset.openCanvas);
  $$(".color-choice",panel).forEach(button=>button.onclick=()=>{item.color=button.dataset.color;scheduleSave();render();});
  $(".danger-btn",panel).onclick=deleteSelection;
}
function sideOptions(value) { return ["","top","right","bottom","left"].map(s=>`<option value="${s}" ${value===s?"selected":""}>${s||"Auto"}</option>`).join(""); }
function deleteSelection() {
  if (!selected) return;const before=aiCardSignatures();
  if (selected.kind==="node") {
    const node=documentData.nodes.find(item=>item.id===selected.id),subcanvasId=subcanvasIdFromNode(node);
    if(subcanvasId&&!confirm(`Delete “${workspace.canvases[subcanvasId].title}” and every canvas nested inside it?`))return;
    if(subcanvasId)deleteCanvasTree(subcanvasId);
    documentData.nodes=documentData.nodes.filter(n=>n.id!==selected.id);
    documentData.edges=(documentData.edges||[]).filter(e=>e.fromNode!==selected.id&&e.toNode!==selected.id);
  } else documentData.edges=documentData.edges.filter(e=>e.id!==selected.id);
  selected=null;shell.classList.remove("inspector-open");scheduleSave();render();scheduleChangedAICards(before);toast("Deleted");
}

function updateCounts() {
  const nodes=(documentData.nodes||[]).filter(n=>n.type!=="group");
  $("#allCount").textContent=nodes.length;
  [["goalCount","1"],["habitCount","4"],["projectCount","6"],["ideaCount","2"]].forEach(([id,c])=>$("#"+id).textContent=nodes.filter(n=>n.color===c).length);
}
function renderMinimap() {
  const mini=$("#miniWorld"),view=$("#miniViewport");if(!mini)return;if(!documentData.nodes?.length){mini.innerHTML="";view.style.cssText="display:none";return;}view.style.display="";
  const bounds=getBounds(), pad=8, mw=128-pad*2,mh=82-pad*2, scale=Math.min(mw/bounds.width,mh/bounds.height,.12);
  const ox=pad+(mw-bounds.width*scale)/2-bounds.minX*scale, oy=pad+(mh-bounds.height*scale)/2-bounds.minY*scale;
  mini.innerHTML=documentData.nodes.map(n=>`<i class="mini-node ${n.type==="group"?"group":""}" style="left:${ox+n.x*scale}px;top:${oy+n.y*scale}px;width:${Math.max(2,n.width*scale)}px;height:${Math.max(2,n.height*scale)}px;background-color:${n.type==="group"?"transparent":colorValue(n.color)}"></i>`).join("");
  const worldLeft=-camera.x/camera.zoom, worldTop=-camera.y/camera.zoom;
  view.style.cssText=`left:${ox+worldLeft*scale}px;top:${oy+worldTop*scale}px;width:${canvas.clientWidth/camera.zoom*scale}px;height:${canvas.clientHeight/camera.zoom*scale}px`;
}
function getBounds() {
  const nodes=documentData.nodes||[]; if(!nodes.length)return{minX:0,minY:0,width:1,height:1};
  const minX=Math.min(...nodes.map(n=>n.x)),minY=Math.min(...nodes.map(n=>n.y));
  return {minX,minY,width:Math.max(...nodes.map(n=>n.x+n.width))-minX,height:Math.max(...nodes.map(n=>n.y+n.height))-minY};
}
function fitView() {
  const b=getBounds(), pad=75; camera.zoom=Math.max(.2,Math.min(1.15,Math.min((canvas.clientWidth-pad*2)/b.width,(canvas.clientHeight-pad*2)/b.height)));
  camera.x=(canvas.clientWidth-b.width*camera.zoom)/2-b.minX*camera.zoom; camera.y=(canvas.clientHeight-b.height*camera.zoom)/2-b.minY*camera.zoom; applyCamera();
}
function setZoom(next) {
  const cx=canvas.clientWidth/2,cy=canvas.clientHeight/2,wx=(cx-camera.x)/camera.zoom,wy=(cy-camera.y)/camera.zoom;
  camera.zoom=Math.max(.2,Math.min(2.5,next));camera.x=cx-wx*camera.zoom;camera.y=cy-wy*camera.zoom;applyCamera();
}
function toast(message) { const el=$("#toast");el.textContent=message;el.classList.add("show");clearTimeout(el._timer);el._timer=setTimeout(()=>el.classList.remove("show"),1800); }

function exportCanvas() {
  const blob=new Blob([JSON.stringify(documentData,null,2)],{type:"application/json"});
  const anchor=document.createElement("a");anchor.href=URL.createObjectURL(blob);anchor.download=($("#canvasTitle").value||"life-canvas").toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,"")+".canvas";anchor.click();URL.revokeObjectURL(anchor.href);toast("Canvas exported");
}
async function importCanvas(file) {
  try { const parsed=JSON.parse(await file.text());if(!isCanvas(parsed))throw new Error("Not a valid JSON Canvas 1.0 document");documentData={nodes:parsed.nodes||[],edges:parsed.edges||[]};selected=null;scheduleSave();render();fitView();toast("Canvas imported"); }
  catch(error){alert(`Could not import this file.\n\n${error.message}`);}
}

// Canvas-aware assistant prototype. A remote model should produce these operations,
// never arbitrary host-page JavaScript. Each operation is checked before commit.
function validateCanvasOperations(operations) {
  if(!Array.isArray(operations)||operations.length>50||JSON.stringify(operations).length>100000)throw new Error("The operation plan is too large or malformed");
  const draft=clone(documentData),themes=[],nodeKeys=new Set(["text","file","subpath","url","label","background","backgroundStyle","x","y","width","height","color"]),edgeKeys=new Set(["fromSide","fromEnd","toSide","toEnd","color","label"]);
  for (const operation of operations) {
    if (!operation || typeof operation.type!=="string") throw new Error("Malformed canvas operation");
    if (operation.type==="node.add") draft.nodes.push(clone(operation.node));
    else if (operation.type==="node.update") {
      const node=draft.nodes.find(item=>item.id===operation.id);if(!node)throw new Error(`Unknown node ${operation.id}`);
      for(const [key,value] of Object.entries(operation.patch||{})){if(!nodeKeys.has(key))throw new Error(`Field ${key} cannot be changed`);node[key]=value;}
    } else if (operation.type==="node.remove") {
      draft.nodes=draft.nodes.filter(item=>item.id!==operation.id);draft.edges=draft.edges.filter(edge=>edge.fromNode!==operation.id&&edge.toNode!==operation.id);
    } else if (operation.type==="edge.add") draft.edges.push(clone(operation.edge));
    else if (operation.type==="edge.update") {
      const edge=draft.edges.find(item=>item.id===operation.id);if(!edge)throw new Error(`Unknown edge ${operation.id}`);
      for(const [key,value] of Object.entries(operation.patch||{})){if(!edgeKeys.has(key))throw new Error(`Field ${key} cannot be changed`);edge[key]=value;}
    } else if(operation.type==="edge.remove")draft.edges=draft.edges.filter(item=>item.id!==operation.id);
    else if (operation.type==="theme.set") {
      if(!["default","warm","calm","contrast"].includes(operation.theme))throw new Error("Unknown theme");themes.push(operation.theme);
    } else throw new Error(`Unsupported operation ${operation.type}`);
  }
  if(!isCanvas(draft))throw new Error("The resulting canvas is not valid JSON Canvas 1.0");
  return {draft,themes};
}
function applyCanvasOperations(operations) {
  const before=aiCardSignatures(),{draft,themes}=validateCanvasOperations(operations);documentData=draft;themes.forEach(applyCanvasTheme);
  selected=null;shell.classList.remove("inspector-open");scheduleSave();render();updateAssistantContext();scheduleChangedAICards(before);
}

function applyCanvasTheme(theme) {
  const allowed=new Set(["default","warm","calm","contrast"]), value=allowed.has(theme)?theme:"default";
  if(value==="default")document.body.removeAttribute("data-canvas-theme");else document.body.dataset.canvasTheme=value;
  localStorage.setItem("orbit-canvas-theme",value);
}
function canvasSummary() {
  const nodes=(documentData.nodes||[]).filter(node=>node.type!=="group"), counts={goals:0,habits:0,projects:0,ideas:0,widgets:0,subcanvases:0};
  nodes.forEach(node=>{if(node.color==="1")counts.goals++;if(node.color==="4")counts.habits++;if(node.color==="6")counts.projects++;if(node.color==="2")counts.ideas++;if(node.type==="file"&&/\.html?$/i.test(node.file))counts.widgets++;if(subcanvasIdFromNode(node))counts.subcanvases++;});
  const openTasks=nodes.filter(n=>n.type==="text").reduce((total,n)=>total+(n.text.match(/- \[ \]/g)||[]).length,0);
  return {canvasId:currentCanvasId,canvasTitle:canvasRecord().title,nodes:nodes.length,edges:(documentData.edges||[]).length,openTasks,...counts};
}
function updateAssistantContext() {
  const context=$("#aiContext");if(!context)return;const s=canvasSummary();
  context.innerHTML=`READING <b>${escapeHTML(s.canvasTitle)}</b> · <b>${s.nodes} nodes</b> · <b>${s.edges} links</b> · <b>${s.openTasks} tasks</b> · <b>${s.subcanvases} portals</b>`;
}
function setAssistantOpen(open) {
  const panel=$("#aiPanel");panel.classList.toggle("open",open);panel.setAttribute("aria-hidden",String(!open));panel.inert=!open;updateAssistantContext();if(open)setTimeout(()=>$("#aiPrompt").focus(),180);
}
function assistantMessage(text,role="assistant") {
  const message=document.createElement("div");message.className=`ai-message ${role}`;message.innerHTML=role==="assistant"?"<span>✦</span><p></p>":"<p></p>";$("p",message).textContent=text;$("#aiMessages").append(message);message.scrollIntoView({behavior:"smooth",block:"end"});return message;
}
function operationDescription(operation) {
  const names={"node.add":"Add node","node.update":"Update node","node.remove":"Delete node","edge.add":"Add connection","edge.update":"Update connection","edge.remove":"Delete connection","theme.set":"Set theme"};
  const target=operation.id||operation.node?.id||operation.edge?.id||operation.theme||"";return `<div><b>${escapeHTML(names[operation.type]||operation.type)}</b>${target?` · ${escapeHTML(target)}`:""}</div>`;
}
function assistantProposal(text,operations) {
  validateCanvasOperations(operations);const message=document.createElement("div");message.className="ai-message assistant";message.innerHTML=`<span>✦</span><div class="ai-proposal"><p></p>${operations.length?`<div class="ai-operation-list">${operations.map(operationDescription).join("")}</div><div class="ai-proposal-actions"><button class="apply">Apply ${operations.length} change${operations.length===1?"":"s"}</button><button class="discard">Discard</button></div>`:""}</div>`;$("p",message).textContent=text||"I reviewed the canvas.";$("#aiMessages").append(message);
  if(operations.length){const apply=$(".apply",message),discard=$(".discard",message);apply.onclick=()=>{try{applyCanvasOperations(operations);apply.textContent="Applied";apply.disabled=true;discard.remove();toast("AI changes applied");}catch(error){assistantMessage(`I could not apply that plan: ${error.message}`);}};discard.onclick=()=>{apply.disabled=true;discard.textContent="Discarded";discard.disabled=true;};}
  message.scrollIntoView({behavior:"smooth",block:"end"});
}
function runLocalAssistant(prompt) {
  const request=prompt.trim();if(!request)return;assistantMessage(request,"user");const lower=request.toLowerCase();let response="";
  try {
    if(/summar|what(?:'s| is) (?:on|in)|parse/.test(lower)) {
      const s=canvasSummary();response=`I parsed the current JSON Canvas: ${s.nodes} content nodes and ${s.edges} connections. I found ${s.goals} goals, ${s.projects} projects, ${s.habits} habits, ${s.ideas} ideas, ${s.widgets} live widgets, and ${s.openTasks} unchecked tasks.`;
    } else if(/warm|cozy|earth/.test(lower)) {applyCanvasOperations([{type:"theme.set",theme:"warm"}]);response="Applied a warmer, earth-toned canvas theme. This visual preference stays separate from the portable .canvas document.";
    } else if(/calm|ocean|cool|teal/.test(lower)) {applyCanvasOperations([{type:"theme.set",theme:"calm"}]);response="Applied the calm teal canvas theme.";
    } else if(/contrast|accessible/.test(lower)) {applyCanvasOperations([{type:"theme.set",theme:"contrast"}]);response="Applied the high-contrast canvas theme.";
    } else if(/reset.*(?:theme|style)|default (?:theme|style)/.test(lower)) {applyCanvasOperations([{type:"theme.set",theme:"default"}]);response="Reset the canvas styling to its default theme.";
    } else if(/(?:add|create).*(?:sub.?canvas|nested canvas)/.test(lower)){createSubcanvas();response="Created a nested canvas portal. Double-click it or zoom into it to enter.";
    } else if(/(?:add|create).*(?:3d|three|html|widget)/.test(lower)) {
      const center=canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),node={id:uid("node"),type:"file",x:Math.round(center.x-240),y:Math.round(center.y-145),width:480,height:290,color:"5",file:"widgets/focus-orbit.html"};applyCanvasOperations([{type:"node.add",node}]);response="Added a sandboxed Three.js file node. It is still a standard JSON Canvas file node pointing to an HTML file.";
    } else {
      const match=request.match(/(?:add|create)\s+(?:a |an )?(goal|habit|project|note)(?:\s+(?:called|named|to))?\s+(.+)/i);
      if(match){const kind=match[1].toLowerCase(),title=match[2].replace(/[.!]$/,"");const preset={goal:["1",`# ${title}\nWhat does success look like?\n\n- [ ] Choose the first step\n\nProgress: 0%`],habit:["4",`# ${title}\nMake the practice small and repeatable.`],project:["6",`# ${title}\nDefine the desired outcome.\n\n- [ ] First milestone\n\nProgress: 0%`],note:["2",`# ${title}\nStart writing here…`]}[kind];const center=canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),node={id:uid("node"),type:"text",x:Math.round(center.x-150),y:Math.round(center.y-90),width:300,height:kind==="project"||kind==="goal"?200:150,color:preset[0],text:preset[1]};applyCanvasOperations([{type:"node.add",node}]);response=`Added “${title}” as a ${kind} near the center of your current view.`;}
      else response="I understand this canvas, but the GitHub Pages demo uses a local intent parser rather than a remote model. Try asking me to summarize it, add a goal/habit/project, add a 3D widget, or change the theme to warm, calm, or high contrast.";
    }
  } catch(error){response=`I did not apply that change: ${error.message}`;}
  setTimeout(()=>assistantMessage(response),180);
}

const AI_SETTINGS_KEY="orbit-ai-provider-v1",AI_SECRET_KEY="orbit-ai-secret-v1";
let aiConversation=[];
function loadAISettings() {
  let saved={};try{saved=JSON.parse(localStorage.getItem(AI_SETTINGS_KEY)||"{}");}catch(_){}
  return {baseURL:saved.baseURL||"https://api.mistral.ai/v1",model:saved.model||"mistral-small-latest",rememberKey:Boolean(saved.rememberKey),apiKey:(saved.rememberKey?localStorage:sessionStorage).getItem(AI_SECRET_KEY)||""};
}
let aiSettings=loadAISettings();
function checkedProviderURL(value) {
  const url=new URL(value);if(url.protocol!=="https:"&&!(url.protocol==="http:"&&["localhost","127.0.0.1"].includes(url.hostname)))throw new Error("Use HTTPS, or HTTP only for a localhost provider");
  url.pathname=url.pathname.replace(/\/$/,"");url.search="";url.hash="";return url.toString().replace(/\/$/,"");
}
function settingsFromForm() {return {baseURL:checkedProviderURL($("#aiBaseURL").value.trim()),model:$("#aiModel").value.trim(),apiKey:$("#aiAPIKey").value.trim(),rememberKey:$("#rememberAIKey").checked};}
function persistAISettings(settings) {
  localStorage.setItem(AI_SETTINGS_KEY,JSON.stringify({baseURL:settings.baseURL,model:settings.model,rememberKey:settings.rememberKey}));localStorage.removeItem(AI_SECRET_KEY);sessionStorage.removeItem(AI_SECRET_KEY);(settings.rememberKey?localStorage:sessionStorage).setItem(AI_SECRET_KEY,settings.apiKey);aiSettings=settings;aiConversation=[];updateProviderUI();
}
function updateProviderUI() {
  const remote=Boolean(aiSettings.apiKey&&aiSettings.baseURL&&aiSettings.model),label=$("#aiProviderLabel"),status=$("#aiProviderStatus");
  label.textContent=remote?aiSettings.model:"Local canvas tools";status.classList.toggle("remote",remote);status.innerHTML=remote?`<i></i> Direct connection · ${escapeHTML(new URL(aiSettings.baseURL).hostname)}`:"<i></i> Local mode — canvas data stays in this browser";
}
function openAISettings() {
  $("#aiBaseURL").value=aiSettings.baseURL;$("#aiModel").value=aiSettings.model;$("#aiAPIKey").value=aiSettings.apiKey;$("#rememberAIKey").checked=aiSettings.rememberKey;setSettingsResult("");$("#aiSettingsDialog").showModal();
}
function setSettingsResult(message,type="") {const result=$("#aiSettingsResult");result.textContent=message;result.className=`settings-test ${type}`;}
async function providerFetch(settings,path,options={}) {
  const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),60000),headers={Authorization:`Bearer ${settings.apiKey}`,...options.headers};
  try{const response=await fetch(`${settings.baseURL}${path}`,{...options,headers,signal:controller.signal});if(!response.ok){let detail="";try{const body=await response.json();detail=body.error?.message||body.message||"";}catch(_){detail=await response.text();}throw new Error(`${response.status} ${response.statusText}${detail?`: ${detail.slice(0,240)}`:""}`);}return response;}catch(error){if(error.name==="AbortError")throw new Error("The provider request timed out");if(error instanceof TypeError)throw new Error("Network or CORS error. Check that this provider permits browser requests.");throw error;}finally{clearTimeout(timer);}
}
async function testAIProvider(settings) {
  const response=await providerFetch(settings,"/models",{method:"GET"}),body=await response.json(),models=Array.isArray(body.data)?body.data.length:null;return models===null?"Connected successfully.":`Connected successfully · ${models} models available.`;
}
function assistantSystemPrompt() {
  return `You are Orbit Copilot, an assistant operating a JSON Canvas 1.0 life-management canvas. Respond with exactly one JSON object and no markdown fences: {"message":"Brief response to the user","operations":[]}.
Allowed operations:
{"type":"node.add","node":<complete JSON Canvas node with unique id, type, integer x/y/width/height and required type field>}
{"type":"node.update","id":"existing id","patch":<changed standard fields>}
{"type":"node.remove","id":"existing id"}
{"type":"edge.add","edge":<complete JSON Canvas edge with unique id>}
{"type":"edge.update","id":"existing id","patch":<changed edge fields>}
{"type":"edge.remove","id":"existing id"}
{"type":"theme.set","theme":"default|warm|calm|contrast"}
Use only standard JSON Canvas fields. Use Markdown checkboxes in text nodes for tasks. Colors: 1 red/goals, 2 orange/ideas, 3 yellow/notes, 4 green/habits, 5 cyan/resources, 6 purple/projects. Preserve user data unless explicitly asked to remove it. Ask a question with an empty operations array if intent is ambiguous. Never put executable HTML or JavaScript in a text node. A live widget is a file node pointing to widgets/focus-orbit.html. Keep responses concise.`;
}
function parseProviderJSON(content) {
  if(Array.isArray(content))content=content.map(part=>part.text||part.content||"").join("");if(typeof content!=="string")throw new Error("Provider returned no text content");
  const cleaned=content.trim().replace(/^```(?:json)?\s*/i,"").replace(/\s*```$/,"");const start=cleaned.indexOf("{"),end=cleaned.lastIndexOf("}");if(start<0||end<start)throw new Error("Provider did not return the requested JSON plan");
  const parsed=JSON.parse(cleaned.slice(start,end+1));if(typeof parsed.message!=="string"||!Array.isArray(parsed.operations))throw new Error("Provider response is missing message or operations");return parsed;
}
async function runRemoteAssistant(prompt) {
  assistantMessage(prompt,"user");const loading=assistantMessage("Thinking…");loading.classList.add("loading");const send=$("#aiForm button");send.disabled=true;
  try{
    const box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2),context=`Current canvas: ${canvasTrail().map(record=>record.title).join(" / ")}\nCurrent viewport center: ${Math.round(center.x)}, ${Math.round(center.y)}.\nCurrent JSON Canvas:\n${JSON.stringify(documentData)}`;
    const messages=[{role:"system",content:assistantSystemPrompt()},...aiConversation.slice(-8),{role:"user",content:`${prompt}\n\n${context}`}];
    const response=await providerFetch(aiSettings,"/chat/completions",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({model:aiSettings.model,messages,temperature:.2,max_tokens:1800})}),body=await response.json(),content=body.choices?.[0]?.message?.content,plan=parseProviderJSON(content);
    validateCanvasOperations(plan.operations);loading.remove();assistantProposal(plan.message,plan.operations);aiConversation.push({role:"user",content:prompt},{role:"assistant",content:JSON.stringify(plan)});
  }catch(error){loading.remove();assistantMessage(`Provider error: ${error.message}`);}finally{send.disabled=false;$("#aiPrompt").focus();}
}
function runAssistant(prompt) {if(!prompt.trim())return;if(aiSettings.apiKey&&aiSettings.baseURL&&aiSettings.model)runRemoteAssistant(prompt);else runLocalAssistant(prompt);}

function aiCardHasCycle(cardId){
  const next=new Map();for(const edge of documentData.edges||[]){if(!next.has(edge.fromNode))next.set(edge.fromNode,[]);next.get(edge.fromNode).push(edge.toNode);}const seen=new Set();
  function visit(id){if(seen.has(id))return false;seen.add(id);for(const child of next.get(id)||[]){if(child===cardId||visit(child))return true;}return false;}return visit(cardId);
}
function scheduleAICard(cardId,delay=1200){
  const card=documentData.nodes.find(node=>node.id===cardId&&isAICard(node));if(!card)return;const state=aiCardRuntime.get(cardId)||{};clearTimeout(state.timer);
  if(aiCardHasCycle(cardId)){state.status="Paused · connection cycle";aiCardRuntime.set(cardId,state);renderNodes();return;}
  state.status="Inputs changed · queued";state.timer=setTimeout(()=>runAICard(cardId),delay);aiCardRuntime.set(cardId,state);renderNodes();
}
function scheduleChangedAICards(before) {
  const after=aiCardSignatures();for(const [id,signature] of after){if(before.get(id)!==signature&&(before.has(id)||inputNodesForAICard(id).length))scheduleAICard(id);}
}
function providerMessageContent(content){if(Array.isArray(content))content=content.map(part=>part.text||part.content||"").join("");if(typeof content!=="string"||!content.trim())throw new Error("Provider returned an empty note");return content.trim().replace(/^```(?:markdown|md)?\s*/i,"").replace(/\s*```$/,"").replaceAll(AI_CARD_MARKER,"").trim();}
function openAINoteDialog(){const dialog=$("#aiNoteDialog");$("#aiNoteResult").textContent="";$("#aiNoteResult").className="settings-test";dialog.showModal();setTimeout(()=>$("#aiNotePrompt").focus(),50);}
async function createAINote(prompt){
  if(!aiSettings.apiKey){$("#aiNoteDialog").close();setAssistantOpen(true);openAISettings();toast("Configure an AI provider, then add the AI note again");return;}
  const dialog=$("#aiNoteDialog"),button=$("#generateAINote"),result=$("#aiNoteResult");dialog.classList.add("generating");button.disabled=true;result.className="settings-test";result.textContent=`Asking ${aiSettings.model}…`;
  try{
    const response=await providerFetch(aiSettings,"/chat/completions",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({model:aiSettings.model,messages:[{role:"system",content:"Create one clear, useful Markdown note that answers the user's prompt. Return only the note Markdown without code fences, commentary, HTML, or scripts. Give the note a concise level-one heading."},{role:"user",content:prompt}],temperature:.4,max_tokens:2200})}),body=await response.json();let generated=providerMessageContent(body.choices?.[0]?.message?.content);if(!/^#\s/m.test(generated))generated=`# AI note\n\n${generated}`;
    const box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2),node={id:uid("node"),type:"text",x:Math.round(center.x-190),y:Math.round(center.y-130),width:380,height:Math.min(480,Math.max(240,180+Math.round(generated.length/12))),color:"5",text:generated};documentData.nodes.push(node);selected={kind:"node",id:node.id};shell.classList.add("inspector-open");scheduleSave();render();dialog.close();$("#aiNotePrompt").value="";toast("AI note added");
  }catch(error){result.className="settings-test error";result.textContent=error.message;}
  finally{dialog.classList.remove("generating");button.disabled=false;}
}
async function runAICard(cardId,{manual=false}={}) {
  const card=documentData.nodes.find(node=>node.id===cardId&&isAICard(node));if(!card)return;const state=aiCardRuntime.get(cardId)||{};clearTimeout(state.timer);
  if(state.running){state.pending=true;aiCardRuntime.set(cardId,state);return;}if(!aiSettings.apiKey){state.status="Configure an AI provider";aiCardRuntime.set(cardId,state);renderNodes();setAssistantOpen(true);openAISettings();return;}
  if(aiCardHasCycle(cardId)&&!manual){state.status="Paused · connection cycle";aiCardRuntime.set(cardId,state);renderNodes();return;}
  const signature=aiCardSignature(card);if(!manual&&state.lastSignature===signature){state.status="Up to date";aiCardRuntime.set(cardId,state);renderNodes();return;}
  const config=parseAICard(card),inputs=inputNodesForAICard(card.id);state.running=true;state.pending=false;state.status=`Reading ${inputs.length} input${inputs.length===1?"":"s"}…`;aiCardRuntime.set(cardId,state);renderNodes();
  try{
    const inputText=inputs.length?inputs.map((node,index)=>`## Input ${index+1}: ${nodeTitle(node)}\nType: ${node.type}\n${nodeAIContent(node).slice(0,30000)}`).join("\n\n---\n\n"):"No connected inputs.";
    const response=await providerFetch(aiSettings,"/chat/completions",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({model:aiSettings.model,messages:[{role:"system",content:"You generate one useful Markdown note from connected JSON Canvas inputs. Follow the operator instructions. Return only the note Markdown, without code fences, commentary, or JSON. Do not include HTML or scripts."},{role:"user",content:`Operator: ${config.title}\nInstructions:\n${config.prompt}\n\nConnected inputs:\n${inputText}`}],temperature:.3,max_tokens:2200})}),body=await response.json();
    let generated=providerMessageContent(body.choices?.[0]?.message?.content);if(!/^#\s/m.test(generated))generated=`# ${config.title} — output\n\n${generated}`;
    const before=aiCardSignatures();let outputEdge=(documentData.edges||[]).find(edge=>edge.fromNode===card.id&&edge.label==="AI output"),output=outputEdge&&documentData.nodes.find(node=>node.id===outputEdge.toNode&&node.type==="text");
    if(!output){output={id:uid("node"),type:"text",x:card.x+card.width+90,y:card.y,width:380,height:240,color:"5",text:generated};documentData.nodes.push(output);outputEdge={id:uid("edge"),fromNode:card.id,fromSide:"right",toNode:output.id,toSide:"left",toEnd:"arrow",color:"5",label:"AI output"};documentData.edges.push(outputEdge);}else output.text=generated;
    state.lastSignature=signature;state.status=`Updated ${new Date().toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"})}`;scheduleSave();render();scheduleChangedAICards(before);toast(`${config.title} updated its output`);
  }catch(error){state.status=`Error · ${error.message.slice(0,55)}`;toast("AI operator failed");}
  finally{state.running=false;const pending=state.pending;state.pending=false;aiCardRuntime.set(cardId,state);renderNodes();if(pending)scheduleAICard(cardId,250);}
}

window.orbitCanvas={getDocument:()=>clone(documentData),getWorkspace:()=>clone(workspace),getCurrentCanvas:()=>({id:currentCanvasId,title:canvasRecord().title,trail:canvasTrail().map(record=>({id:record.id,title:record.title}))}),getSummary:canvasSummary,validateOperations:validateCanvasOperations,applyOperations:applyCanvasOperations,runAICard,createSubcanvas,switchCanvas};
applyCanvasTheme(localStorage.getItem("orbit-canvas-theme")||"default");updateProviderUI();

$$("[data-add]").forEach(button=>button.onclick=()=>button.dataset.add==="ai-note"?openAINoteDialog():addNode(button.dataset.add));
$("#newGroup").onclick=()=>addNode("group");$("#newCanvas").onclick=()=>createSubcanvas();
$$(".nav-item[data-filter]").forEach(button=>button.onclick=()=>{activeFilter=button.dataset.filter;$$(".nav-item[data-filter]").forEach(b=>b.classList.toggle("active",b===button));renderNodes();renderEdges();});
$$(".tool").forEach(button=>button.onclick=()=>{const tool=button.dataset.tool;if(tool==="note")setTool("note");else setTool(tool);});
$("#zoomIn").onclick=()=>setZoom(camera.zoom*1.2);$("#zoomOut").onclick=()=>setZoom(camera.zoom/1.2);$("#zoomLabel").onclick=()=>setZoom(1);$("#fitView").onclick=fitView;
$("#exportButton").onclick=exportCanvas;$("#importButton").onclick=()=>$("#fileInput").click();$("#fileInput").onchange=e=>{if(e.target.files[0])importCanvas(e.target.files[0]);e.target.value="";};
$("#sidebarToggle").onclick=()=>shell.classList.toggle("sidebar-closed");
$("#assistantButton").onclick=()=>setAssistantOpen(!$("#aiPanel").classList.contains("open"));$("#closeAssistant").onclick=()=>setAssistantOpen(false);$("#openAISettings").onclick=openAISettings;
$("#aiForm").onsubmit=event=>{event.preventDefault();const input=$("#aiPrompt"),prompt=input.value;input.value="";runAssistant(prompt);};
$("#aiPrompt").onkeydown=event=>{if(event.key==="Enter"&&!event.shiftKey){event.preventDefault();$("#aiForm").requestSubmit();}};
$$(".ai-suggestions button").forEach(button=>button.onclick=()=>runAssistant(button.textContent));
$("#closeAINote").onclick=$("#cancelAINote").onclick=()=>$("#aiNoteDialog").close();
$("#aiNoteForm").onsubmit=event=>{event.preventDefault();const prompt=$("#aiNotePrompt").value.trim();if(prompt)createAINote(prompt);};
$("#closeAISettings").onclick=$("#cancelAISettings").onclick=()=>$("#aiSettingsDialog").close();
$("#toggleAIKey").onclick=()=>{const input=$("#aiAPIKey"),show=input.type==="password";input.type=show?"text":"password";$("#toggleAIKey").textContent=show?"Hide":"Show";};
$("#aiSettingsForm").onsubmit=event=>{event.preventDefault();if(!event.currentTarget.reportValidity())return;try{const settings=settingsFromForm();if(!settings.model||!settings.apiKey)throw new Error("Model and API key are required");persistAISettings(settings);$("#aiSettingsDialog").close();toast(`Connected to ${settings.model}`);}catch(error){setSettingsResult(error.message,"error");}};
$("#testAIProvider").onclick=async()=>{const form=$("#aiSettingsForm");if(!form.reportValidity())return;const button=$("#testAIProvider");try{const settings=settingsFromForm();button.disabled=true;setSettingsResult("Testing direct browser connection…");setSettingsResult(await testAIProvider(settings),"success");}catch(error){setSettingsResult(error.message,"error");}finally{button.disabled=false;}};
$("#clearAIProvider").onclick=()=>{persistAISettings({...aiSettings,apiKey:"",rememberKey:false});localStorage.removeItem(AI_SECRET_KEY);sessionStorage.removeItem(AI_SECRET_KEY);$("#aiSettingsDialog").close();toast("Using local canvas tools");};
$("#canvasTitle").value=canvasRecord().title;$("#canvasTitle").oninput=e=>{canvasRecord().title=e.target.value||"Untitled canvas";scheduleSave();renderWorkspaceNavigation();};
$("#resetDemo").onclick=()=>{if(confirm("Reset the whole space to the demo? Every nested canvas and local change will be replaced.")){workspace=freshWorkspace(clone(demoCanvas));currentCanvasId=workspace.rootId;documentData=workspace.canvases[currentCanvasId].document;camera={x:80,y:55,zoom:.78};selected=null;$("#canvasTitle").value=canvasRecord().title;scheduleSave();render();fitView();toast("Demo space restored");}};
$("#minimap").onclick=fitView;

window.addEventListener("keydown",event=>{
  if (["INPUT","TEXTAREA","SELECT"].includes(event.target.tagName)) return;
  if(event.code==="Space"){spaceDown=true;event.preventDefault();}
  if(event.altKey&&event.key==="ArrowUp"&&canvasRecord().parentId){event.preventDefault();leaveSubcanvas();return;}
  if(event.key==="Enter"&&selected?.kind==="node"){const node=documentData.nodes.find(item=>item.id===selected.id),subcanvasId=subcanvasIdFromNode(node);if(subcanvasId){event.preventDefault();enterSubcanvas(subcanvasId);return;}}
  if((event.key==="Delete"||event.key==="Backspace")&&selected)deleteSelection();
  if(event.key.toLowerCase()==="v")setTool("select");if(event.key.toLowerCase()==="h")setTool("pan");if(event.key.toLowerCase()==="c")setTool("connect");if(event.key.toLowerCase()==="n")setTool("note");
  if(event.key==="0")fitView();if(event.key==="+"||event.key==="=")setZoom(camera.zoom*1.2);if(event.key==="-")setZoom(camera.zoom/1.2);
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==="s"){event.preventDefault();exportCanvas();}
});
window.addEventListener("keyup",event=>{if(event.code==="Space")spaceDown=false;});
window.addEventListener("resize",()=>{applyCamera();});
window.addEventListener("beforeunload",persistWorkspace);

render();
setTimeout(fitView,50);
