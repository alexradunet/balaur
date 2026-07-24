/* Balaur is dependency-free. Every canvas document follows JSON Canvas 1.0; hierarchy and cameras live in a local workspace sidecar. */
import { IndexedDbVault } from "./storage/indexeddb-vault.js";
import { WorkspaceStore, hasWorkspace, canvasPathFor } from "./storage/workspace-vault.js";
import { isCanvas } from "./storage/canvas-validate.js";
import { ComponentCardCatalog } from "./storage/component-card-catalog.js";
import { FileComponentCardRepository } from "./storage/component-card-repository.js";
import { WidgetCatalog } from "./storage/widget-catalog.js";
import { FileWidgetRepository } from "./storage/widget-repository.js";
import { componentCardPath, slugify } from "./storage/vault-path.js";
import { LifeIndexer } from "./storage/life-indexer.js";
import { parseEntity } from "./storage/entity-codec.js";
import { MemoryIndex } from "./storage/memory-index.js";
import { LifeQuery } from "./storage/life-query.js";
import { FileTaskRepository } from "./storage/task-repository.js";
import { FileJournalRepository, journalPath } from "./storage/journal-event-repository.js";
import { exportBundle, importBundle, serializeBundle, assertCompleteExport } from "./storage/workspace-backup.js";
import { auditIndex } from "./storage/index-integrity.js";
import { assertPlainDataTree, describeGeneratedOperation, recoverGeneratedPlacementFailure, validateGeneratedOperation } from "./ai/generated-operations.js";
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

const NOTE_MARKERS={inbox:"<!-- orbit:inbox -->",reference:"<!-- orbit:reference -->"};
const DORMANT_NODE_COLOR="#6c757d";
const STARTER_TASK_ID="task-citybreak";
const STARTER_TASK_PATH="tasks/choose-dates-for-the-autumn-trip-task-citybreak.md";
function createGraphStarterWorkspace(){
  const rootDocument={nodes:[
    {id:"home-guide",type:"text",x:0,y:-220,width:1180,height:170,color:"3",text:"# Start here — your life, as a graph\nHome is the entry point. Four hubs hang off it; structure emerges from labelled connections, not folders.\n\n- Inbox — capture pending processing\n- Projects — committed efforts with a finish line\n- Wiki — durable reference\n- Archive — dormant and completed"},
    {id:"portal-inbox",type:"file",x:0,y:20,width:360,height:240,color:"2",file:"canvases/inbox.canvas"},
    {id:"portal-projects",type:"file",x:410,y:20,width:360,height:240,color:"6",file:"canvases/projects.canvas"},
    {id:"portal-wiki",type:"file",x:0,y:300,width:360,height:240,color:"5",file:"canvases/wiki.canvas"},
    {id:"portal-archive",type:"file",x:410,y:300,width:360,height:240,color:"3",file:"canvases/archive.canvas"},
  ],edges:[]};
  const result=freshWorkspace(rootDocument),root=result.canvases[result.rootId];
  root.title="Home";root.document=rootDocument;root.camera=null;

  const today=localDateISO(),journalFile=journalPath(today);
  const hub=(id,title,path,portalNodeId,doc)=>{result.canvases[id]={id,title,parentId:result.rootId,portalNodeId,path,document:doc,camera:null,kind:"hub"};};
  const inboxDoc={nodes:[
    {id:"inbox-guide",type:"text",x:0,y:-160,width:760,height:120,color:"3",text:"# Inbox\nCapture quick notes here, then process them into a project or the wiki. A healthy inbox trends toward empty."},
    {id:"inbox-trip",type:"text",x:0,y:20,width:340,height:200,color:"2",text:`${NOTE_MARKERS.inbox}\n# Autumn city break idea\nDecide dates and a rough budget, then turn it into a real project.`},
    {id:"inbox-to-citybreak",type:"file",x:420,y:20,width:340,height:200,color:"6",file:"canvases/city-break.canvas"},
  ],edges:[
    {id:"e-inbox-filed",fromNode:"inbox-trip",fromSide:"right",toNode:"inbox-to-citybreak",toSide:"left",toEnd:"arrow",color:"6",label:"filed-to"},
  ]};
  const projectsDoc={nodes:[
    {id:"projects-guide",type:"text",x:0,y:-160,width:760,height:120,color:"3",text:"# Projects\nCommitted efforts with a finish line. Each project is its own canvas holding tasks, notes, and sub-canvases."},
    {id:"projects-citybreak",type:"file",x:0,y:20,width:360,height:240,color:"6",file:"canvases/city-break.canvas"},
  ],edges:[]};
  const cityBreakDoc={nodes:[
    {id:"cb-note",type:"text",x:0,y:0,width:360,height:200,color:"6",text:"# Autumn city break\nThree days, one city, room to wander. Finish line: transport and accommodation booked."},
    {id:"cb-task",type:"file",x:420,y:0,width:340,height:200,color:"5",file:STARTER_TASK_PATH},
  ],edges:[
    {id:"e-cb-partof",fromNode:"cb-task",fromSide:"left",toNode:"cb-note",toSide:"right",toEnd:"arrow",color:"6",label:"part-of"},
  ]};
  const wikiDoc={nodes:[
    {id:"wiki-guide",type:"text",x:0,y:-180,width:760,height:120,color:"3",text:"# Wiki\nDurable reference and responsibilities. Pages interlink; this is the long-term memory of your life."},
    {id:"wiki-budget",type:"text",x:0,y:0,width:340,height:200,color:"5",text:`${NOTE_MARKERS.reference}\n# Monthly budget\nFixed costs, everyday spending, and fun each get a limit. Reconcile monthly.`},
    {id:"wiki-subscriptions",type:"text",x:420,y:0,width:340,height:200,color:"5",text:`${NOTE_MARKERS.reference}\n# Subscriptions\nReview recurring costs quarterly before they become invisible.`},
    {id:"wiki-journal",type:"file",x:840,y:0,width:320,height:200,color:"3",file:journalFile},
  ],edges:[
    {id:"e-wiki-relates",fromNode:"wiki-subscriptions",fromSide:"left",toNode:"wiki-budget",toSide:"right",toEnd:"arrow",color:"5",label:"relates-to"},
  ]};
  const archiveDoc={nodes:[
    {id:"archive-guide",type:"text",x:0,y:-160,width:760,height:120,color:"3",text:"# Archive\nDormant and completed work. Nothing is deleted; things are filed here when they are done or paused."},
    {id:"archive-portfolio",type:"text",x:0,y:20,width:360,height:200,color:DORMANT_NODE_COLOR,text:`${NOTE_MARKERS.reference}\n# Portfolio refresh (completed)\nShipped and shared. Kept for the record.`},
  ],edges:[]};

  hub("hub-inbox","Inbox","canvases/inbox.canvas","portal-inbox",inboxDoc);
  hub("hub-projects","Projects","canvases/projects.canvas","portal-projects",projectsDoc);
  hub("hub-wiki","Wiki","canvases/wiki.canvas","portal-wiki",wikiDoc);
  hub("hub-archive","Archive","canvases/archive.canvas","portal-archive",archiveDoc);
  result.canvases["project-city-break"]={id:"project-city-break",title:"City break",parentId:"hub-projects",portalNodeId:"projects-citybreak",path:"canvases/city-break.canvas",document:cityBreakDoc,camera:null,kind:"project"};
  return normalizeWorkspace(result);
}
async function seedGraphStarterEntities(){
  // Task file at the exact path the starter's cb-task node already references.
  // No canvasId: the placement node exists in the City break canvas already.
  if(taskRepository){
    try{
      await taskRepository.createTask({
        id:STARTER_TASK_ID,path:STARTER_TASK_PATH,
        title:"Choose dates for the autumn trip",
        body:"Check the calendar and agree on a realistic budget window.",
        status:"inbox",
      });
    }catch(_){/* already seeded */}
  }
  // Journal file for today at the path the starter's wiki-journal node references.
  if(journalRepository){
    const today=localDateISO();
    try{await journalRepository.getJournal(today);}
    catch(_){await journalRepository.createJournal({localDate:today,body:"Welcome to Balaur. This is your daily note — a quiet place to think out loud. Open Today to edit it, or place it on a canvas."});}
  }
}

const $ = (selector, root=document) => root.querySelector(selector);
const $$ = (selector, root=document) => [...root.querySelectorAll(selector)];
const clone = value => JSON.parse(JSON.stringify(value));
const uid = prefix => `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2,7)}`;

const WORKSPACE_KEY="orbit-workspace-v1",ROOT_CANVAS_ID="canvas-root";
// Workspace globals are populated by the asynchronous vault-first boot
// (bootCanvasApp). They begin as a valid placeholder workspace so module-eval
// event wiring (closures) can attach safely before boot completes; real reads
// happen at call-time, after boot reassigns these bindings.
let workspace={version:1,rootId:ROOT_CANVAS_ID,activeId:ROOT_CANVAS_ID,canvases:{[ROOT_CANVAS_ID]:{id:ROOT_CANVAS_ID,title:"Loading…",parentId:null,portalNodeId:null,path:null,document:{nodes:[],edges:[]},camera:{x:80,y:55,zoom:.78}}}};
let currentCanvasId=ROOT_CANVAS_ID;
let documentData=workspace.canvases[ROOT_CANVAS_ID].document;
let camera={x:80,y:55,zoom:.78};
let selected = null;
let currentTool = "select";
let connectSource = null;
let connectSourceSide = null;
let activeFilter = "all";
let activeAppView = "canvas";
let spaceDown = false;
let saveTimer;
let pendingSave=Promise.resolve();
let mutationQueue=Promise.resolve();
const aiCardRuntime=new Map();
let renderedSelectionKey = null;
const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)");

function updateWithViewTransition(update) {
  if (reducedMotion.matches || !document.startViewTransition) {
    update();
    return null;
  }
  return document.startViewTransition(update);
}
let vaultStore=null;
let vaultReady=null;
let lifeIndex=null;
let lifeIndexer=null;
let lifeQuery=null;
let taskRepository=null;
let journalRepository=null;
let componentCardCatalog=null;
let componentCardRepository=null;
let widgetCatalog=null;
let widgetRepository=null;
let canonicalWritable=false;
const aiFileContentCache=new Map();
const taskUpdateTimers=new Map();

const canvas = $("#canvas");
const world = $("#world");
const nodeLayer = $("#nodes");
const edgeLayer = $("#edges");
const shell = $(".app-shell");
const narrowShell = matchMedia("(max-width: 850px)");
const syncNarrowShell = event => shell.classList.toggle("sidebar-closed", event.matches);
syncNarrowShell(narrowShell);
narrowShell.addEventListener("change", syncNarrowShell);

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
  return {version:1,rootId:ROOT_CANVAS_ID,activeId:ROOT_CANVAS_ID,canvases:{[ROOT_CANVAS_ID]:{id:ROOT_CANVAS_ID,title:localStorage.getItem("orbit-title")||"Life OS — Summer",parentId:null,portalNodeId:null,path:null,document,camera:{x:80,y:55,zoom:.78}}}};
}
function normalizeWorkspace(parsed){
  delete parsed.johnnyDecimal; // legacy JD index (ADR-0003)
  for(const record of Object.values(parsed.canvases)){
    delete record.jdCode; delete record.jdTitle; delete record.jdKind;
    if(record.kind!=="hub"&&record.kind!=="project")delete record.kind;
    if(record.id===parsed.rootId){record.path=null;continue;}
    if(!record.path){const parent=parsed.canvases[record.parentId],portal=parent?.document.nodes?.find(node=>node.id===record.portalNodeId);record.path=portal?.file||`canvases/${record.id}.canvas`;}
  }
  return parsed;
}
function loadWorkspace(){
  try{
    const parsed=JSON.parse(localStorage.getItem(WORKSPACE_KEY)||"null"),canvases=parsed?.canvases;
    if(parsed?.version===1&&canvases&&typeof canvases==="object"&&Object.keys(canvases).length&&Object.values(canvases).every(record=>record&&typeof record.id==="string"&&typeof record.title==="string"&&isCanvas(record.document))){
      parsed.rootId=canvases[parsed.rootId]?parsed.rootId:Object.keys(canvases)[0];parsed.activeId=canvases[parsed.activeId]?parsed.activeId:parsed.rootId;return normalizeWorkspace(parsed);
    }
  }catch(_){}
  return localStorage.getItem("orbit-canvas-v1")?freshWorkspace():createGraphStarterWorkspace();
}


function saveCurrentCanvasState(){
  const record=workspace.canvases[currentCanvasId];if(!record)return;record.document=documentData;record.camera={...camera};const value=$("#canvasTitle")?.value.trim();if(value)record.title=value;
}
function enqueueMutation(task){
  const run=mutationQueue.then(task,task);
  mutationQueue=run.catch(()=>{});
  return run;
}
async function saveWorkspaceNow(){
  saveCurrentCanvasState();workspace.activeId=currentCanvasId;
  if(!vaultStore)return;
  const fromRevision=Number(lifeIndex?.getIndexState("indexedRevision")||0);
  await vaultStore.save(workspace);
  await lifeIndexer?.reconcileWarm(fromRevision);
  if(lifeIndexer){const stats=lifeIndexer.stats();setIndexStatus(`Files · ${stats.sourceFiles} indexed`,`${stats.tasks} tasks · ${stats.habits} habits · ${stats.diagnostics} diagnostics`);}
}
function persistWorkspace(){
  const operation=enqueueMutation(saveWorkspaceNow);
  pendingSave=operation;
  operation.then(()=>{},()=>{}).then(()=>{if(pendingSave===operation)pendingSave=Promise.resolve();});
  return operation;
}
function markSaveResult(promise){
  promise.then(()=>{$("#saveState").innerHTML="<i></i> Saved locally";},error=>{
    console.warn("Could not persist the canonical vault workspace",error);
    $("#saveState").innerHTML="<i></i> Save failed";
    toast("Could not save the canonical files; your changes are not durable");
  });
  return promise;
}
function scheduleSave() {
  if (!canonicalWritable) { $("#saveState").innerHTML = "<i></i> Read-only"; return; }
  $("#saveState").innerHTML = "<i></i> Saving…";
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => { saveTimer=null; markSaveResult(persistWorkspace()); }, 350);
}
async function flushPendingWorkspaceEdits(){
  if(saveTimer!==undefined&&saveTimer!==null){clearTimeout(saveTimer);saveTimer=null;markSaveResult(persistWorkspace());}
  await pendingSave;
  await mutationQueue;
}
async function reloadCanvasDocuments(canvasIds){
  if(!vaultStore)return;
  for(const id of new Set(canvasIds)){
    const record=workspace.canvases[id];if(!record||record.readOnly)continue;
    const path=canvasPathFor(record,workspace.rootId);
    const parsed=JSON.parse(await vaultStore.vault.read(path));
    if(!isCanvas(parsed))throw new Error(`Canonical canvas is invalid: ${record.path}`);
    const stat=await vaultStore.vault.stat(path);if(stat)vaultStore.hashes.set(path,stat.hash);
    record.document=parsed;
    if(id===currentCanvasId)documentData=parsed;
  }
}

function setIndexStatus(message, detail = message) {
  const state = $("#lifeIndexStatus");
  if (state) {
    state.textContent = message;
    state.title = detail;
    const wrap = state.closest(".storage-state");
    if (wrap) { wrap.classList.remove("is-indexing"); wrap.title = detail; }
  }
}
function setCanonicalWritable(writable, message = "") {
  canonicalWritable = writable;
  document.documentElement.toggleAttribute("data-canonical-read-only", !writable);
  const selectors = ["[data-add]", "#newGroup", "#newCanvas", "#createTaskButton", "#newTodayTask", "#todayQuickAdd input", "#todayQuickAdd button", "#resetDemo", "#importButton"];
  for (const selector of selectors) $$(selector).forEach((control) => { control.disabled = !writable; control.title = writable ? "" : (message || "Canonical files are read-only until repaired or restored."); });
  if(!writable&&message)setIndexStatus("Files read-only · repair/export required",message);
}
function canvasIdFromPath(path) {
  const record = Object.values(workspace.canvases).find(item => canvasPathFor(item, workspace.rootId) === path);
  return record?.id || String(path).split("/").pop().replace(/\.canvas$/, "");
}
async function seedBundledWidget(vault) {
  const path = "widgets/focus-orbit.html";
  if (await vault.stat(path)) return;
  const response = await fetch(new URL(path, document.baseURI));
  if (!response.ok) throw new Error(`Could not load bundled widget source: ${response.status}`);
  await vault.write(path, await response.text(), { expectedHash: null, mediaType: "text/html" });
}

function configureLifeRuntime(vault) {
  lifeIndex = new MemoryIndex();
  lifeIndexer = new LifeIndexer({ vault, index: lifeIndex, canvasIdFromPath });
  lifeQuery = new LifeQuery(lifeIndex);
  componentCardCatalog = new ComponentCardCatalog({ vault });
  componentCardRepository = new FileComponentCardRepository({
    vault,
    catalog: componentCardCatalog,
    canvasPathFromId: id => {
      const record = workspace.canvases[id];
      return record ? canvasPathFor(record, workspace.rootId) : null;
    },
  });
  widgetCatalog = new WidgetCatalog({ vault });
  widgetRepository = new FileWidgetRepository({
    vault,
    catalog: widgetCatalog,
    canvasPathFromId: id => {
      const record = workspace.canvases[id];
      return record ? canvasPathFor(record, workspace.rootId) : null;
    },
  });
  taskRepository = new FileTaskRepository({
    vault, index: lifeIndex, indexer: lifeIndexer,
    canvasPathFromId: id => { const record = workspace.canvases[id]; return record ? canvasPathFor(record, workspace.rootId) : null; }
  });
  journalRepository = new FileJournalRepository({
    vault, index: lifeIndex, indexer: lifeIndexer,
    canvasPathFromId: id => { const record = workspace.canvases[id]; return record ? canvasPathFor(record, workspace.rootId) : null; }
  });
}
// Vault-first asynchronous boot. The only post-migration source of truth is the
// IndexedDB vault; the MemoryIndex is rebuilt from its files for every session.
async function bootCanvasApp(){
  try {
    const vault = new IndexedDbVault("orbit-vault");
    const store = new WorkspaceStore(vault);
    const hadWorkspace = await hasWorkspace(vault);
    const firstRun = !hadWorkspace && !localStorage.getItem(WORKSPACE_KEY) && !localStorage.getItem("orbit-canvas-v1");
    if (!hadWorkspace) await store.migrate(loadWorkspace());
    const result = await store.load();
    if (!result?.workspace?.canvases || !Object.keys(result.workspace.canvases).length) throw new Error("The vault workspace is empty");
    workspace = result.workspace;
    vaultStore = store;
    window.orbitVaultStore = store;
    setCanonicalWritable(!result.diagnostics.some((diagnostic) => diagnostic.code === "CANVAS_MISSING" || diagnostic.code === "CANVAS_INVALID" || diagnostic.code === "CANVAS_PARSE"), result.diagnostics.map((diagnostic) => diagnostic.message).join("; "));
    configureLifeRuntime(vault);
    await seedBundledWidget(vault);
    for (const diagnostic of result.diagnostics) console.warn("Vault workspace diagnostic", diagnostic);
    if (firstRun) {
      await seedGraphStarterEntities();
      workspace = (await store.load()).workspace;
    }
    await Promise.all([lifeIndexer.rebuild(), componentCardCatalog.rebuild(), widgetCatalog.rebuild()]);
    const stats = lifeIndexer.stats();
    setIndexStatus(canonicalWritable ? `Files · ${stats.sourceFiles} indexed` : "Files read-only · repair/export required", canonicalWritable ? `${stats.tasks} tasks · ${stats.habits} habits · ${stats.diagnostics} diagnostics` : "Repair the canonical vault or export it before editing.");
  } catch (error) {
    console.warn("Vault-first boot failed; canonical files are unavailable", error);
    vaultStore = null; lifeIndex = null; lifeIndexer = null; lifeQuery = null; taskRepository = null; journalRepository = null; componentCardCatalog = null; componentCardRepository = null; widgetCatalog = null; widgetRepository = null;
    setIndexStatus("Files unavailable", error.message);
    setCanonicalWritable(false, "Canonical files are unavailable; export or repair the vault before editing.");
  }
  currentCanvasId = workspace.canvases[workspace.activeId] ? workspace.activeId : workspace.rootId;
  documentData = workspace.canvases[currentCanvasId].document;
  camera = workspace.canvases[currentCanvasId].camera || { x: 80, y: 55, zoom: .78 };
  $("#canvasTitle").value = canvasRecord().title;
  renderWorkspaceNavigation(); render();
  setTimeout(fitView, 50);
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
  if(node?.type!=="file")return null;const record=Object.values(workspace.canvases).find(item=>item.path===node.file);return record?.id||null;
}
function canvasRecord(id=currentCanvasId){return workspace.canvases[id];}
function canvasDepth(id){
  let depth=0,seen=new Set();while(workspace.canvases[id]?.parentId&&!seen.has(id)){seen.add(id);id=workspace.canvases[id].parentId;depth++;}return depth;
}
function canvasTrail(id=currentCanvasId){
  const trail=[],seen=new Set();while(workspace.canvases[id]&&!seen.has(id)){seen.add(id);trail.unshift(workspace.canvases[id]);id=workspace.canvases[id].parentId;}return trail;
}
function slug(value){return String(value).toLowerCase().normalize("NFKD").replace(/[\u0300-\u036f]/g,"").replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,"").slice(0,54)||"canvas";}
async function rebuildLifeIndex(){
  if (!lifeIndexer) return;
  await Promise.all([lifeIndexer.rebuild(), componentCardCatalog?.rebuild(), widgetCatalog?.rebuild()]);
  const stats = lifeIndexer.stats();
  setIndexStatus(`Files · ${stats.sourceFiles} indexed`, `${stats.tasks} tasks · ${stats.habits} habits · ${stats.diagnostics} diagnostics`);
  renderToday(); renderNodes();
}
async function loadGraphStarter(){
  if(!canonicalWritable||!vaultStore) { toast("Canonical files are unavailable or read-only"); return; }
  if(!confirm("Replace your current local space with the graph starter? Export your space first if you want a backup."))return;
  try {
    await flushPendingWorkspaceEdits();
    const starter=createGraphStarterWorkspace();
    const stagingVault=new IndexedDbVault(`orbit-vault-${uid("reset")}`), stagingStore=new WorkspaceStore(stagingVault);
    await stagingStore.migrate(starter);
    await seedBundledWidget(stagingVault);
    workspace=starter;configureLifeRuntime(stagingVault);await seedGraphStarterEntities();
    const snapshot=await stagingVault.snapshot(), canonicalVault=vaultStore.vault;
    await canonicalVault.restore(snapshot);
    const nextStore=new WorkspaceStore(canonicalVault), result=await nextStore.load();
    if(!result?.workspace)throw new Error("Starter activation did not produce a workspace");
    vaultStore=nextStore;window.orbitVaultStore=nextStore;workspace=result.workspace;configureLifeRuntime(canonicalVault);await Promise.all([lifeIndexer.rebuild(),componentCardCatalog.rebuild(),widgetCatalog.rebuild()]);
    currentCanvasId=workspace.rootId;documentData=workspace.canvases[currentCanvasId].document;camera={x:80,y:55,zoom:.78};selected=null;connectSource=null;connectSourceSide=null;$("#canvasTitle").value=canvasRecord().title;renderWorkspaceNavigation();render();fitView();toast("Graph starter space loaded");
  } catch(error) { console.warn("Could not reset the canonical vault",error); toast(`Could not load starter: ${error.message}`); }
}
function portalPreview(document){
  const nodes=(document.nodes||[]).slice(0,28);if(!nodes.length)return '<span class="portal-empty">Empty canvas · open to begin</span>';
  const minX=Math.min(...nodes.map(node=>node.x)),minY=Math.min(...nodes.map(node=>node.y)),maxX=Math.max(...nodes.map(node=>node.x+node.width)),maxY=Math.max(...nodes.map(node=>node.y+node.height)),width=Math.max(1,maxX-minX),height=Math.max(1,maxY-minY),scale=Math.min(210/width,82/height);
  return nodes.map(node=>`<i class="${node.type==="group"?"group":""}" style="left:${(node.x-minX)*scale}px;top:${(node.y-minY)*scale}px;width:${Math.max(2,node.width*scale)}px;height:${Math.max(2,node.height*scale)}px;${node.type==="group"?"":`background:${colorValue(node.color)}`}" ></i>`).join("");
}
function componentDefined(name){return Boolean(customElements.get(name));}
function fallbackButton(label,className=""){
  const button=document.createElement("button");button.type="button";button.textContent=label;if(className)button.className=className;return button;
}
function renderFallbackWorkspaceNavigation(host,trail,canvases){
  host.dataset.nativeFallback="";
  const mode=host.getAttribute("mode")==="trail"?"trail":"canvases",items=mode==="trail"?trail:canvases,fragment=document.createDocumentFragment();
  items.forEach((item,index)=>{
    const button=fallbackButton(item.title||"Untitled canvas",mode==="canvases"?"nav-item canvas-list-item":"");
    button.dataset.fallbackCanvasId=item.id;
    if(item.id===currentCanvasId){button.classList.add("active");button.setAttribute("aria-current","page");}
    if(mode==="canvases"){
      button.textContent="";
      const icon=document.createElement("span"),title=document.createElement("b"),count=document.createElement("em");
      icon.setAttribute("aria-hidden","true");icon.textContent=item.icon||"↳";title.textContent=item.title||"Untitled canvas";count.textContent=String(item.count||0);
      button.style.setProperty("--canvas-depth",String(item.depth||0));button.append(icon,title,count);
    }
    fragment.append(button);
    if(mode==="trail"&&index<items.length-1){const separator=document.createElement("span");separator.setAttribute("aria-hidden","true");separator.textContent="›";fragment.append(separator);}
  });
  host.replaceChildren(fragment);
  host.onclick=event=>{
    const button=event.target.closest?.("[data-fallback-canvas-id]");if(!button||!host.contains(button))return;
    host.dispatchEvent(new CustomEvent("balaur-canvas-open",{bubbles:true,composed:true,detail:{canvasId:button.dataset.fallbackCanvasId}}));
  };
}
function orderedCanvasRecords(){
  const records=Object.values(workspace.canvases),result=[],seen=new Set(),compare=(a,b)=>a.title.localeCompare(b.title),visit=record=>{if(!record||seen.has(record.id))return;seen.add(record.id);result.push(record);records.filter(item=>item.parentId===record.id).sort(compare).forEach(visit);};visit(workspace.canvases[workspace.rootId]);records.sort(compare).forEach(visit);return result;
}
function defaultCanvasIcon(record){return record.id===workspace.rootId?"◫":record.kind==="hub"?"▦":record.kind==="project"?"◆":"↳";}
function canvasIconFor(record){return record?.icon||defaultCanvasIcon(record);}
function renderWorkspaceNavigation(){
  const trail=canvasTrail().map(record=>({id:record.id,title:record.title}));
  const canvases=orderedCanvasRecords().map(record=>({
    id:record.id,
    title:record.title,
    depth:canvasDepth(record.id),
    count:(record.document.nodes||[]).length,
    icon:canvasIconFor(record),
  }));
  const glyph=$("#canvasIconGlyph");if(glyph)glyph.textContent=canvasIconFor(canvasRecord());
  const breadcrumbs=$("#canvasBreadcrumbs"),list=$("#canvasList");
  if(breadcrumbs){
    if(componentDefined("balaur-workspace-nav")){breadcrumbs.trail=trail;breadcrumbs.canvases=canvases;breadcrumbs.activeId=currentCanvasId;}
    else renderFallbackWorkspaceNavigation(breadcrumbs,trail,canvases);
  }
  if(list){
    if(componentDefined("balaur-workspace-nav")){list.trail=trail;list.canvases=canvases;list.activeId=currentCanvasId;}
    else renderFallbackWorkspaceNavigation(list,trail,canvases);
  }
}
const CANVAS_ICON_SETS=[
  ["Atlas","🗺️ 🧭 🌍 🏔️ 🌊 🌲 🏠 🏛️ ✈️ 🧳"],
  ["Hoard","💰 🪙 💎 🏦 📈 🧾 🔑 🗝️ 📦 🧰"],
  ["Craft","💼 📁 📝 💡 🎯 🧠 🔬 📚 ⚙️ 🛠️"],
  ["Body","❤️ 🌱 🍎 🏃 😴 🧘 💊 🔥 ⭐ 🐉"],
  ["Kin","👤 👥 🤝 👪 🎁 🎉 💬 📞 🐾 🕯️"],
  ["Marks","◫ # ↳ ⚑ ◆ ✦ ☾ ☀ ♻ ⚡"],
];
function firstGrapheme(value){
  const text=String(value||"").trim();if(!text)return "";
  if(Intl.Segmenter){const segment=new Intl.Segmenter(undefined,{granularity:"grapheme"}).segment(text)[Symbol.iterator]().next();if(!segment.done)return segment.value.segment;}
  return Array.from(text)[0]||"";
}
function setCanvasIcon(value){
  const record=canvasRecord();if(!record)return;
  if(value)record.icon=value;else delete record.icon;
  scheduleSave();renderWorkspaceNavigation();
}
function initCanvasIconPicker(){
  const toggle=$("#canvasIconToggle"),panel=$("#canvasIconPanel");if(!toggle||!panel)return;
  const grid=panel.querySelector("[data-icon-grid]"),input=panel.querySelector("input"),fragment=document.createDocumentFragment();
  for(const [label,icons] of CANVAS_ICON_SETS){
    const group=document.createElement("div");group.className="canvas-icon-group";
    const heading=document.createElement("p");heading.textContent=label;
    const row=document.createElement("div");row.className="canvas-icon-grid";
    for(const icon of icons.split(" ")){const button=document.createElement("button");button.type="button";button.textContent=icon;button.dataset.icon=icon;button.setAttribute("aria-label",`Use ${icon} for this canvas`);row.append(button);}
    group.append(heading,row);fragment.append(group);
  }
  grid.append(fragment);
  const isOpen=()=>!panel.hidden;
  const open=()=>{panel.hidden=false;toggle.setAttribute("aria-expanded","true");};
  const close=()=>{panel.hidden=true;toggle.setAttribute("aria-expanded","false");};
  toggle.addEventListener("click",()=>{isOpen()?close():open();});
  panel.addEventListener("click",event=>{
    const choice=event.target.closest("[data-icon]");
    if(choice){setCanvasIcon(choice.dataset.icon);close();return;}
    if(event.target.closest("[data-icon-reset]")){input.value="";setCanvasIcon("");close();}
  });
  input.addEventListener("input",()=>{const grapheme=firstGrapheme(input.value);if(grapheme)setCanvasIcon(grapheme);});
  panel.addEventListener("keydown",event=>{if(event.key==="Escape"){event.stopPropagation();close();toggle.focus();}});
  document.addEventListener("pointerdown",event=>{const path=event.composedPath();if(isOpen()&&!path.includes(panel)&&!path.includes(toggle))close();});
}
function activateCanvas(id,{focusNodeId=null,fit=false}={}){
  const record=workspace.canvases[id];if(!record)return;currentCanvasId=id;workspace.activeId=id;documentData=record.document;camera=record.camera?{...record.camera}:{x:80,y:55,zoom:1};selected=null;connectSource=null;connectSourceSide=null;activeFilter="all";aiCardRuntime.clear();shell.classList.remove("inspector-open");$$('.nav-item[data-filter]').forEach(button=>button.classList.toggle("active",button.dataset.filter==="all"));$("#canvasTitle").value=record.title;render();
  if(focusNodeId){const node=documentData.nodes.find(item=>item.id===focusNodeId);if(node)focusNode(node,1.05);else fitView();}
  else if(fit||!record.camera)fitView();
}
function switchCanvas(id,{direction="in",focusNodeId=null,fit=false}={}){
  if(!workspace.canvases[id]||id===currentCanvasId)return;
  saveCurrentCanvasState();
  document.body.dataset.canvasNavigation=direction;
  const update=()=>activateCanvas(id,{focusNodeId,fit});
  const transition=updateWithViewTransition(update);
  if(transition)transition.finished.finally(()=>delete document.body.dataset.canvasNavigation);
  else delete document.body.dataset.canvasNavigation;
  scheduleSave();
}
function enterSubcanvas(id){if(workspace.canvases[id])switchCanvas(id,{direction:"in",fit:!workspace.canvases[id].camera});}
function leaveSubcanvas(){
  const child=canvasRecord(),parentId=child?.parentId;if(!parentId)return;switchCanvas(parentId,{direction:"out",focusNodeId:child.portalNodeId});
}
function focusNode(node,zoom=1.05){camera.zoom=zoom;camera.x=(canvas.clientWidth-node.width*zoom)/2-node.x*zoom;camera.y=(canvas.clientHeight-node.height*zoom)/2-node.y*zoom;applyCamera();}
function createSubcanvas(point){
  if(!canonicalWritable){toast("Canonical files are read-only until repaired or restored");return null;}
  const center=point||canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),id=uid("canvas"),nodeId=uid("node"),siblings=Object.values(workspace.canvases).filter(record=>record.parentId===currentCanvasId).length,title=`New canvas ${siblings+1}`,node={id:nodeId,type:"file",x:Math.round(center.x-180),y:Math.round(center.y-125),width:360,height:250,color:"3",file:`canvases/${id}.canvas`};
  workspace.canvases[id]={id,title,parentId:currentCanvasId,portalNodeId:nodeId,path:node.file,document:{nodes:[],edges:[]},camera:null};documentData.nodes.push(node);selected={kind:"node",id:node.id};shell.classList.add("inspector-open");scheduleSave();render();toast("Sub-canvas created · double-click or zoom into it");return node;
}
function nextNodePosition(document,width,height){
  if(document===documentData){const box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2);return{x:Math.round(center.x-width/2),y:Math.round(center.y-height/2)};}const nodes=document.nodes||[];if(!nodes.length)return{x:0,y:0};return{x:Math.max(...nodes.map(node=>node.x+node.width))+60,y:Math.min(...nodes.map(node=>node.y))};
}
function revealWorkspaceNode(canvasId,nodeId){
  const reveal=()=>{if(currentCanvasId!==canvasId)return;const node=documentData.nodes.find(item=>item.id===nodeId);if(!node)return;selected={kind:"node",id:nodeId};shell.classList.add("inspector-open");render();focusNode(node,1.05);};if(currentCanvasId===canvasId)reveal();else{switchCanvas(canvasId,{direction:"switch",focusNodeId:nodeId});setTimeout(reveal,320);}
}
function localDateISO(date=new Date()){return `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,"0")}-${String(date.getDate()).padStart(2,"0")}`;}
function taskForNode(node){
  if(node?.type!=="file"||!lifeIndex)return null;
  return lifeIndex.allTasks().find(task=>task.sourcePath===node.file)||null;
}
function taskIdFromNode(node){return taskForNode(node)?.id||null;}
function taskPlacement(task){return task&&lifeIndex?.placementsForEntity(task.id)[0]||null;}
async function updateTask(id, patch){
  if(!canonicalWritable||!taskRepository) throw new Error("Canonical files are unavailable or read-only.");
  await flushPendingWorkspaceEdits();
  const task=await enqueueMutation(()=>taskRepository.updateTask(id, patch));
  renderNodes(); renderToday(); return task;
}
function scheduleTaskFieldUpdate(id, patch) {
  const prior=taskUpdateTimers.get(id);if(prior)clearTimeout(prior.timer);
  const merged={...(prior?.patch||{}),...patch};
  const timer=setTimeout(()=>{taskUpdateTimers.delete(id);updateTask(id,merged).catch(error=>toast(error.message));},250);
  taskUpdateTimers.set(id,{timer,patch:merged});
}
function flushTaskFieldUpdate(id, patch) {
  const prior=taskUpdateTimers.get(id);if(prior)clearTimeout(prior.timer);
  taskUpdateTimers.delete(id);
  return updateTask(id,{...(prior?.patch||{}),...patch}).catch(error=>toast(error.message));
}
async function completeTask(id){
  if(!canonicalWritable||!taskRepository) throw new Error("Canonical files are unavailable or read-only.");
  await flushPendingWorkspaceEdits();
  const task=await enqueueMutation(()=>taskRepository.completeTask(id));
  renderNodes(); renderToday(); return task;
}
async function createTask({title,notes="",canvasId=currentCanvasId,status="inbox",scheduledOn=null,dueOn=null,priority=null}={}){
  title=String(title||"").trim();if(!title)throw new Error("Add a task title.");if(!canonicalWritable||!taskRepository)throw new Error("Canonical files are unavailable or read-only.");const record=workspace.canvases[canvasId];if(!record)throw new Error("Choose an existing canvas.");const nodeId=uid("node"),position=nextNodePosition(record.document,310,180);
  await flushPendingWorkspaceEdits();
  const result=await enqueueMutation(async()=>{
    const created=await taskRepository.createTask({title,body:notes,canvasId,status,scheduledOn:scheduledOn||null,dueOn:dueOn||null,priority:priority===""||priority==null?null:Number(priority),geometry:{...position,width:310,height:180,color:"5",id:nodeId}});
    await reloadCanvasDocuments([canvasId]);
    return created;
  });
  const node=workspace.canvases[canvasId].document.nodes.find(item=>item.id===result.placement?.nodeId||item.file===result.path);
  renderToday();if(canvasId===currentCanvasId&&activeAppView==="canvas"){selected={kind:"node",id:node?.id||nodeId};shell.classList.add("inspector-open");render();}toast("Task created");return node;
}
function openTaskDialog({today=false}={}){
  const dialog=$("#taskDialog"),select=$("#taskCanvas");select.innerHTML=orderedCanvasRecords().map(record=>`<option value="${escapeHTML(record.id)}">${escapeHTML(record.title)}</option>`).join("");select.value=currentCanvasId;$("#taskTitle").value="";$("#taskNotes").value="";$("#taskStatus").value=today?"scheduled":"inbox";$("#taskScheduledOn").value=today?localDateISO():"";$("#taskDueOn").value="";$("#taskPriority").value="";$("#taskResult").textContent="";dialog.showModal();setTimeout(()=>$("#taskTitle").focus(),60);
}
function taskContext(task){const placement=taskPlacement(task);return workspace.canvases[placement?.canvasId]?.title||"Inbox";}
function renderFallbackTaskList(list,tasks,emptyMessage){
  list.dataset.nativeFallback="";
  if(!tasks.length){const empty=document.createElement("p");empty.className="task-list-empty";empty.textContent=emptyMessage;list.replaceChildren(empty);return;}
  const rows=document.createElement("ul");rows.className="task-list";
  for(const task of tasks){
    const row=document.createElement("li");row.dataset.fallbackTaskId=task.id;
    const open=fallbackButton(task.title||"Untitled task");open.dataset.fallbackTaskOpen=task.id;
    const context=document.createElement("small");context.textContent=task.context||"Inbox";row.append(open,context);
    if(task.status!=="done"){const complete=fallbackButton("Mark done");complete.dataset.fallbackTaskComplete=task.id;row.append(complete);}
    rows.append(row);
  }
  list.replaceChildren(rows);
  list.onclick=event=>{
    const open=event.target.closest?.("[data-fallback-task-open]"),complete=event.target.closest?.("[data-fallback-task-complete]");
    if(open&&list.contains(open))list.dispatchEvent(new CustomEvent("balaur-task-open",{bubbles:true,composed:true,detail:{taskId:open.dataset.fallbackTaskOpen}}));
    else if(complete&&list.contains(complete))list.dispatchEvent(new CustomEvent("balaur-task-complete",{bubbles:true,composed:true,detail:{taskId:complete.dataset.fallbackTaskComplete}}));
  };
}
function renderToday(){
  const root=$("#todayView");if(!root||!lifeQuery)return;
  const today=localDateISO(),all=lifeQuery.tasks(),active=task=>!["done","cancelled"].includes(task.status);
  const scheduled=all.filter(task=>active(task)&&task.scheduledOn===today);
  const overdue=all.filter(task=>active(task)&&task.dueOn&&task.dueOn<today&&task.scheduledOn!==today);
  const queue=all.filter(task=>active(task)&&["inbox","next"].includes(task.status)&&task.scheduledOn!==today&&!overdue.includes(task));
  const completed=all.filter(task=>task.status==="done"&&task.completedAt&&localDateISO(new Date(task.completedAt))===today);
  $("#todayDate").textContent=new Intl.DateTimeFormat(undefined,{weekday:"long",month:"long",day:"numeric"}).format(new Date());
  $("#todayPlannedCount").textContent=scheduled.length;$("#todayDueCount").textContent=overdue.length;$("#todayDoneCount").textContent=completed.length;
  const assign=(selector,tasks,empty)=>{
    const list=$(selector);if(!list)return;
    const items=tasks.map(task=>({...task,context:taskContext(task)}));
    if(componentDefined("balaur-task-list")){list.emptyMessage=empty;list.items=items;}
    else renderFallbackTaskList(list,items,empty);
  };
  assign("#todayScheduled",scheduled,"Nothing scheduled yet. Choose deliberately rather than carrying everything forward.");
  assign("#todayOverdue",overdue,"No overdue tasks.");
  assign("#todayQueue",queue,"The task inbox is clear.");
  assign("#todayCompleted",completed,"Completed tasks will appear here.");
  renderJournalPanel();
}
let journalViewDate=localDateISO();
let journalLoadedDate=null;
let journalSaveTimer=null;
function flushJournalSave(){
  if(journalSaveTimer!==null){clearTimeout(journalSaveTimer);journalSaveTimer=null;saveJournalBody(journalViewDate);}
}
function shiftJournalDate(delta){
  flushJournalSave();
  const d=new Date(`${journalViewDate}T00:00:00Z`);d.setUTCDate(d.getUTCDate()+delta);
  journalViewDate=`${d.getUTCFullYear()}-${String(d.getUTCMonth()+1).padStart(2,"0")}-${String(d.getUTCDate()).padStart(2,"0")}`;
  renderJournalPanel();
}
async function renderJournalPanel(){
  const body=$("#journalBody");if(!body||!journalRepository)return;
  $("#journalDate").textContent=new Intl.DateTimeFormat(undefined,{weekday:"long",year:"numeric",month:"long",day:"numeric"}).format(new Date(`${journalViewDate}T00:00:00`));
  $("#journalToday").disabled=journalViewDate===localDateISO();
  if(journalLoadedDate===journalViewDate)return;
  journalLoadedDate=journalViewDate;
  let text="";
  try{text=(await journalRepository.getJournal(journalViewDate)).body||"";}catch(_){text="";}
  if(journalLoadedDate===journalViewDate)body.value=text;
  $("#journalStatus").textContent="";
}
async function saveJournalBody(date){
  if(!journalRepository)return;
  const body=$("#journalBody").value,status=$("#journalStatus");
  try{
    try{await journalRepository.getJournal(date);}
    catch(_){await journalRepository.createJournal({localDate:date,body:""});}
    await journalRepository.updateJournal(date,{body});
    if(date===journalViewDate)status.textContent="Saved";
  }catch(error){if(date===journalViewDate)status.textContent=`Save failed: ${error.message}`;}
}
async function placeJournalOnCanvas(){
  if(!journalRepository||!canonicalWritable){toast("Canonical files are read-only");return;}
  flushJournalSave();
  try{
    try{await journalRepository.getJournal(journalViewDate);}
    catch(_){await journalRepository.createJournal({localDate:journalViewDate,body:$("#journalBody").value});}
    await flushPendingWorkspaceEdits();
    const placement=await enqueueMutation(()=>journalRepository.addPlacement(journalViewDate,currentCanvasId,{}));
    await reloadCanvasDocuments([currentCanvasId]);
    setAppView("canvas");revealWorkspaceNode(currentCanvasId,placement.nodeId);
    toast("Journal placed on canvas");
  }catch(error){toast(error.message);}
}
function deleteCanvasTree(id){for(const child of Object.values(workspace.canvases).filter(record=>record.parentId===id))deleteCanvasTree(child.id);delete workspace.canvases[id];}

const AI_CARD_MARKER="<!-- orbit:ai-card -->";
function isAICard(node){return node?.type==="text"&&node.text.includes(AI_CARD_MARKER);}
function parseAICard(node){
  const lines=(node.text||"").split(/\r?\n/).filter(line=>line.trim()!==AI_CARD_MARKER),heading=lines.findIndex(line=>line.startsWith("# ")),title=heading>=0?lines[heading].slice(2).trim():"AI operator";
  if(heading>=0)lines.splice(heading,1);return {title,prompt:lines.join("\n").trim()||"Summarize the connected notes."};
}
function buildAICardText(title,prompt){return `${AI_CARD_MARKER}\n# ${title.trim()||"AI operator"}\n${prompt.trim()||"Summarize the connected notes."}`;}
function indexedEntityForPath(path) {
  const source = lifeIndex?.getSourceFile(path);
  if (!source?.entityId || source.parseStatus === "error") return null;
  const readers = { task: "allTasks", habit: "allHabits", journal: "allJournals", "calendar-event": "allEvents" };
  const row = readers[source.entityType] ? lifeIndex[readers[source.entityType]]().find(item => item.id === source.entityId || item.orbitId === source.entityId) : null;
  return { source, row };
}
function nodeTitle(node){
  if(isAICard(node))return parseAICard(node).title;if(node.type==="text"){const heading=node.text.match(/^#{1,2}\s+(.+)$/m);return heading?heading[1]:"Text note";}if(node.type==="group")return node.label||"Group";if(node.type==="link")try{return new URL(node.url).hostname;}catch(_){return "Link";}if(node.type==="file"){const subcanvasId=subcanvasIdFromNode(node);if(subcanvasId)return workspace.canvases[subcanvasId].title;return indexedEntityForPath(node.file)?.row?.title||node.file.split("/").pop();}return node.id;
}
function inputNodesForAICard(cardId,data=documentData){const byId=Object.fromEntries((data.nodes||[]).map(node=>[node.id,node]));return (data.edges||[]).filter(edge=>edge.toNode===cardId&&edge.label!=="AI output").map(edge=>byId[edge.fromNode]).filter(Boolean);}
function nodeAIContent(node){
  if(node.type==="text")return node.text;if(node.type==="link")return node.url;
  if(node.type==="file"){
    const cached=aiFileContentCache.get(node.file);if(cached)return cached;
    const entity=indexedEntityForPath(node.file);if(entity?.row)return [`Title: ${entity.row.title||entity.row.localDate||node.file}`, "Canonical body is loading.", node.subpath].filter(Boolean).join("\n");
    return node.subpath ? `${node.file}\nSubpath: ${node.subpath}` : node.file;
  }
  if(node.type==="group")return node.label||"";return "";
}
async function preloadAIFileInputs(nodes) {
  for(const node of nodes.filter(item=>item.type==="file"&&!subcanvasIdFromNode(item))) {
    if(aiFileContentCache.has(node.file)||!vaultStore)continue;
    try {
      const raw=await vaultStore.vault.read(node.file);
      try { const parsed=parseEntity(raw); aiFileContentCache.set(node.file, [`Title: ${parsed.title||parsed.localDate||node.file}`, parsed.body||""].filter(Boolean).join("\n\n")); }
      catch (_) { aiFileContentCache.set(node.file, raw); }
    } catch (error) { aiFileContentCache.set(node.file, `Canonical file unavailable: ${node.file}`); console.warn("Could not preload AI file context", node.file, error); }
  }
}
function aiCardSignature(card,data=documentData){return JSON.stringify([card.text,inputNodesForAICard(card.id,data).map(node=>[node.id,nodeAIContent(node)])]);}
function aiCardSignatures(data=documentData){return new Map((data.nodes||[]).filter(isAICard).map(card=>[card.id,aiCardSignature(card,data)]));}

function textMeta(node) {
  const map = {"1":"GOAL", "2":"IDEA", "3":"NOTE", "4":"HABIT", "5":"RESOURCE", "6":"PROJECT"};
  return map[node.color] || node.type.toUpperCase();
}

function noteKind(node){
  if(node?.type!=="text")return null;
  if(node.text.includes(NOTE_MARKERS.inbox))return "inbox";
  if(node.text.includes(NOTE_MARKERS.reference))return "reference";
  return null;
}
function canvasKind(record){return record?.kind==="hub"||record?.kind==="project"?record.kind:null;}
// One-line summary convention (ADR-0003): heading/title first, else first body line.
function nodeSummary(node){
  const title=nodeTitle(node);
  if(node?.type==="text"){
    const bodyLine=(node.text||"").split(/\r?\n/).find(line=>line.trim()&&!/^#{1,2}\s/.test(line)&&!/^\s*<!--\s*orbit:/.test(line));
    const summary=(bodyLine||"").trim().slice(0,120);
    return summary&&summary!==title?`${title} — ${summary}`:title;
  }
  return title;
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

function widgetThemeSnapshot() {
  const style=getComputedStyle(document.documentElement),read=name=>style.getPropertyValue(name).trim();
  return {
    surface:read("--balaur-surface-oak"),surfaceRaised:read("--balaur-surface-oak-raised"),
    content:read("--balaur-content-on-dark"),contentMuted:read("--balaur-content-on-dark-muted"),
    paper:read("--balaur-surface-parchment"),ink:read("--balaur-content-on-paper"),
    primary:read("--balaur-action-primary"),focus:read("--balaur-border-focus"),
    danger:read("--balaur-status-danger"),radius:read("--balaur-radius-panel"),
    fontBody:read("--balaur-font-body"),fontMono:read("--balaur-font-mono"),
  };
}
function widgetPreferences(){return {reducedMotion:matchMedia("(prefers-reduced-motion: reduce)").matches,reducedTransparency:matchMedia("(prefers-reduced-transparency: reduce)").matches,contrast:matchMedia("(prefers-contrast: more)").matches?"more":matchMedia("(prefers-contrast: less)").matches?"less":"no-preference"};}
function showWidgetSourceReview({title="Live widget",path="",source=""}={}) {
  let dialog=$("#widgetSourceDialog");
  if(!dialog){
    dialog=document.createElement("dialog");dialog.id="widgetSourceDialog";dialog.className="widget-source-dialog";
    dialog.innerHTML='<article><header><div><small>REVIEWED CANONICAL SOURCE</small><h2></h2></div><button type="button" data-close-widget-source aria-label="Close source review">Close</button></header><p class="widget-capability-summary">Sandboxed scripts and inline styles only. No host data or mutation, storage, network, forms, popups, workers, or nested frames. Self-navigation pauses the widget; hard request suppression is not claimed.</p><code></code><pre></pre></article>';
    document.body.append(dialog);$("[data-close-widget-source]",dialog).onclick=()=>dialog.close();
  }
  $("h2",dialog).textContent=title;$("code",dialog).textContent=path;$("pre",dialog).textContent=source;dialog.showModal();
}
document.addEventListener("balaur-widget-view-source",event=>{event.stopPropagation();showWidgetSourceReview(event.detail);});

function renderFallbackFileContent(content,kind,model){
  const article=document.createElement("article");
  article.dataset[kind==="component"?"fallbackComponentCard":"fallbackWidget"]="";
  const kicker=document.createElement("small"),title=document.createElement("h3"),identity=document.createElement("p");
  kicker.textContent=kind==="component"?"COMPONENT CARD · STATIC FALLBACK":"LIVE WIDGET · INACTIVE FALLBACK";
  title.textContent=model.title||model.path||"Unavailable file";
  identity.textContent=kind==="component"
    ? `${model.recipe?`Recipe: ${model.recipe} · `:""}${model.path||""}`
    : `${model.path||""} · Source is not executed while component registration is unavailable.`;
  article.append(kicker,title,identity);
  const detail=kind==="component"?(model.diagnostic||model.body):(model.diagnostic||"Reviewed canonical source remains inactive.");
  if(detail){const description=document.createElement("p");description.textContent=String(detail).slice(0,600);article.append(description);}
  content.replaceChildren(article);
}
function renderNodes() {
  const selectionKey = selected?.kind === "node" ? selected.id : null;
  const selectionEntering = selectionKey !== null && selectionKey !== renderedSelectionKey;
  const retainedElements = new Map();
  for (const element of [...nodeLayer.children]) {
    const stableContent = element.querySelector("balaur-component-card, balaur-widget-frame");
    if (stableContent) retainedElements.set(element.dataset.id, element);
    else element.remove();
  }
  const renderedElements = new Set();
  (documentData.nodes || []).forEach((node, index) => {
    const retained = retainedElements.get(node.id);
    const componentPath = node.type === "file" && /^cards\/.+\.md$/i.test(node.file || "");
    const widgetPath = node.type === "file" && /\.html?$/i.test(node.file || "") && !subcanvasIdFromNode(node);
    const canRetain = retained
      && retained.dataset.filePath === (node.file || "")
      && ((componentPath && retained.dataset.renderKind === "component-card")
        || (widgetPath && retained.dataset.renderKind === "html-widget"));
    const element = canRetain ? retained : $("#nodeTemplate").content.firstElementChild.cloneNode(true);
    element.dataset.id = node.id;
    element.dataset.color = node.color || "";
    element.style.cssText = `left:${node.x}px;top:${node.y}px;width:${node.width}px;height:${node.height}px;`;
    const isSelected = selectionKey === node.id;
    element.classList.toggle("selected", isSelected);
    element.classList.toggle("selection-entering", isSelected && selectionEntering);
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
        content.innerHTML=`<div class="node-kicker">AI OPERATOR</div><div class="node-body"><h3 class="ai-card-title">${escapeHTML(config.title)}</h3><p class="ai-card-prompt">${escapeHTML(config.prompt)}</p><div class="ai-inputs">${inputs.length?inputs.map(input=>`<span class="ai-input-chip">← ${escapeHTML(nodeTitle(input))}</span>`).join(""):"<span class=\"ai-input-chip\">No inputs connected</span>"}</div></div><div class="ai-run-row"><span class="ai-run-status">${escapeHTML(runtime.status||"Ready")}</span><button class="ai-run-button" data-ai-run ${runtime.running?"disabled":""}>${runtime.running?"Running…":"Run now"}</button></div>`;
      } else {
        const kind=noteKind(node);
        const kicker=kind==="inbox"?"INBOX · capture":kind==="reference"?"REFERENCE · wiki":textMeta(node);
        content.innerHTML=`<div class="node-kicker">${kicker}</div><div class="node-body">${markdownToHTML(node.text)}</div>`;
      }
    } else if (componentPath) {
      element.classList.add("component-card-node");
      element.dataset.renderKind = "component-card";
      element.dataset.filePath = node.file;
      const model = componentCardCatalog?.getByPath(node.file)
        || componentCardCatalog?.getFallbackByPath(node.file)
        || Object.freeze({
          id: null,
          title: node.file.split("/").pop() || "Component card",
          recipe: null,
          body: "",
          path: node.file,
          diagnostic: `Component-card file is unavailable: ${node.file}`,
        });
      if(!componentDefined("balaur-component-card"))renderFallbackFileContent(content,"component",model);
      else{
        let host = $("balaur-component-card", content);
        if (!host) {
          host = document.createElement("balaur-component-card");
          host.addEventListener("balaur-card-open", event => {
            event.stopPropagation();
            const nodeId = event.detail?.nodeId;
            if (nodeId && documentData.nodes.some(item => item.id === nodeId)) selectItem("node", nodeId);
          });
          content.replaceChildren(host);
        }
        host.dataset.nodeId = node.id;
        host.placementColor = node.color || null;
        host.model = model;
      }
    } else if (node.type === "file" && taskIdFromNode(node)) {
      const task=taskForNode(node),taskId=task.id,status=task.status||"inbox";element.classList.add("task-card");element.classList.toggle("task-complete",status==="done");element.dataset.taskId=taskId;content.innerHTML=`<div class="node-kicker">TASK · ${escapeHTML(status.toUpperCase())}</div><div class="node-body">${markdownToHTML(`# ${task.title}`)}</div><div class="task-node-footer"><span>${task.scheduledOn?`Plan ${escapeHTML(task.scheduledOn)}`:task.dueOn?`Due ${escapeHTML(task.dueOn)}`:"Not scheduled"}</span><button type="button" data-node-complete-task ${status==="done"?"disabled":""}>${status==="done"?"Completed":"Mark done"}</button></div>`;
    } else if (node.type === "link") {
      let linkTitle = "Saved link";
      try { linkTitle = new URL(node.url).hostname.replace(/^www\./, ""); } catch (_) {}
      content.innerHTML = `<div class="node-kicker">LINK</div><div class="node-body"><h3>${escapeHTML(linkTitle)}</h3><p>Open this resource in a new tab.</p><a class="node-link" href="${safeURL(node.url)}" target="_blank" rel="noreferrer">${escapeHTML(node.url)} ↗</a></div>`;
    } else if (node.type === "file") {
      const subcanvasId=subcanvasIdFromNode(node),subcanvas=subcanvasId&&workspace.canvases[subcanvasId];
      const fileEntity=indexedEntityForPath(node.file);
      if(subcanvas){
        const children=Object.values(workspace.canvases).filter(record=>record.parentId===subcanvasId).length;element.classList.add("subcanvas-node");element.dataset.subcanvasId=subcanvasId;
        content.innerHTML=`<div class="node-kicker">${subcanvas.kind==="hub"?"HUB · PORTAL":subcanvas.kind==="project"?"PROJECT · PORTAL":"SUB-CANVAS · ZOOM PORTAL"}</div><div class="node-body"><h3>${escapeHTML(subcanvas.title)}</h3><p>${subcanvas.document.nodes.length} item${subcanvas.document.nodes.length===1?"":"s"}${children?` · ${children} nested`:""}</p><div class="portal-preview">${portalPreview(subcanvas.document)}</div><div class="portal-actions"><span>Double-click or zoom to 220%</span><button type="button" data-open-subcanvas>Open ↘</button></div></div>`;
      } else if (widgetPath) {
        element.classList.add("html-widget");
        element.dataset.renderKind = "html-widget";
        element.dataset.filePath = node.file;
        const model=widgetCatalog?.getByPath(node.file)||widgetCatalog?.getFallbackByPath(node.file)||{path:node.file,title:node.file.split("/").pop()||"Live widget",source:"",diagnostic:`Widget file is unavailable: ${node.file}`};
        if(!componentDefined("balaur-widget-frame"))renderFallbackFileContent(content,"widget",model);
        else{
          let host=$("balaur-widget-frame",content);
          if(!host){
            content.innerHTML='<div class="node-kicker">LIVE HTML · REVIEWED SANDBOX</div><div class="node-body widget-node-body"></div><div class="widget-shield"></div>';
            host=document.createElement("balaur-widget-frame");
            $(".widget-node-body",content).append(host);
          }
          host.path=model.path;host.title=model.title;host.source=model.source;host.diagnostic=model.diagnostic||"";host.themeSnapshot=widgetThemeSnapshot();host.preferences=widgetPreferences();
        }
      } else if (fileEntity?.source?.entityType==="journal") {
        const row=fileEntity.row;
        content.innerHTML=`<div class="node-kicker">JOURNAL · ${escapeHTML(row.localDate)}</div><div class="node-body"><h3>${escapeHTML(row.localDate)}</h3><p>Daily note. Open in Today to edit, or place it on a canvas.</p></div>`;
      } else content.innerHTML = `<div class="node-kicker">FILE</div><div class="node-body"><div class="file-preview">▧</div><h3>${escapeHTML(node.file.split("/").pop())}</h3><p>${escapeHTML(node.subpath || node.file)}</p></div>`;
    }
    if (!canRetain) {
      const liveNode = () => documentData.nodes.find(item => item.id === element.dataset.id);
      element.addEventListener("pointerdown", event => { const item=liveNode();if(item)nodePointerDown(event,item); });
      $$("[data-connection-side]",element).forEach(handle=>{handle.addEventListener("pointerdown",event=>{const item=liveNode();if(item)startConnectionDrag(event,item,handle.dataset.connectionSide);});handle.addEventListener("keydown",event=>{if(event.key==="Enter"||event.key===" "){event.preventDefault();event.stopPropagation();connectSource=element.dataset.id;connectSourceSide=handle.dataset.connectionSide;setTool("connect");toast("Choose a destination node");}});});
      const aiRun=$("[data-ai-run]",element);if(aiRun){aiRun.addEventListener("pointerdown",event=>event.stopPropagation());aiRun.addEventListener("click",event=>{event.stopPropagation();runAICard(element.dataset.id,{manual:true});});}
      const taskComplete=$("[data-node-complete-task]",element);if(taskComplete){taskComplete.addEventListener("pointerdown",event=>event.stopPropagation());taskComplete.addEventListener("click",async event=>{event.stopPropagation();try{await completeTask(element.dataset.taskId);toast("Task completed");}catch(error){toast(error.message);}});}
      const portalButton=$("[data-open-subcanvas]",element);if(portalButton){portalButton.addEventListener("pointerdown",event=>event.stopPropagation());portalButton.addEventListener("click",event=>{event.stopPropagation();enterSubcanvas(element.dataset.subcanvasId);});element.addEventListener("dblclick",event=>{if(event.target.closest("button"))return;event.preventDefault();event.stopPropagation();enterSubcanvas(element.dataset.subcanvasId);});}
      element.addEventListener("click", event => {
        const anchor = event.target.closest("a");
        if (anchor) event.stopPropagation();
      });
    }
    const currentAtIndex = nodeLayer.children[index];
    if (currentAtIndex !== element) nodeLayer.insertBefore(element, currentAtIndex || null);
    renderedElements.add(element);
  });
  for (const element of [...nodeLayer.children]) if (!renderedElements.has(element)) element.remove();
  renderedSelectionKey = selectionKey;
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
function setAppView(view){
  $("balaur-add-menu")?.close?.();
  activeAppView=view==="today"?"today":"canvas";if(activeAppView==="today")shell.classList.remove("inspector-open");$("#canvas").hidden=activeAppView!=="canvas";$("#todayView").hidden=activeAppView!=="today";$$('[data-app-view]').forEach(button=>button.classList.toggle("active",button.dataset.appView===activeAppView));if(activeAppView==="today")renderToday();else applyCamera();
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
// Geometry hit-test: the node (if any) whose rect contains a screen point, topmost
// first. Unlike event.target / elementFromPoint this is immune to the node layer
// being rebuilt mid-click, so it reliably tells us a pointer is over a card even
// when the browser retargets the event to the background. Filtered (dimmed,
// pointer-events:none) nodes are skipped to match their non-interactive state.
function nodeAtClientPoint(clientX,clientY){
  const p=canvasPoint(clientX,clientY),nodes=documentData.nodes||[];
  for(let i=nodes.length-1;i>=0;i--){const n=nodes[i];
    if(activeFilter!=="all"&&n.type!=="group"&&n.color!==activeFilter)continue;
    if(p.x>=n.x&&p.x<=n.x+n.width&&p.y>=n.y&&p.y<=n.y+n.height)return n;}
  return null;
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
    // Re-rendering on selection can retarget events to the background layer;
    // geometry-test so a click that lands on a card never deselects or creates.
    if (nodeAtClientPoint(event.clientX,event.clientY)) return;
    selected=null;connectSource=null;connectSourceSide=null;shell.classList.remove("inspector-open");render();
    if (currentTool === "note") { const p=canvasPoint(event.clientX,event.clientY); addNode("note",p); setTool("select"); }
  }
});
canvas.addEventListener("dblclick", event => {
  if (event.target.closest?.(".canvas-tools,.zoom-tools,.minimap,.edges")) return;
  // Create a note on empty canvas only. The geometry test (not event.target) is
  // authoritative so a double click over a card never spawns a note on top of it,
  // even when a mid-click re-render retargets the event to the background layer.
  if (nodeAtClientPoint(event.clientX,event.clientY)) return;
  addNode("note",canvasPoint(event.clientX,event.clientY));
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
  canvas.classList.toggle("tool-pan",tool==="pan"); canvas.classList.toggle("tool-connect",tool==="connect"); renderNodes();
}


function addNode(kind, point) {
  if(!canonicalWritable){toast("Canonical files are read-only until repaired or restored");return null;}
  if(kind==="subcanvas")return createSubcanvas(point);if(kind==="task"){openTaskDialog();return;}
  const center = point || canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2);
  const presets={
    note:{type:"text",color:"2",width:260,height:150,text:"# New thought\nStart writing here…"},
    inbox:{type:"text",color:"2",width:280,height:160,text:`${NOTE_MARKERS.inbox}\n# New capture\nWrite it down now; process it later.`},
    reference:{type:"text",color:"5",width:300,height:190,text:`${NOTE_MARKERS.reference}\n# New reference\nDurable knowledge worth keeping.`},
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

function renderFallbackInspector(panel,model){
  panel.dataset.fallbackInspector="";
  if(!model){const empty=document.createElement("p");empty.textContent="Select a node or connection to inspect it.";panel.replaceChildren(empty);return;}
  const header=document.createElement("header"),title=document.createElement("h2"),close=fallbackButton("Close");
  title.textContent=model.title||"Inspector";close.dataset.inspectorClose="";header.append(title,close);
  const fields=document.createElement("div");
  for(const field of model.fields||[]){
    const label=document.createElement("label"),caption=document.createElement("span");
    caption.textContent=field.label||field.key;
    let control;
    if(field.control==="textarea")control=document.createElement("textarea");
    else if(field.control==="select"){
      control=document.createElement("select");
      for(const option of field.options||[]){const node=document.createElement("option");node.value=option.value;node.textContent=option.label;control.append(node);}
    }else{control=document.createElement("input");control.type=field.control||"text";}
    control.dataset.fieldKey=field.key;control.value=field.value??"";control.disabled=Boolean(model.readonly);label.append(caption,control);fields.append(label);
  }
  const notes=document.createElement("div");for(const note of model.notes||[]){const text=document.createElement("p");text.textContent=note.text||"";notes.append(text);}
  const colors=document.createElement("div");for(const color of model.colors||[]){const button=fallbackButton(`Color ${color.value}`);button.dataset.color=color.value;button.disabled=Boolean(model.readonly);colors.append(button);}
  const actions=document.createElement("div");for(const action of model.actions||[]){const button=fallbackButton(action.label||action.intent);button.dataset.intent=action.intent;button.disabled=Boolean(model.readonly)&&action.requiresWrite!==false;actions.append(button);}
  panel.replaceChildren(header,fields,notes,colors,actions);
  const fieldDetail=control=>{const field=model.fields.find(candidate=>candidate.key===control.dataset.fieldKey);return field?{key:field.key,value:field.control==="number"?Math.round(Number(control.value)):control.value,scope:field.scope||"item",modelKey:String(model.key||""),taskId:field.taskId||null,canvasId:field.canvasId||null}:null;};
  const fieldEvent=(event,phase)=>{const control=event.target.closest?.("[data-field-key]"),detail=control&&fieldDetail(control);if(detail)panel.dispatchEvent(new CustomEvent(`balaur-inspector-field-${phase}`,{bubbles:true,composed:true,detail}));};
  panel.oninput=event=>fieldEvent(event,"input");panel.onchange=event=>fieldEvent(event,"change");panel.onfocusout=event=>fieldEvent(event,"blur");
  panel.onclick=event=>{
    const closeButton=event.target.closest?.("[data-inspector-close]"),color=event.target.closest?.("[data-color]"),action=event.target.closest?.("[data-intent]");
    if(closeButton)panel.dispatchEvent(new CustomEvent("balaur-inspector-close",{bubbles:true,composed:true}));
    else if(color&&!color.disabled)panel.dispatchEvent(new CustomEvent("balaur-inspector-color",{bubbles:true,composed:true,detail:{value:color.dataset.color,modelKey:String(model.key||"")}}));
    else if(action&&!action.disabled){const configured=model.actions.find(candidate=>candidate.intent===action.dataset.intent);if(configured)panel.dispatchEvent(new CustomEvent("balaur-inspector-action",{bubbles:true,composed:true,detail:{intent:configured.intent,modelKey:String(model.key||""),taskId:configured.taskId||null,cardId:configured.cardId||null,canvasId:configured.canvasId||null}}));}
  };
}
function setInspectorModel(panel,model){
  if(componentDefined("balaur-inspector"))panel.model=model;
  else renderFallbackInspector(panel,model);
}
function renderInspector() {
  const panel=$("#inspector");
  if(!selected){setInspectorModel(panel,null);return;}
  const item=selected.kind==="node"?documentData.nodes.find(node=>node.id===selected.id):documentData.edges.find(edge=>edge.id===selected.id);
  if(!item){selected=null;shell.classList.remove("inspector-open");setInspectorModel(panel,null);return;}
  const fields=[],notes=[],actions=[];
  let title="Connection";
  if(selected.kind==="node"){
    const task=taskForNode(item),componentCard=item.type==="file"&&!task?componentCardCatalog?.getByPath(item.file):null;
    title=task?"Task":componentCard?"Component card":`${item.type[0].toUpperCase()+item.type.slice(1)} node`;
    if(item.type==="text"&&isAICard(item)){
      const config=parseAICard(item);
      fields.push({key:"aiTitle",label:"Operator name",control:"text",value:config.title},{key:"aiPrompt",label:"AI instructions",control:"textarea",value:config.prompt});
      notes.push({text:"Incoming connections become context. The generated note updates automatically when that context changes."});
    }else if(task){
      const statuses=["inbox","next","scheduled","waiting","done","cancelled"];
      fields.push(
        {key:"title",label:"Task title",control:"text",value:task.title,scope:"task",taskId:task.id},
        {key:"status",label:"Task status",control:"select",value:task.status,scope:"task",taskId:task.id,options:statuses.map(status=>({value:status,label:status[0].toUpperCase()+status.slice(1)}))},
        {key:"scheduledOn",label:"Scheduled",control:"date",value:task.scheduledOn||"",scope:"task",taskId:task.id,row:"task-dates"},
        {key:"dueOn",label:"Due",control:"date",value:task.dueOn||"",scope:"task",taskId:task.id,row:"task-dates"},
        {key:"priority",label:"Priority",control:"select",value:task.priority??"",scope:"task",taskId:task.id,options:[{value:"",label:"None"},{value:"1",label:"High"},{value:"2",label:"Medium"},{value:"3",label:"Low"}]},
      );
      actions.push({intent:"delete-task",label:"Delete task everywhere",taskId:task.id});
    }else if(item.type==="text")fields.push({key:"text",label:"Markdown",control:"textarea",value:item.text});
    if(item.type==="link")fields.push({key:"url",label:"URL",control:"url",value:item.url});
    if(item.type==="file"&&!task){
      const subcanvasId=subcanvasIdFromNode(item),subcanvas=subcanvasId&&workspace.canvases[subcanvasId];
      if(subcanvas){
        fields.push({key:"title",label:"Canvas title",control:"text",value:subcanvas.title,scope:"canvas",canvasId:subcanvasId});
        notes.push({text:"This portal is a standard JSON Canvas file node. Double-click it or zoom in to enter the nested canvas."});
        actions.push({intent:"open-canvas",label:"Open sub-canvas ↘",canvasId:subcanvasId,requiresWrite:false,className:"button open-subcanvas-inspector"});
      }else{
        fields.push({key:"file",label:"File path",control:"text",value:item.file},{key:"subpath",label:"Subpath",control:"text",value:item.subpath||""});
        if(componentCard){
          notes.push({text:"This node is one placement of a canonical component card. Delete node removes only this placement."});
          actions.push({intent:"delete-card",label:"Delete card everywhere",cardId:componentCard.id,danger:true});
        }
      }
    }
    if(item.type==="group")fields.push({key:"label",label:"Label",control:"text",value:item.label||""},{key:"background",label:"Background path",control:"text",value:item.background||""});
    fields.push(
      {key:"x",label:"X",control:"number",value:item.x,row:"position"},
      {key:"y",label:"Y",control:"number",value:item.y,row:"position"},
      {key:"width",label:"Width",control:"number",value:item.width,row:"size"},
      {key:"height",label:"Height",control:"number",value:item.height,row:"size"},
    );
  }else{
    const sideOptions=["","top","right","bottom","left"].map(value=>({value,label:value||"Auto"}));
    fields.push(
      {key:"label",label:"Label",control:"text",value:item.label||""},
      {key:"fromSide",label:"From side",control:"select",value:item.fromSide||"",options:sideOptions,row:"sides"},
      {key:"toSide",label:"To side",control:"select",value:item.toSide||"",options:sideOptions,row:"sides"},
    );
  }
  actions.push({intent:"delete-selection",label:selected.kind==="node"?"Delete node":"Delete connection",danger:true});
  setInspectorModel(panel,{
    key:`${selected.kind}:${selected.id}`,
    kind:selected.kind,
    title,
    readonly:!canonicalWritable,
    readonlyMessage:"Canonical files are read-only until repaired or restored.",
    fields,
    notes,
    colors:[...Object.entries(COLORS).map(([value,color])=>({value,color,active:item.color===value})),{value:DORMANT_NODE_COLOR,color:DORMANT_NODE_COLOR,active:item.color===DORMANT_NODE_COLOR}],
    actions,
  });
}
function inspectorSelection(detail){
  if(!selected||detail?.modelKey!==`${selected.kind}:${selected.id}`)return null;
  return selected.kind==="node"?documentData.nodes.find(node=>node.id===selected.id):documentData.edges.find(edge=>edge.id===selected.id);
}
function taskPatchFromInspector(detail){
  if(!detail.taskId)return null;
  const value=detail.key==="priority"?(detail.value?Number(detail.value):null):detail.value||null;
  const patch={[detail.key]:value};
  if(detail.key==="status")patch.completedAt=value==="done"?new Date().toISOString():null;
  return {id:detail.taskId,patch};
}
function applyInspectorField(detail,phase){
  const item=inspectorSelection(detail);if(!item)return;
  if(detail.scope==="task"){
    const next=taskPatchFromInspector(detail);if(!next)return;
    if(phase==="input")scheduleTaskFieldUpdate(next.id,next.patch);
    else if(phase==="change")flushTaskFieldUpdate(next.id,next.patch);
    else if(taskUpdateTimers.has(next.id))flushTaskFieldUpdate(next.id,next.patch);
    return;
  }
  if(detail.scope==="canvas"){
    if(phase!=="input")return;
    const record=workspace.canvases[detail.canvasId];if(!record)return;
    const value=detail.value||"Untitled";
    record.title=value;
    scheduleSave();renderNodes();renderWorkspaceNavigation();return;
  }
  if(phase!=="input")return;
  const before=aiCardSignatures(),key=detail.key;
  if(key==="aiTitle"||key==="aiPrompt"){const config=parseAICard(item);item.text=buildAICardText(key==="aiTitle"?detail.value:config.title,key==="aiPrompt"?detail.value:config.prompt);}
  else if((key==="fromSide"||key==="toSide")&&!detail.value)delete item[key];
  else item[key]=detail.value;
  scheduleSave();renderNodes();renderEdges();renderMinimap();scheduleChangedAICards(before);
}
document.addEventListener("balaur-inspector-field-input",event=>applyInspectorField(event.detail,"input"));
document.addEventListener("balaur-inspector-field-change",event=>applyInspectorField(event.detail,"change"));
document.addEventListener("balaur-inspector-field-blur",event=>applyInspectorField(event.detail,"blur"));
document.addEventListener("balaur-inspector-close",event=>{if(event.target!==$("#inspector"))return;selected=null;shell.classList.remove("inspector-open");render();});
document.addEventListener("balaur-inspector-color",event=>{const item=inspectorSelection(event.detail);if(!item)return;item.color=event.detail.value;scheduleSave();render();});
document.addEventListener("balaur-inspector-action",event=>{
  const item=inspectorSelection(event.detail);if(!item)return;
  if(event.detail.intent==="open-canvas"&&event.detail.canvasId)enterSubcanvas(event.detail.canvasId);
  else if(event.detail.intent==="delete-task")deleteTaskEverywhere(event.detail.taskId);
  else if(event.detail.intent==="delete-card")deleteComponentCardEverywhere(event.detail.cardId);
  else if(event.detail.intent==="delete-selection")deleteSelection();
});
document.addEventListener("balaur-task-complete",async event=>{
  const host=event.target;if(!(host instanceof HTMLElement)||!host.matches("balaur-task-list")||!host.isConnected)return;
  const id=event.detail?.taskId;if(!lifeQuery?.tasks().some(task=>task.id===id))return;
  try{await completeTask(id);toast("Task completed");}catch(error){toast(error.message);}
});
document.addEventListener("balaur-task-open",event=>{
  const host=event.target;if(!(host instanceof HTMLElement)||!host.matches("balaur-task-list")||!host.isConnected)return;
  const task=lifeQuery?.tasks().find(item=>item.id===event.detail?.taskId),placement=taskPlacement(task);if(!placement)return;
  setAppView("canvas");revealWorkspaceNode(placement.canvasId,placement.nodeId);
});
document.addEventListener("balaur-canvas-open",event=>{
  const host=event.target;if(!(host instanceof HTMLElement)||!host.matches("balaur-workspace-nav")||!host.isConnected)return;
  const id=event.detail?.canvasId;if(!workspace.canvases[id])return;
  switchCanvas(id,{direction:"switch"});
});
async function deleteComponentCardEverywhere(cardId){
  if(!cardId||!componentCardRepository)return;
  if(!confirm("Delete this component card, its canonical file, and every canvas placement? This cannot be undone."))return;
  const card=componentCardCatalog?.getById(cardId);
  const affected=[...new Set((card?.placements||[]).map(placement=>Object.values(workspace.canvases).find(record=>record.path===placement.canvasPath)?.id).filter(Boolean))];
  try{
    await flushPendingWorkspaceEdits();
    await enqueueMutation(async()=>{await componentCardRepository.deleteCard(cardId);await reloadCanvasDocuments(affected);});
    selected=null;shell.classList.remove("inspector-open");render();toast("Component card deleted everywhere");
  }catch(error){toast(error.message);}
}

async function deleteTaskEverywhere(taskId){
  if(!taskId||!taskRepository)return;
  if(!confirm("Delete this task, its canonical file, and every canvas placement? This cannot be undone."))return;
  const affected=lifeIndex?.placementsForEntity(taskId).map((placement)=>placement.canvasId)||[];
  try{
    await flushPendingWorkspaceEdits();
    await enqueueMutation(async()=>{await taskRepository.deleteTask(taskId);await reloadCanvasDocuments(affected);});
    selected=null;shell.classList.remove("inspector-open");render();renderToday();toast("Task deleted everywhere");
  }catch(error){toast(error.message);}
}
async function deleteSelection() {
  if (!canonicalWritable) { toast("Canonical files are read-only until repaired or restored"); return; }
  if (!selected) return;const before=aiCardSignatures();let canonicalMutation=false;
  if (selected.kind==="node") {
    const node=documentData.nodes.find(item=>item.id===selected.id),subcanvasId=subcanvasIdFromNode(node),taskId=taskIdFromNode(node);
    if(subcanvasId&&!confirm(`Delete “${workspace.canvases[subcanvasId].title}” and every canvas nested inside it?`))return;
    if(subcanvasId)deleteCanvasTree(subcanvasId);
    if(taskId){
      canonicalMutation=true;
      try{
        await flushPendingWorkspaceEdits();
        await enqueueMutation(async()=>{await taskRepository.removePlacement(currentCanvasId,selected.id);await reloadCanvasDocuments([currentCanvasId]);});
      }catch(error){toast(error.message);return;}
    } else {
      documentData.nodes=documentData.nodes.filter(n=>n.id!==selected.id);
      documentData.edges=(documentData.edges||[]).filter(e=>e.fromNode!==selected.id&&e.toNode!==selected.id);
    }
  } else documentData.edges=documentData.edges.filter(e=>e.id!==selected.id);
  selected=null;shell.classList.remove("inspector-open");if(!canonicalMutation) scheduleSave();render();scheduleChangedAICards(before);toast("Deleted");
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

function downloadJSON(data,filename){const blob=new Blob([JSON.stringify(data,null,2)],{type:"application/json"}),anchor=document.createElement("a");anchor.href=URL.createObjectURL(blob);anchor.download=filename;anchor.click();setTimeout(()=>URL.revokeObjectURL(anchor.href),0);}
function exportCanvas() {
  downloadJSON(documentData,slug($("#canvasTitle").value||"life-canvas")+".canvas");toast("Current canvas exported");
}
async function exportWorkspace(){
  if(!vaultStore)throw new Error("Canonical files are unavailable.");
  await flushPendingWorkspaceEdits();
  saveCurrentCanvasState(); workspace.activeId=currentCanvasId; await persistWorkspace();
  const {bundle,diagnostics}=await exportBundle(vaultStore.vault);
  assertCompleteExport(diagnostics);
  downloadJSON(JSON.parse(serializeBundle(bundle)),`${slug(workspace.canvases[workspace.rootId].title)}.orbit.json`);
  toast(`${Object.keys(workspace.canvases).length} canvases and canonical files exported`);
}
async function importCanvas(file) {
  try {
    const parsed=JSON.parse(await file.text());
    if(parsed?.format==="orbit-workspace"){
      if(!confirm("Import this whole Balaur space? Your current canonical vault will be replaced."))return;
      await flushPendingWorkspaceEdits();
      const stagingVault=new IndexedDbVault(`orbit-vault-${uid("import")}`);
      await importBundle(stagingVault,JSON.stringify(parsed));
      // Validate and index the complete staging space before touching the
      // canonical vault. IndexedDB restore is one transaction, so reloads use
      // the same orbit-vault name rather than a stranded import database.
      const stagingIndex=new MemoryIndex();
      const stagingCanvasIds=new Map(Object.values(parsed.workspace.canvases||{}).map(record=>[record.path,record.id]));
      const stagingCanvasIdFromPath=path=>stagingCanvasIds.get(path)||String(path).split("/").pop().replace(/\.canvas$/,"");
      const stagingIndexer=new LifeIndexer({vault:stagingVault,index:stagingIndex,canvasIdFromPath:stagingCanvasIdFromPath});
      await stagingIndexer.rebuild();
      const audit=await auditIndex(stagingVault,stagingIndex,{canvasIdFromPath:stagingCanvasIdFromPath});
      if(!audit.ok)throw new Error(`Imported space failed index audit: ${audit.problems.map(problem=>problem.message).join("; ")}`);
      const snapshot=await stagingVault.snapshot();
      const canonicalVault=new IndexedDbVault("orbit-vault");
      await canonicalVault.restore(snapshot);
      const nextStore=new WorkspaceStore(canonicalVault),result=await nextStore.load();
      if(!result?.workspace)throw new Error("Canonical vault activation did not produce a workspace");
      // Switch application globals only after canonical activation and reload
      // have both succeeded.
      vaultStore=nextStore; workspace=result.workspace; window.orbitVaultStore=vaultStore; configureLifeRuntime(canonicalVault); await Promise.all([lifeIndexer.rebuild(),componentCardCatalog.rebuild(),widgetCatalog.rebuild()]);
      currentCanvasId=workspace.activeId; documentData=workspace.canvases[currentCanvasId].document; camera=workspace.canvases[currentCanvasId].camera||{x:80,y:55,zoom:1}; selected=null; $("#canvasTitle").value=canvasRecord().title; render(); fitView();
      const stats=lifeIndexer.stats();setIndexStatus(`Files · ${stats.sourceFiles} indexed`,`${stats.tasks} tasks · ${stats.habits} habits · ${stats.diagnostics} diagnostics`);toast("Whole workspace and canonical files imported");return;
    }
    if(!isCanvas(parsed))throw new Error("Not a valid JSON Canvas document or version-2 Balaur workspace");
    documentData={nodes:parsed.nodes,edges:parsed.edges};workspace.canvases[currentCanvasId].document=documentData;selected=null;scheduleSave();render();fitView();toast("Canvas imported");
  } catch(error){alert(`Could not import this file.\n\n${error.message}`);}
}

// Canvas-aware assistant prototype. A remote model should produce these operations,
// never arbitrary host-page JavaScript. Each operation is checked before commit.
function idsForDraft(draft) {
  return new Set([...(draft?.nodes||[]),...(draft?.edges||[])].map(item=>item.id));
}
function prepareGeneratedOperation(operation, context) {
  const normalized=validateGeneratedOperation(operation,context);
  const prepared=normalized.type==="component-card.create"?{...normalized,card:{...normalized.card},placement:{...normalized.placement}}
    :normalized.type==="component-card.update"?{...normalized,patch:{...normalized.patch,...(normalized.patch.fields?{fields:{...normalized.patch.fields}}:{})},...(normalized.placement?{placement:{...normalized.placement}}:{})}
    :normalized.type==="widget.create"?{...normalized,widget:{...normalized.widget},placement:{...normalized.placement}}
    :{...normalized,placement:{...normalized.placement}};
  if(prepared.type==="component-card.create"){prepared.card.id ||= uid("card");prepared.placement.id ||= uid("node");}
  else if(prepared.placement)prepared.placement.id ||= uid("node");
  return validateGeneratedOperation(prepared,context);
}
function generatedOperationContext(operation, plannedCards, plannedDrafts) {
  const canvasId=operation.canvasId||currentCanvasId;
  if(operation.type==="component-card.update"&&!plannedCards.has(operation.id)){
    const card=componentCardCatalog?.getById(operation.id);if(card)plannedCards.set(card.id,card);
  }
  return {canvasId,canvasIds:new Set(Object.keys(workspace.canvases)),cards:plannedCards,cardIds:new Set(plannedCards.keys()),widgetPaths:new Set((widgetCatalog?.widgets()||[]).map(widget=>widget.path)),nodeIds:idsForDraft(plannedDrafts.get(canvasId))};
}
function evolvePlannedCard(operation, plannedCards) {
  if(operation.type==="component-card.create"){
    plannedCards.set(operation.card.id,{id:operation.card.id,path:operation.card.path,title:operation.card.title,recipe:operation.card.recipe,...operation.card.fields,body:operation.card.body});
    return;
  }
  if(operation.type!=="component-card.update")return;
  const current=plannedCards.get(operation.id),patch=operation.patch,next={...current,...patch,...(patch.fields||{})};
  if(patch.title)next.path=componentCardPath(patch.title,operation.id);
  plannedCards.set(operation.id,next);
}
function validateCanvasOperations(operations) {
  assertPlainDataTree(operations,"Generated operation plan");
  if(!Array.isArray(operations)||operations.length>50||JSON.stringify(operations).length>160*1024)throw new Error("The operation plan is too large or malformed");
  const themes=[],generatedOperations=[],normalizedOperations=[],nodeKeys=new Set(["text","file","subpath","url","label","background","backgroundStyle","x","y","width","height","color"]),edgeKeys=new Set(["fromSide","fromEnd","toSide","toEnd","color","label"]),plannedCards=new Map((componentCardCatalog?.cards()||[]).map(card=>[card.id,card])),plannedDrafts=new Map(Object.entries(workspace.canvases).map(([id,record])=>[id,clone(record.document)])),affectedCanvasIds=new Set(),draft=plannedDrafts.get(currentCanvasId);
  for (const sourceOperation of operations) {
    if (!sourceOperation || typeof sourceOperation.type!=="string") throw new Error("Malformed canvas operation");
    if(sourceOperation.type==="component-card.create"||sourceOperation.type==="component-card.update"||sourceOperation.type==="widget.create"||sourceOperation.type==="widget.place"){
      const context=generatedOperationContext(sourceOperation,plannedCards,plannedDrafts),normalized=prepareGeneratedOperation(sourceOperation,context);generatedOperations.push(normalized);normalizedOperations.push(normalized);evolvePlannedCard(normalized,plannedCards);
      if(normalized.placement){
        const targetDraft=plannedDrafts.get(normalized.canvasId),path=normalized.type==="widget.create"?normalized.widget.path:normalized.type==="widget.place"?normalized.path:plannedCards.get(normalized.type==="component-card.create"?normalized.card.id:normalized.id).path;
        targetDraft.nodes.push({id:normalized.placement.id,type:"file",file:path,x:normalized.placement.x,y:normalized.placement.y,width:normalized.placement.width,height:normalized.placement.height,...(normalized.placement.color?{color:normalized.placement.color}:{})});
        affectedCanvasIds.add(normalized.canvasId);
      }
      continue;
    }
    const operation=clone(sourceOperation);normalizedOperations.push(operation);
    if (operation.type==="node.add") {
      if(idsForDraft(draft).has(operation.node?.id))throw new Error(`Canvas id already exists: ${operation.node?.id}`);draft.nodes.push(clone(operation.node));affectedCanvasIds.add(currentCanvasId);
    } else if (operation.type==="node.update") {
      const node=draft.nodes.find(item=>item.id===operation.id);if(!node)throw new Error(`Unknown node ${operation.id}`);
      for(const [key,value] of Object.entries(operation.patch||{})){if(!nodeKeys.has(key))throw new Error(`Field ${key} cannot be changed`);node[key]=value;}affectedCanvasIds.add(currentCanvasId);
    } else if (operation.type==="node.remove") {
      if(!draft.nodes.some(item=>item.id===operation.id))throw new Error(`Unknown node ${operation.id}`);
      draft.nodes=draft.nodes.filter(item=>item.id!==operation.id);draft.edges=draft.edges.filter(edge=>edge.fromNode!==operation.id&&edge.toNode!==operation.id);affectedCanvasIds.add(currentCanvasId);
    } else if (operation.type==="edge.add") {
      if(idsForDraft(draft).has(operation.edge?.id))throw new Error(`Canvas id already exists: ${operation.edge?.id}`);draft.edges.push(clone(operation.edge));affectedCanvasIds.add(currentCanvasId);
    } else if (operation.type==="edge.update") {
      const edge=draft.edges.find(item=>item.id===operation.id);if(!edge)throw new Error(`Unknown edge ${operation.id}`);
      for(const [key,value] of Object.entries(operation.patch||{})){if(!edgeKeys.has(key))throw new Error(`Field ${key} cannot be changed`);edge[key]=value;}affectedCanvasIds.add(currentCanvasId);
    } else if(operation.type==="edge.remove"){
      if(!draft.edges.some(item=>item.id===operation.id))throw new Error(`Unknown edge ${operation.id}`);
      draft.edges=draft.edges.filter(item=>item.id!==operation.id);affectedCanvasIds.add(currentCanvasId);
    } else if (operation.type==="theme.set") {
      if(!["default","warm","calm","contrast"].includes(operation.theme))throw new Error("Unknown theme");themes.push(operation.theme);
    } else throw new Error(`Unsupported operation ${operation.type}`);
  }
  for(const canvasId of affectedCanvasIds)if(!isCanvas(plannedDrafts.get(canvasId)))throw new Error(`The resulting canvas ${canvasId} is not valid JSON Canvas 1.0`);
  return {draft,themes,generatedOperations,normalizedOperations};
}
function repositoryPatch(patch) {
  const result={...patch,...(patch.fields||{})};delete result.fields;return result;
}
async function applyGeneratedOperation(operation) {
  if(operation.type==="widget.create"||operation.type==="widget.place"){
    if(!widgetRepository)throw new Error("Widget files are unavailable.");
    if(operation.type==="widget.create")return widgetRepository.createWidget({...operation.widget,canvasId:operation.canvasId,geometry:operation.placement});
    return widgetRepository.addPlacement(operation.path,operation.canvasId,operation.placement);
  }
  if(!componentCardRepository)throw new Error("Component-card files are unavailable.");
  if(operation.type==="component-card.create")return componentCardRepository.createCard({...operation.card,canvasId:operation.canvasId,geometry:operation.placement});
  const patch=repositoryPatch(operation.patch);
  if(operation.placement)return componentCardRepository.updateCardAndPlace(operation.id,patch,operation.canvasId,operation.placement);
  return Object.keys(patch).length?componentCardRepository.updateCard(operation.id,patch):componentCardCatalog.getById(operation.id);
}
function applyValidatedCanvasOperation(operation) {
  if(operation.type==="theme.set"){applyCanvasTheme(operation.theme);return false;}
  if(operation.type==="node.add")documentData.nodes.push(clone(operation.node));
  else if(operation.type==="node.update"){const node=documentData.nodes.find(item=>item.id===operation.id);Object.assign(node,operation.patch);}
  else if(operation.type==="node.remove"){documentData.nodes=documentData.nodes.filter(item=>item.id!==operation.id);documentData.edges=documentData.edges.filter(edge=>edge.fromNode!==operation.id&&edge.toNode!==operation.id);}
  else if(operation.type==="edge.add")documentData.edges.push(clone(operation.edge));
  else if(operation.type==="edge.update"){const edge=documentData.edges.find(item=>item.id===operation.id);Object.assign(edge,operation.patch);}
  else if(operation.type==="edge.remove")documentData.edges=documentData.edges.filter(item=>item.id!==operation.id);
  workspace.canvases[currentCanvasId].document=documentData;return true;
}
async function applyCanvasOperations(operations) {
  if(!canonicalWritable)throw new Error("Canonical files are read-only until repaired or restored.");
  const before=aiCardSignatures(),plan=validateCanvasOperations(operations);
  await flushPendingWorkspaceEdits();
  let failure=null,dirty=false,dirtyStart=null;
  const saveDirty=async()=>{
    if(!dirty)return;
    try{await markSaveResult(persistWorkspace());dirty=false;dirtyStart=null;}
    catch(error){error.operationState={operations:plan.normalizedOperations,retryIndex:dirtyStart,appliedCount:dirtyStart,reload:true};throw error;}
  };
  try{
    for(let operationIndex=0;operationIndex<plan.normalizedOperations.length;operationIndex+=1){
      const operation=plan.normalizedOperations[operationIndex];
      if(operation.type.startsWith("component-card.")||operation.type==="widget.create"||operation.type==="widget.place"){
        await saveDirty();
        try{await enqueueMutation(()=>applyGeneratedOperation(operation));}
        catch(error){error.operationState={operations:plan.normalizedOperations,failedIndex:operationIndex,retryIndex:operationIndex,appliedCount:operationIndex};throw error;}
        await reloadCanvasDocuments(Object.keys(workspace.canvases));
      }else{
        const changed=applyValidatedCanvasOperation(operation);
        if(changed&&!dirty)dirtyStart=operationIndex;
        dirty=changed||dirty;
      }
    }
    await saveDirty();
  }catch(error){failure=error;}
  finally{
    if(plan.generatedOperations.length||failure?.operationState?.reload){
      try{await Promise.all([componentCardCatalog.rebuild(),widgetCatalog.rebuild()]);await reloadCanvasDocuments(Object.keys(workspace.canvases));}
      catch(error){if(!failure)failure=error;else console.warn("Could not reconcile a partially applied component-card plan",error);}
    }
    selected=null;shell.classList.remove("inspector-open");render();updateAssistantContext();scheduleChangedAICards(before);
  }
  if(failure)throw failure;
  return plan.normalizedOperations;
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
const GRAPH_MEMORY_DEPTH=2, GRAPH_MEMORY_NODE_CAP=60;
function graphMemoryDigest({maxDepth=GRAPH_MEMORY_DEPTH,nodeCap=GRAPH_MEMORY_NODE_CAP}={}){
  const lines=[],visited=new Set();let nodeCount=0;
  const edgeLabelsFrom=(doc,nodeId)=>(doc.edges||[]).filter(e=>e.fromNode===nodeId&&e.label&&e.label!=="AI output").map(e=>{const to=(doc.nodes||[]).find(n=>n.id===e.toNode);return to?`${e.label} → ${nodeTitle(to)}`:null;}).filter(Boolean);
  const visit=(canvasId,depth)=>{
    const record=workspace.canvases[canvasId];
    if(!record||visited.has(canvasId)||depth>maxDepth)return;
    visited.add(canvasId);
    const kind=canvasKind(record);
    lines.push(`${"  ".repeat(depth)}${record.title}${kind?` (${kind})`:canvasId===workspace.rootId?" (home)":""}`);
    for(const node of record.document.nodes||[]){
      if(nodeCount>=nodeCap)return;
      if(node.type==="group")continue;
      nodeCount++;
      const marker=noteKind(node)?`[${noteKind(node)}] `:node.type==="file"?"[file] ":"";
      lines.push(`${"  ".repeat(depth+1)}- ${marker}${nodeSummary(node)}`);
      for(const rel of edgeLabelsFrom(record.document,node.id))lines.push(`${"  ".repeat(depth+2)}${rel}`);
      const sub=subcanvasIdFromNode(node);
      if(sub)visit(sub,depth+1);
    }
  };
  visit(workspace.rootId,0);
  if(nodeCount>=nodeCap)lines.push(`(truncated at ${nodeCap} nodes)`);
  return lines.join("\n");
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
  if(operation.type==="component-card.create"||operation.type==="component-card.update"||operation.type==="widget.create"){
    const description=describeGeneratedOperation(operation),source=description.source;
    return `<div class="${source?"widget-operation-review":""}"><b>${escapeHTML(description.title)}</b> · ${escapeHTML(description.summary)}${description.details.length?`<small>${description.details.map(escapeHTML).join(" · ")}</small>`:""}${source?`<details><summary>Review complete source (${source.length} characters)</summary><pre>${escapeHTML(source)}</pre></details>`:""}</div>`;
  }
  const names={"node.add":"Add node","node.update":"Update node","node.remove":"Delete node","edge.add":"Add connection","edge.update":"Update connection","edge.remove":"Delete connection","theme.set":"Set theme"};
  const target=operation.id||operation.node?.id||operation.edge?.id||operation.theme||"";return `<div><b>${escapeHTML(names[operation.type]||operation.type)}</b>${target?` · ${escapeHTML(target)}`:""}</div>`;
}
function assistantProposal(text,operations) {
  const plan=validateCanvasOperations(operations),normalized=plan.normalizedOperations,message=document.createElement("div");
  let pendingOperations=normalized,appliedCount=0,durablePartial=null;
  message.className="ai-message assistant";message.innerHTML=`<span>✦</span><div class="ai-proposal"><p></p>${normalized.length?`<div class="ai-operation-list">${normalized.map(operationDescription).join("")}</div><div class="ai-proposal-actions"><button class="apply">Apply ${normalized.length} change${normalized.length===1?"":"s"}</button><button class="discard">Discard</button></div>`:""}</div>`;$("p",message).textContent=text||"I reviewed the canvas.";$("#aiMessages").append(message);
  if(normalized.length){
    const apply=$(".apply",message),discard=$(".discard",message),list=$(".ai-operation-list",message),renderPending=()=>{list.innerHTML=`${appliedCount?`<div><b>${appliedCount} earlier change${appliedCount===1?"":"s"} applied</b></div>`:""}${pendingOperations.map(operationDescription).join("")}`;};
    apply.onclick=async()=>{
      apply.disabled=true;discard.disabled=true;apply.textContent="Applying…";
      try{await applyCanvasOperations(pendingOperations);appliedCount+=pendingOperations.length;pendingOperations=[];durablePartial=null;apply.textContent="Applied";discard.remove();renderPending();toast("AI changes applied");}
      catch(error){
        const state=error.operationState,recoverable=error.details?.recoverable;
        if(state)appliedCount+=state.appliedCount??state.failedIndex??0;
        if(recoverable&&Number.isInteger(state?.failedIndex)){
          const widgetFailure=state.operations[state.failedIndex]?.type==="widget.create",savedLabel=widgetFailure?"widget":"card";
          pendingOperations=recoverGeneratedPlacementFailure(state.operations,state.failedIndex,recoverable);renderPending();durablePartial=recoverable.updated?"saved update":`saved ${savedLabel}`;discard.textContent=`Keep ${durablePartial}`;
          const suffix=pendingOperations.length-1,subject=recoverable.updated?"The canonical card update":`The ${savedLabel} file`;
          assistantMessage(`${subject} was saved at ${recoverable.path}, but its Canvas placement failed. ${appliedCount?`${appliedCount} earlier change${appliedCount===1?" was":"s were"} already applied. `:""}Choose “Place saved ${savedLabel}${suffix?" + continue":""}” to retry only the unfinished placement${suffix?` and then apply ${suffix} untouched remaining change${suffix===1?"":"s"}`:""}; the durable ${recoverable.updated?"patch will not run again":"file will not be recreated"}.`);apply.textContent=`Place saved ${savedLabel}${suffix?" + continue":""}`;
        }else{
          if(state){pendingOperations=state.operations.slice(state.retryIndex??state.failedIndex);renderPending();}
          assistantMessage(`I could not apply that plan: ${error.message}`);apply.textContent="Try again";
        }
        apply.disabled=false;discard.disabled=false;
      }
    };
    discard.onclick=()=>{apply.disabled=true;discard.textContent=durablePartial?`${durablePartial[0].toUpperCase()}${durablePartial.slice(1)} kept`:"Discarded";discard.disabled=true;};
  }
  message.scrollIntoView({behavior:"smooth",block:"end"});return message;
}
function proposeLocalWidget() {
  const title="Canvas focus dial",id=uid("widget"),box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2);
  const source=`<!doctype html>
<title>${title}</title>
<style>:root{color-scheme:dark}body{margin:0;min-height:100vh;display:grid;place-items:center;background:var(--balaur-surface,#24150c);color:var(--balaur-content,#f1e7d4);font-family:var(--balaur-fontBody,system-ui)}focus-dial{display:grid;place-items:center;inline-size:12rem;aspect-ratio:1;border:3px solid var(--balaur-primary,#f2c14e);border-radius:50%;box-shadow:inset 0 0 0 1rem color-mix(in srgb,var(--balaur-primary,#f2c14e) 14%,transparent)}strong{font-size:2rem}</style>
<focus-dial><strong>72%</strong><span>Focused</span></focus-dial>
<script>customElements.define(\"focus-dial\",class extends HTMLElement{});<\/script>`;
  const operation={type:"widget.create",widget:{path:`widgets/${slugify(title)}--${id}.html`,title,source},canvasId:currentCanvasId,placement:{id:uid("node"),x:Math.round(center.x-210),y:Math.round(center.y-130),width:420,height:260,color:"5"}};
  setAssistantOpen(true);assistantProposal("I prepared a self-contained live widget. Review the complete source and capability disclosure. It will not be written or executed until you approve it.",[operation]);
}

async function runLocalAssistant(prompt) {
  const request=prompt.trim();if(!request)return;assistantMessage(request,"user");const lower=request.toLowerCase();let response="";
  try {
    if(/summar|what(?:'s| is) on|parse/.test(lower)) {
      const s=canvasSummary();response=`I parsed the current JSON Canvas: ${s.nodes} content nodes and ${s.edges} connections. I found ${s.goals} goals, ${s.projects} projects, ${s.habits} habits, ${s.ideas} ideas, ${s.widgets} live widgets, and ${s.openTasks} unchecked tasks.`;
    } else if(/graph|memory|my life|what(?:'s| is) in|overview/.test(lower)){
      response=`Here is your graph, traversed from Home (depth ${GRAPH_MEMORY_DEPTH}):\n\n${graphMemoryDigest()}`;
    } else if(/(?:add|create).*(?:metric card|metric)/.test(lower)){
      const named=request.match(/(?:called|named)\s+(.+?)(?:\s+and\s+(?:set|use|make).*)?$/i)?.[1]?.replace(/[.!]$/,"")||"Weekly metric",box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2),operations=[{type:"component-card.create",card:{id:uid("card"),title:named,recipe:"metric",fields:{value:"72%",label:"Current progress",progress:.72,trend:"up"},body:"Created locally without contacting an AI provider."},canvasId:currentCanvasId,placement:{id:uid("node"),x:Math.round(center.x-180),y:Math.round(center.y-110),width:360,height:220,color:"5"}}],theme=/warm|cozy|earth/.test(lower)?"warm":/calm|ocean|cool|teal/.test(lower)?"calm":/contrast|accessible/.test(lower)?"contrast":null;
      if(theme)operations.push({type:"theme.set",theme});assistantProposal("I prepared a local declarative metric card. Review every change before writing the canonical file.",operations);return;
    } else if(/(?:add|create).*(?:3d|live )?widget/.test(lower)){
      proposeLocalWidget();return;
    } else if(/warm|cozy|earth/.test(lower)) {await applyCanvasOperations([{type:"theme.set",theme:"warm"}]);response="Applied a warmer, earth-toned canvas theme. This visual preference stays separate from the portable .canvas document.";
    } else if(/calm|ocean|cool|teal/.test(lower)) {await applyCanvasOperations([{type:"theme.set",theme:"calm"}]);response="Applied the calm teal canvas theme.";
    } else if(/contrast|accessible/.test(lower)) {await applyCanvasOperations([{type:"theme.set",theme:"contrast"}]);response="Applied the high-contrast canvas theme.";
    } else if(/reset.*(?:theme|style)|default (?:theme|style)/.test(lower)) {await applyCanvasOperations([{type:"theme.set",theme:"default"}]);response="Reset the canvas styling to its default theme.";
    } else if(/(?:add|create).*(?:sub.?canvas|nested canvas)/.test(lower)){createSubcanvas();response="Created a nested canvas portal. Double-click it or zoom into it to enter.";
    } else if(/(?:add|create).*(?:3d|webgl|html|widget)/.test(lower)) {
      const center=canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),node={id:uid("node"),type:"file",x:Math.round(center.x-240),y:Math.round(center.y-145),width:480,height:290,color:"5",file:"widgets/focus-orbit.html"};await applyCanvasOperations([{type:"node.add",node}]);response="Added a sandboxed WebGL file node. It is still a standard JSON Canvas file node pointing to an HTML file.";
    } else {
      const match=request.match(/(?:add|create)\s+(?:a |an )?(goal|habit|project|note)(?:\s+(?:called|named|to))?\s+(.+)/i);
      if(match){const kind=match[1].toLowerCase(),title=match[2].replace(/[.!]$/,"");const preset={goal:["1",`# ${title}\nWhat does success look like?\n\n- [ ] Choose the first step\n\nProgress: 0%`],habit:["4",`# ${title}\nMake the practice small and repeatable.`],project:["6",`# ${title}\nDefine the desired outcome.\n\n- [ ] First milestone\n\nProgress: 0%`],note:["2",`# ${title}\nStart writing here…`]}[kind];const center=canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),node={id:uid("node"),type:"text",x:Math.round(center.x-150),y:Math.round(center.y-90),width:300,height:kind==="project"||kind==="goal"?200:150,color:preset[0],text:preset[1]};await applyCanvasOperations([{type:"node.add",node}]);response=`Added a ${kind} card using standard JSON Canvas fields.`;}
      else response="I understand this canvas, but the GitHub Pages demo uses a local intent parser rather than a remote model. Try asking me to summarize it, create a metric card, add a goal/habit/project, add a 3D widget, or change the theme.";
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
  return `You are Balaur, an assistant operating a JSON Canvas 1.0 life-management canvas. Respond with exactly one JSON object and no markdown fences: {"message":"Brief response to the user","operations":[]}.
Allowed operations:
{"type":"node.add","node":<complete JSON Canvas node with unique id, type, integer x/y/width/height and required type field>}
{"type":"node.update","id":"existing id","patch":<changed standard fields>}
{"type":"node.remove","id":"existing id"}
{"type":"edge.add","edge":<complete JSON Canvas edge with unique id>}
{"type":"edge.update","id":"existing id","patch":<changed edge fields>}
{"type":"edge.remove","id":"existing id"}
{"type":"theme.set","theme":"default|warm|calm|contrast"}
{"type":"component-card.create","card":{"title":"title","recipe":"metric|progress|callout|list|timeline","fields":<recipe fields>,"body":"optional Markdown"},"canvasId":"target canvas id","placement":{"x":0,"y":0,"width":360,"height":220,"color":"1-6 or #rrggbb"}}
{"type":"component-card.update","id":"existing component-card id","patch":{"title":"optional","recipe":"optional","fields":<changed recipe fields>,"body":"optional"},"canvasId":"optional target canvas id","placement":{"id":"optional unique placement id","x":0,"y":0,"width":360,"height":220,"color":"optional 1-6 or #rrggbb"}}
{"type":"widget.create","widget":{"path":"widgets/safe-stable-name.html","title":"title exactly matching the HTML title","source":"complete self-contained reviewed HTML"},"canvasId":"target canvas id","placement":{"x":0,"y":0,"width":420,"height":260,"color":"1-6 or #rrggbb"}}
An update may contain only canvasId plus placement to add another placement without changing the canonical card fields.
Component-card deletion is not a generated operation; it remains a separate confirmed action. Component cards are declarative data only: never return HTML, JavaScript, event handlers, host code, repository calls, or source fields. A widget.create source must be self-contained, include a matching non-empty title, use no external resources/network/navigation/forms/workers/nested frames, and handle reduced motion when animated. Widget code executes only after complete source review, explicit approval, and a second explicit Run action inside sandbox="allow-scripts"; it never receives host data or mutation access. Use only standard JSON Canvas fields for Canvas nodes. Use Markdown checkboxes in text nodes for tasks. Colors: 1 red/goals, 2 orange/ideas, 3 yellow/notes, 4 green/habits, 5 cyan/resources, 6 purple/projects. Edge labels are a convention, not a schema: part-of (structural), relates-to (associative), filed-to (lifecycle). AI output is reserved. Propose changes only as the allowed operations; never auto-tag or auto-link. Preserve user data unless explicitly asked to remove it. Ask a question with an empty operations array if intent is ambiguous. Never put executable HTML or JavaScript in a text node. Keep responses concise.`;
}
function parseProviderJSON(content) {
  if(Array.isArray(content))content=content.map(part=>part.text||part.content||"").join("");if(typeof content!=="string")throw new Error("Provider returned no text content");
  const cleaned=content.trim().replace(/^```(?:json)?\s*/i,"").replace(/\s*```$/,"");const start=cleaned.indexOf("{"),end=cleaned.lastIndexOf("}");if(start<0||end<start)throw new Error("Provider did not return the requested JSON plan");
  const parsed=JSON.parse(cleaned.slice(start,end+1));if(typeof parsed.message!=="string"||!Array.isArray(parsed.operations))throw new Error("Provider response is missing message or operations");return parsed;
}
async function runRemoteAssistant(prompt) {
  assistantMessage(prompt,"user");const loading=assistantMessage("Thinking…");loading.classList.add("loading");const send=$("#aiForm button");send.disabled=true;
  try{
    const box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2),cards=[...new Map((documentData.nodes||[]).map(node=>[node.file,componentCardCatalog?.getByPath(node.file)]).filter(([,card])=>card).map(([,card])=>[card.id,card])).values()].map(card=>({id:card.id,title:card.title,recipe:card.recipe,value:card.value,label:card.label,progress:card.progress,trend:card.trend,maximum:card.maximum,unit:card.unit,tone:card.tone,path:card.path})),memory=graphMemoryDigest(),context=`Graph memory (traversed from Home, depth ${GRAPH_MEMORY_DEPTH}):\n${memory}\n\nCurrent canvas: ${canvasTrail().map(record=>record.title).join(" / ")}\nCurrent canvas id: ${currentCanvasId}\nCurrent viewport center: ${Math.round(center.x)}, ${Math.round(center.y)}.\nCurrent component cards:\n${JSON.stringify(cards)}\nCurrent JSON Canvas:\n${JSON.stringify(documentData)}`;
    const messages=[{role:"system",content:assistantSystemPrompt()},...aiConversation.slice(-8),{role:"user",content:`${prompt}\n\n${context}`}];
    const response=await providerFetch(aiSettings,"/chat/completions",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({model:aiSettings.model,messages,temperature:.2,max_tokens:1800})}),body=await response.json(),content=body.choices?.[0]?.message?.content,plan=parseProviderJSON(content);
    loading.remove();assistantProposal(plan.message,plan.operations);aiConversation.push({role:"user",content:prompt},{role:"assistant",content:JSON.stringify(plan)});
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
  const config=parseAICard(card),inputs=inputNodesForAICard(card.id);
  await preloadAIFileInputs(inputs);
  const signature=aiCardSignature(card);if(!manual&&state.lastSignature===signature){state.status="Up to date";aiCardRuntime.set(cardId,state);renderNodes();return;}
  state.running=true;state.pending=false;state.status=`Reading ${inputs.length} input${inputs.length===1?"":"s"}…`;aiCardRuntime.set(cardId,state);renderNodes();
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

applyCanvasTheme(localStorage.getItem("orbit-canvas-theme")||"default");updateProviderUI();

const ADD_KINDS=new Set(["note","goal","habit","project","inbox","reference","task","ai-note","ai","widget","subcanvas"]);
function runAddKind(kind){if(kind==="ai-note")openAINoteDialog();else if(kind==="widget")proposeLocalWidget();else addNode(kind);}
document.addEventListener("balaur-add",event=>{
  const menu=event.target,kind=event.detail?.kind;
  if(!(menu instanceof HTMLElement)||!menu.matches("balaur-add-menu")||!menu.isConnected||!ADD_KINDS.has(kind))return;
  runAddKind(kind);
});
document.addEventListener("click",event=>{
  if(componentDefined("balaur-add-menu"))return;
  const menu=event.target.closest?.("balaur-add-menu");if(!menu||!menu.isConnected)return;
  const toggle=event.target.closest?.("#addMenuToggle"),item=event.target.closest?.("[data-add]"),panel=$("#addMenu");
  if(toggle){const open=panel.hidden;panel.hidden=!open;toggle.setAttribute("aria-expanded",String(open));return;}
  const kind=item?.dataset.add;if(!item||!menu.contains(item)||!ADD_KINDS.has(kind))return;
  panel.hidden=true;$("#addMenuToggle")?.setAttribute("aria-expanded","false");runAddKind(kind);
});
$$('[data-app-view]').forEach(button=>button.onclick=()=>setAppView(button.dataset.appView));
$("#newGroup").onclick=()=>addNode("group");$("#newCanvas").onclick=()=>createSubcanvas();
$$(".nav-item[data-filter]").forEach(button=>button.onclick=()=>{activeFilter=button.dataset.filter;$$(".nav-item[data-filter]").forEach(b=>b.classList.toggle("active",b===button));renderNodes();renderEdges();});
$$(".tool:not(.add-menu-toggle)").forEach(button=>button.onclick=()=>{const tool=button.dataset.tool;if(tool==="note")setTool("note");else setTool(tool);});
$("#zoomIn").onclick=()=>setZoom(camera.zoom*1.2);$("#zoomOut").onclick=()=>setZoom(camera.zoom/1.2);$("#zoomLabel").onclick=()=>setZoom(1);$("#fitView").onclick=fitView;
$("#exportButton").onclick=exportCanvas;$("#exportWorkspaceButton").onclick=()=>exportWorkspace().catch(error=>{alert(`Could not export a complete backup.\n\n${error.message}`);});$("#importButton").onclick=()=>$("#fileInput").click();$("#fileInput").onchange=e=>{if(e.target.files[0])importCanvas(e.target.files[0]);e.target.value="";};
$("#sidebarToggle").onclick=()=>shell.classList.toggle("sidebar-closed");$("#sidebar").addEventListener("click",event=>{if(narrowShell.matches&&event.target.closest("button"))shell.classList.add("sidebar-closed");});
$("#assistantButton").onclick=()=>setAssistantOpen(!$("#aiPanel").classList.contains("open"));$("#closeAssistant").onclick=()=>setAssistantOpen(false);$("#openAISettings").onclick=openAISettings;
$("#aiForm").onsubmit=event=>{event.preventDefault();const input=$("#aiPrompt"),prompt=input.value;input.value="";runAssistant(prompt);};
$("#aiPrompt").onkeydown=event=>{if(event.key==="Enter"&&!event.shiftKey){event.preventDefault();$("#aiForm").requestSubmit();}};
$$(".ai-suggestions button").forEach(button=>button.onclick=()=>runAssistant(button.textContent));
$("#newTodayTask").onclick=()=>openTaskDialog({today:true});$("#closeTaskDialog").onclick=$("#cancelTaskDialog").onclick=()=>$("#taskDialog").close();$("#taskForm").onsubmit=async event=>{event.preventDefault();const result=$("#taskResult"),button=$("#createTaskButton");try{if(!event.currentTarget.reportValidity())return;button.disabled=true;await createTask({title:$("#taskTitle").value,notes:$("#taskNotes").value,canvasId:$("#taskCanvas").value,status:$("#taskStatus").value,scheduledOn:$("#taskScheduledOn").value,dueOn:$("#taskDueOn").value,priority:$("#taskPriority").value});$("#taskDialog").close();}catch(error){result.className="settings-test error";result.textContent=error.message;}finally{button.disabled=false;}};$("#todayQuickAdd").onsubmit=async event=>{event.preventDefault();const input=$("#todayTaskTitle"),title=input.value.trim();if(!title)return;const button=$("button",event.currentTarget);button.disabled=true;try{await createTask({title,status:"scheduled",scheduledOn:localDateISO(),canvasId:currentCanvasId});input.value="";renderToday();}catch(error){toast(error.message);}finally{button.disabled=false;}};
$("#journalPrev").onclick=()=>shiftJournalDate(-1);
$("#journalNext").onclick=()=>shiftJournalDate(1);
$("#journalToday").onclick=()=>{flushJournalSave();journalViewDate=localDateISO();journalLoadedDate=null;renderJournalPanel();};
$("#journalBody").addEventListener("input",()=>{clearTimeout(journalSaveTimer);$("#journalStatus").textContent="Saving…";const date=journalViewDate;journalSaveTimer=setTimeout(()=>{journalSaveTimer=null;saveJournalBody(date);},600);});
$("#journalPlace").onclick=placeJournalOnCanvas;
$("#closeAINote").onclick=$("#cancelAINote").onclick=()=>$("#aiNoteDialog").close();
$("#aiNoteForm").onsubmit=event=>{event.preventDefault();const prompt=$("#aiNotePrompt").value.trim();if(prompt)createAINote(prompt);};
$("#closeAISettings").onclick=$("#cancelAISettings").onclick=()=>$("#aiSettingsDialog").close();
$("#toggleAIKey").onclick=()=>{const input=$("#aiAPIKey"),show=input.type==="password";input.type=show?"text":"password";$("#toggleAIKey").textContent=show?"Hide":"Show";};
$("#aiSettingsForm").onsubmit=event=>{event.preventDefault();if(!event.currentTarget.reportValidity())return;try{const settings=settingsFromForm();if(!settings.model||!settings.apiKey)throw new Error("Model and API key are required");persistAISettings(settings);$("#aiSettingsDialog").close();toast(`Connected to ${settings.model}`);}catch(error){setSettingsResult(error.message,"error");}};
$("#testAIProvider").onclick=async()=>{const form=$("#aiSettingsForm");if(!form.reportValidity())return;const button=$("#testAIProvider");try{const settings=settingsFromForm();button.disabled=true;setSettingsResult("Testing direct browser connection…");setSettingsResult(await testAIProvider(settings),"success");}catch(error){setSettingsResult(error.message,"error");}finally{button.disabled=false;}};
$("#clearAIProvider").onclick=()=>{persistAISettings({...aiSettings,apiKey:"",rememberKey:false});localStorage.removeItem(AI_SECRET_KEY);sessionStorage.removeItem(AI_SECRET_KEY);$("#aiSettingsDialog").close();toast("Using local canvas tools");};
$("#canvasTitle").oninput=()=>{saveCurrentCanvasState();scheduleSave();renderWorkspaceNavigation();};$("#canvasTitle").onblur=()=>{$("#canvasTitle").value=canvasRecord().title;};
initCanvasIconPicker();
$("#resetDemo").onclick=loadGraphStarter;
$("#minimap").onclick=fitView;

window.addEventListener("keydown",event=>{
  if (["INPUT","TEXTAREA","SELECT"].includes(event.target.tagName)) return;
  if(event.key==="Escape"){if(!document.querySelector("dialog[open]")&&!document.body.classList.contains("connection-dragging")&&$("#addMenuToggle")?.getAttribute("aria-expanded")!=="true"&&selected){event.preventDefault();selected=null;connectSource=null;connectSourceSide=null;shell.classList.remove("inspector-open");render();}return;}
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
// Async vault writes cannot be awaited from beforeunload. Durability is
// provided by the serialized queue and resolved save state; users should keep
// the page open until a pending save completes.
vaultReady=bootCanvasApp();
window.orbitVaultReady=vaultReady;
await vaultReady;
window.orbitCanvas={getDocument:()=>clone(documentData),getWorkspace:()=>clone(workspace),getCurrentCanvas:()=>({id:currentCanvasId,title:canvasRecord().title,trail:canvasTrail().map(record=>({id:record.id,title:record.title}))}),getSummary:canvasSummary,validateOperations:validateCanvasOperations,applyOperations:applyCanvasOperations,runAICard,createSubcanvas,createTask,loadGraphStarter,rebuildIndex:rebuildLifeIndex,setView:setAppView,switchCanvas,exportWorkspace};
