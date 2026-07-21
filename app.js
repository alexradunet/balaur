/* Orbit is deliberately dependency-free. The persisted/exported document follows JSON Canvas 1.0. */
const COLORS = {
  "1": "#ff7b78", "2": "#efa66a", "3": "#e9d56b",
  "4": "#7ee0a1", "5": "#64cbd0", "6": "#a78bfa"
};

const demoCanvas = {
  nodes: [
    { id:"g-week", type:"group", x:0, y:0, width:690, height:600, label:"This week", color:"4" },
    { id:"g-horizon", type:"group", x:750, y:0, width:590, height:600, label:"On the horizon", color:"6" },
    { id:"n-focus", type:"text", x:35, y:48, width:620, height:145, color:"3", text:"# A calmer, more intentional week\nProtect the mornings, move the important work forward, and leave enough space to notice life.\n\n`WEEK 29`  ·  **3 priorities**" },
    { id:"n-project", type:"text", x:35, y:230, width:295, height:240, color:"6", text:"# Ship the portfolio refresh\nMake the work feel as considered as the work itself.\n\n- [x] Finalize case study copy\n- [ ] Record walkthrough\n- [ ] Publish and share\n\nProgress: 66%" },
    { id:"n-habit", type:"text", x:360, y:230, width:295, height:150, color:"4", text:"# Morning pages\nWrite three pages before opening any inputs.\n\n**5 day streak**  ·  07:00" },
    { id:"n-idea", type:"text", x:360, y:410, width:295, height:145, color:"2", text:"# Sunday without screens\nA small experiment: books, a long walk, cooking, and no glowing rectangles until evening." },
    { id:"n-goal", type:"text", x:785, y:48, width:520, height:190, color:"1", text:"# Run a comfortable 10K\nBuild patiently. Finish feeling like there was a little more in the tank.\n\n- [x] Choose a training plan\n- [ ] Three easy runs / week\n- [ ] Race day · Sep 14\n\nProgress: 35%" },
    { id:"n-reading", type:"text", x:785, y:280, width:250, height:160, color:"5", text:"# Reading next\n- [ ] Four Thousand Weeks\n- [ ] The Creative Act\n- [ ] Braiding Sweetgrass" },
    { id:"n-trip", type:"link", x:1065, y:280, width:240, height:160, color:"6", url:"https://www.openstreetmap.org" },
    { id:"n-note", type:"text", x:785, y:470, width:520, height:85, color:"3", text:"## Remember\nA plan is useful, but a good day is the point." }
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

let documentData = loadDocument();
let camera = { x: 80, y: 55, zoom: .78 };
let selected = null;
let currentTool = "select";
let connectSource = null;
let activeFilter = "all";
let spaceDown = false;
let saveTimer;

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

function isCanvas(data) {
  return data && typeof data === "object" &&
    (!data.nodes || Array.isArray(data.nodes)) && (!data.edges || Array.isArray(data.edges)) &&
    (data.nodes || []).every(n => n && typeof n.id === "string" && ["text","file","link","group"].includes(n.type) && [n.x,n.y,n.width,n.height].every(Number.isInteger)) &&
    (data.edges || []).every(e => e && typeof e.id === "string" && typeof e.fromNode === "string" && typeof e.toNode === "string");
}

function scheduleSave() {
  $("#saveState").innerHTML = "<i></i> Saving…";
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    localStorage.setItem("orbit-canvas-v1", JSON.stringify(documentData));
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
    if (!line.trim()) continue;
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
      $(".node-accent", element).remove();
    } else if (node.type === "text") {
      content.innerHTML = `<div class="node-kicker">${textMeta(node)}</div>${markdownToHTML(node.text)}`;
    } else if (node.type === "link") {
      let linkTitle = "Saved link";
      try { linkTitle = new URL(node.url).hostname.replace(/^www\./, ""); } catch (_) {}
      content.innerHTML = `<div class="node-kicker">LINK</div><h3>${escapeHTML(linkTitle)}</h3><p>Open this resource in a new tab.</p><a class="node-link" href="${safeURL(node.url)}" target="_blank" rel="noreferrer">${escapeHTML(node.url)} ↗</a>`;
    } else if (node.type === "file") {
      content.innerHTML = `<div class="node-kicker">FILE</div><div class="file-preview">▧</div><h3>${escapeHTML(node.file.split("/").pop())}</h3><p>${escapeHTML(node.subpath || node.file)}</p>`;
    }
    element.addEventListener("pointerdown", event => nodePointerDown(event, node));
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
  applyCamera(); renderEdges(); renderNodes(); renderInspector();
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
  if (event.button !== 0 || event.target.closest("a")) return;
  event.stopPropagation();
  if (currentTool === "connect") {
    if (!connectSource) { connectSource=node.id; toast("Now choose a destination"); }
    else if (connectSource !== node.id) {
      documentData.edges ||= [];
      documentData.edges.push({id:uid("edge"),fromNode:connectSource,toNode:node.id,toEnd:"arrow"});
      connectSource=null; setTool("select"); scheduleSave(); toast("Nodes connected");
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
    const up=()=>{canvas.classList.remove("panning");window.removeEventListener("pointermove",move)};
    window.addEventListener("pointermove",move);window.addEventListener("pointerup",up,{once:true}); return;
  }
  if (event.target === canvas || event.target === world || event.target === nodeLayer) {
    selected=null; connectSource=null; shell.classList.remove("inspector-open"); render();
    if (currentTool === "note") { const p=canvasPoint(event.clientX,event.clientY); addNode("note",p); setTool("select"); }
  }
});
canvas.addEventListener("dblclick", event => {
  if (event.target===canvas || event.target===world || event.target===nodeLayer) addNode("note",canvasPoint(event.clientX,event.clientY));
});
canvas.addEventListener("wheel", event => {
  event.preventDefault();
  const rect=canvas.getBoundingClientRect(), sx=event.clientX-rect.left, sy=event.clientY-rect.top;
  const worldX=(sx-camera.x)/camera.zoom, worldY=(sy-camera.y)/camera.zoom;
  const factor=Math.exp(-event.deltaY*.0012), next=Math.max(.2,Math.min(2.5,camera.zoom*factor));
  camera.x=sx-worldX*next; camera.y=sy-worldY*next; camera.zoom=next; applyCamera();
},{passive:false});

function selectItem(kind,id) {
  selected={kind,id}; shell.classList.add("inspector-open"); render();
}
function setTool(tool) {
  currentTool=tool; connectSource=tool==="connect"?connectSource:null;
  $$(".tool").forEach(b=>b.classList.toggle("active",b.dataset.tool===tool));
  canvas.classList.toggle("tool-pan",tool==="pan"); renderNodes();
}

function addNode(kind, point) {
  const center = point || canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2);
  const presets={
    note:{type:"text",color:"2",width:260,height:150,text:"# New thought\nStart writing here…"},
    goal:{type:"text",color:"1",width:300,height:190,text:"# A meaningful goal\nWhat would make this worth doing?\n\n- [ ] Define the first step\n\nProgress: 0%"},
    habit:{type:"text",color:"4",width:280,height:145,text:"# New daily practice\nMake it small enough to begin today."},
    project:{type:"text",color:"6",width:300,height:210,text:"# Untitled project\nDescribe the outcome, not just the activity.\n\n- [ ] First milestone\n- [ ] Next milestone\n\nProgress: 0%"},
    group:{type:"group",color:"5",width:620,height:430,label:"New area"}
  };
  const preset=presets[kind]||presets.note;
  const node={id:uid("node"),...preset,x:Math.round(center.x-preset.width/2),y:Math.round(center.y-preset.height/2)};
  documentData.nodes ||= [];
  if (kind==="group") documentData.nodes.unshift(node); else documentData.nodes.push(node);
  selected={kind:"node",id:node.id}; shell.classList.add("inspector-open"); scheduleSave(); render();
}

function renderInspector() {
  const panel=$("#inspector");
  if (!selected) { panel.innerHTML='<div class="inspector-empty"><span>↖</span><h3>Nothing selected</h3><p>Select a card or connection to edit its details.</p></div>'; return; }
  const item = selected.kind==="node" ? documentData.nodes.find(n=>n.id===selected.id) : documentData.edges.find(e=>e.id===selected.id);
  if (!item) { selected=null; shell.classList.remove("inspector-open"); renderInspector(); return; }
  const colorButtons=Object.entries(COLORS).map(([key,value])=>`<button type="button" class="color-choice ${item.color===key?"active":""}" data-color="${key}" style="background:${value}" aria-label="Color ${key}"></button>`).join("");
  if (selected.kind==="node") {
    let contentField="";
    if (item.type==="text") contentField=`<label class="field"><span>Markdown</span><textarea data-key="text">${escapeHTML(item.text)}</textarea></label>`;
    if (item.type==="link") contentField=`<label class="field"><span>URL</span><input data-key="url" value="${escapeHTML(item.url)}"></label>`;
    if (item.type==="file") contentField=`<label class="field"><span>File path</span><input data-key="file" value="${escapeHTML(item.file)}"></label><label class="field"><span>Subpath</span><input data-key="subpath" value="${escapeHTML(item.subpath||"")}"></label>`;
    if (item.type==="group") contentField=`<label class="field"><span>Label</span><input data-key="label" value="${escapeHTML(item.label||"")}"></label><label class="field"><span>Background path</span><input data-key="background" value="${escapeHTML(item.background||"")}"></label>`;
    panel.innerHTML=`<div class="inspector-head"><h3>${item.type[0].toUpperCase()+item.type.slice(1)} node</h3><button class="close-inspector">×</button></div><form class="inspector-form">${contentField}<div class="field-row"><label class="field"><span>X</span><input type="number" data-key="x" value="${item.x}"></label><label class="field"><span>Y</span><input type="number" data-key="y" value="${item.y}"></label></div><div class="field-row"><label class="field"><span>Width</span><input type="number" data-key="width" value="${item.width}"></label><label class="field"><span>Height</span><input type="number" data-key="height" value="${item.height}"></label></div><label class="field"><span>Color preset</span><div class="color-list">${colorButtons}</div></label><button type="button" class="danger-btn">Delete node</button></form>`;
  } else {
    panel.innerHTML=`<div class="inspector-head"><h3>Connection</h3><button class="close-inspector">×</button></div><form class="inspector-form"><label class="field"><span>Label</span><input data-key="label" value="${escapeHTML(item.label||"")}"></label><div class="field-row"><label class="field"><span>From side</span><select data-key="fromSide">${sideOptions(item.fromSide)}</select></label><label class="field"><span>To side</span><select data-key="toSide">${sideOptions(item.toSide)}</select></label></div><label class="field"><span>Color preset</span><div class="color-list">${colorButtons}</div></label><button type="button" class="danger-btn">Delete connection</button></form>`;
  }
  $(".close-inspector",panel).onclick=()=>{selected=null;shell.classList.remove("inspector-open");render();};
  $$("[data-key]",panel).forEach(input=>input.addEventListener("input",()=>{
    const key=input.dataset.key; item[key]=input.type==="number"?Math.round(Number(input.value)):input.value;
    if (input.tagName==="SELECT" && !input.value) delete item[key];
    scheduleSave(); renderNodes(); renderEdges(); renderMinimap();
  }));
  $$(".color-choice",panel).forEach(button=>button.onclick=()=>{item.color=button.dataset.color;scheduleSave();render();});
  $(".danger-btn",panel).onclick=deleteSelection;
}
function sideOptions(value) { return ["","top","right","bottom","left"].map(s=>`<option value="${s}" ${value===s?"selected":""}>${s||"Auto"}</option>`).join(""); }
function deleteSelection() {
  if (!selected) return;
  if (selected.kind==="node") {
    documentData.nodes=documentData.nodes.filter(n=>n.id!==selected.id);
    documentData.edges=(documentData.edges||[]).filter(e=>e.fromNode!==selected.id&&e.toNode!==selected.id);
  } else documentData.edges=documentData.edges.filter(e=>e.id!==selected.id);
  selected=null;shell.classList.remove("inspector-open");scheduleSave();render();toast("Deleted");
}

function updateCounts() {
  const nodes=(documentData.nodes||[]).filter(n=>n.type!=="group");
  $("#allCount").textContent=nodes.length;
  [["goalCount","1"],["habitCount","4"],["projectCount","6"],["ideaCount","2"]].forEach(([id,c])=>$("#"+id).textContent=nodes.filter(n=>n.color===c).length);
}
function renderMinimap() {
  const mini=$("#miniWorld"); if (!mini || !documentData.nodes?.length) return;
  const bounds=getBounds(), pad=8, mw=128-pad*2,mh=82-pad*2, scale=Math.min(mw/bounds.width,mh/bounds.height,.12);
  const ox=pad+(mw-bounds.width*scale)/2-bounds.minX*scale, oy=pad+(mh-bounds.height*scale)/2-bounds.minY*scale;
  mini.innerHTML=documentData.nodes.map(n=>`<i class="mini-node ${n.type==="group"?"group":""}" style="left:${ox+n.x*scale}px;top:${oy+n.y*scale}px;width:${Math.max(2,n.width*scale)}px;height:${Math.max(2,n.height*scale)}px;background-color:${n.type==="group"?"transparent":colorValue(n.color)}"></i>`).join("");
  const view=$("#miniViewport"), worldLeft=-camera.x/camera.zoom, worldTop=-camera.y/camera.zoom;
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

$$("[data-add]").forEach(button=>button.onclick=()=>addNode(button.dataset.add));
$("#newGroup").onclick=()=>addNode("group");
$$(".nav-item[data-filter]").forEach(button=>button.onclick=()=>{activeFilter=button.dataset.filter;$$(".nav-item[data-filter]").forEach(b=>b.classList.toggle("active",b===button));renderNodes();renderEdges();});
$$(".tool").forEach(button=>button.onclick=()=>{const tool=button.dataset.tool;if(tool==="note")setTool("note");else setTool(tool);});
$("#zoomIn").onclick=()=>setZoom(camera.zoom*1.2);$("#zoomOut").onclick=()=>setZoom(camera.zoom/1.2);$("#zoomLabel").onclick=()=>setZoom(1);$("#fitView").onclick=fitView;
$("#exportButton").onclick=exportCanvas;$("#importButton").onclick=()=>$("#fileInput").click();$("#fileInput").onchange=e=>{if(e.target.files[0])importCanvas(e.target.files[0]);e.target.value="";};
$("#sidebarToggle").onclick=()=>shell.classList.toggle("sidebar-closed");
$("#canvasTitle").value=localStorage.getItem("orbit-title")||"Life OS — Summer";$("#canvasTitle").oninput=e=>localStorage.setItem("orbit-title",e.target.value);
$("#resetDemo").onclick=()=>{if(confirm("Reset the canvas to the demo? Your local changes will be replaced.")){documentData=clone(demoCanvas);selected=null;scheduleSave();render();fitView();toast("Demo restored");}};
$("#minimap").onclick=fitView;

window.addEventListener("keydown",event=>{
  if (["INPUT","TEXTAREA","SELECT"].includes(event.target.tagName)) return;
  if(event.code==="Space"){spaceDown=true;event.preventDefault();}
  if((event.key==="Delete"||event.key==="Backspace")&&selected)deleteSelection();
  if(event.key.toLowerCase()==="v")setTool("select");if(event.key.toLowerCase()==="h")setTool("pan");if(event.key.toLowerCase()==="c")setTool("connect");if(event.key.toLowerCase()==="n")setTool("note");
  if(event.key==="0")fitView();if(event.key==="+"||event.key==="=")setZoom(camera.zoom*1.2);if(event.key==="-")setZoom(camera.zoom/1.2);
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==="s"){event.preventDefault();exportCanvas();}
});
window.addEventListener("keyup",event=>{if(event.code==="Space")spaceDown=false;});
window.addEventListener("resize",()=>{applyCamera();});

render();
setTimeout(fitView,50);
