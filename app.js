/* Balaur is dependency-free. Every canvas document follows JSON Canvas 1.0; hierarchy and cameras live in a local workspace sidecar. */
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

const JD_LIFE_STARTER = [
  ["10-19","Life admin",[
    ["11","Identity & documents",[["11.01","Key documents","Keep the essentials findable and current.\n\n- [ ] Scan passport and driving licence\n- [ ] Record renewal dates"],["11.02","Insurance & renewals","One place for policies, memberships, and annual renewals.\n\n- [ ] Review coverage\n- [ ] Cancel anything unused"]]],
    ["12","Planning & reviews",[["12.01","Annual direction","A practical plan for work, health, money, and relationships at age 30.\n\n- [ ] Choose three outcomes\n- [ ] Define what to stop doing"],["12.02","Weekly review","Clear loose ends and choose the next important actions.\n\n- [ ] Empty inboxes\n- [ ] Review calendar\n- [ ] Plan the week"]]]
  ]],
  ["20-29","Health & fitness",[
    ["21","Training",[["21.01","Strength routine","Three simple full-body sessions each week.\n\n- [ ] Monday\n- [ ] Wednesday\n- [ ] Friday"],["21.02","Comfortable 10K","Build an aerobic base without turning every run into a test.\n\n- [ ] Two easy runs\n- [ ] One longer run"]]],
    ["22","Care & recovery",[["22.01","Preventive appointments","A personal reminder list, not medical advice.\n\n- [ ] Book routine checkup\n- [ ] Book dental cleaning"],["22.02","Sleep system","Protect a consistent wind-down and wake time.\n\n- [ ] Screens away by 22:30\n- [ ] Prepare tomorrow before bed"]]]
  ]],
  ["30-39","Career",[
    ["31","Current role",[["31.01","Quarterly outcomes","Make the valuable work visible and finish fewer things better.\n\n- [ ] Confirm priorities with manager\n- [ ] Ship the main project"],["31.02","Wins & evidence log","Capture outcomes, feedback, and measurable impact for future reviews."]]],
    ["32","Development",[["32.01","Skill roadmap","Build deeper technical judgment and clearer leadership communication.\n\n- [ ] Pick one course\n- [ ] Practice through a real project"],["32.02","Professional network","Maintain a small, genuine network.\n\n- [ ] Reconnect with two peers\n- [ ] Attend one local event"]]]
  ]],
  ["40-49","Money",[
    ["41","Cash flow",[["41.01","Monthly budget","Give fixed costs, everyday spending, and fun their own limits.\n\n- [ ] Reconcile accounts\n- [ ] Set next month's targets"],["41.02","Subscriptions","Review recurring costs before they become invisible.\n\n- [ ] Audit quarterly"]]],
    ["42","Safety & investing",[["42.01","Emergency fund","Build a calm cash buffer for unexpected changes.\n\nProgress: 55%"],["42.02","Long-term investing","Keep a simple record of retirement contributions and long-term allocation.\n\n- [ ] Review annually"]]]
  ]],
  ["50-59","Home & systems",[
    ["51","Apartment",[["51.01","Maintenance","Small recurring jobs that keep the home pleasant.\n\n- [ ] Replace filters\n- [ ] Test smoke alarm\n- [ ] Deep-clean kitchen"],["51.02","Household inventory","Record important purchases, warranties, and replacement dates."]]],
    ["52","Digital & mobility",[["52.01","Digital security","Keep accounts recoverable and devices protected.\n\n- [ ] Review password manager\n- [ ] Verify backups"],["52.02","Getting around","Track bicycle, public-transport, or car maintenance in one place."]]]
  ]],
  ["60-69","People",[
    ["61","Family",[["61.01","Family rhythm","Make regular calls and visits intentional rather than accidental.\n\n- [ ] Plan next visit"],["61.02","Dates & gifts","Birthdays, celebrations, and gift ideas without last-minute stress."]]],
    ["62","Friends & community",[["62.01","Friend circles","Keep a lightweight list of people to invite, call, or check in with."],["62.02","Community","Find a recurring place to contribute and meet people locally.\n\n- [ ] Try one volunteer event"]]]
  ]],
  ["70-79","Learning & fun",[
    ["71","Learning",[["71.01","Reading queue","Books worth reading next, with a sentence about why each matters."],["71.02","Course roadmap","Finish one structured course before collecting another.\n\n- [ ] Schedule two weekly sessions"]]],
    ["72","Hobbies",[["72.01","Guitar practice","A small repertoire and a sustainable practice rhythm.\n\n- [ ] Practice 20 minutes twice weekly"],["72.02","Outdoor weekends","Hikes, rides, and screen-light weekends to plan with friends."]]]
  ]],
  ["80-89","Travel",[
    ["81","Upcoming",[["81.01","Autumn city break","Choose dates, set a budget, and leave space for unplanned wandering.\n\n- [ ] Book transport\n- [ ] Reserve accommodation"],["81.02","Travel checklist","Reusable packing, document, and home-shutdown checklist."]]],
    ["82","Someday",[["82.01","Japan","A long-range trip idea: seasons, regions, rough budget, and experiences."],["82.02","Long weekends","Nearby places suitable for a low-friction three-day break."]]]
  ]],
  ["90-99","Archive",[
    ["91","Completed & learned",[["91.01","Completed projects","Move finished commitments here with links back to their outcomes."],["91.02","Lessons learned","Short reflections worth carrying into the next season of life."]]]
  ]]
];
const JD_LIFE_STARTER_TASKS = [
  ["12","Complete the weekly review","Clear inboxes, review the calendar, and choose three outcomes."],
  ["21","Schedule three training sessions","Place realistic strength sessions on this week’s calendar."],
  ["22","Book the routine checkup","Choose a provider and reserve a suitable appointment."],
  ["31","Confirm quarterly priorities","Align the next outcomes with the manager before doing more work."],
  ["41","Reconcile this month’s accounts","Compare transactions with the monthly budget."],
  ["51","Replace apartment filters","Check the maintenance list and order replacements if needed."],
  ["61","Plan the next family visit","Offer two dates and agree on the next visit."],
  ["71","Choose the next book","Pick one book from the queue before adding another."],
  ["81","Choose dates for the autumn trip","Check the calendar and agree on a realistic budget window."]
];

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
let activeAppView = "canvas";
let spaceDown = false;
let saveTimer;
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
  return {version:1,rootId:ROOT_CANVAS_ID,activeId:ROOT_CANVAS_ID,johnnyDecimal:{enabled:false,entries:{}},canvases:{[ROOT_CANVAS_ID]:{id:ROOT_CANVAS_ID,title:localStorage.getItem("orbit-title")||"Life OS — Summer",parentId:null,portalNodeId:null,path:null,document,camera:{x:80,y:55,zoom:.78}}}};
}
function normalizeWorkspace(parsed){
  parsed.johnnyDecimal ||= {enabled:false,entries:{}};parsed.johnnyDecimal.entries ||= {};
  for(const record of Object.values(parsed.canvases)){if(record.id===parsed.rootId){record.path=null;continue;}if(!record.path){const parent=parsed.canvases[record.parentId],portal=parent?.document.nodes?.find(node=>node.id===record.portalNodeId);record.path=portal?.file||`canvases/${record.id}.canvas`;}}
  return parsed;
}
function loadWorkspace(){
  try{
    const parsed=JSON.parse(localStorage.getItem(WORKSPACE_KEY)||"null"),canvases=parsed?.canvases;
    if(parsed?.version===1&&canvases&&typeof canvases==="object"&&Object.keys(canvases).length&&Object.values(canvases).every(record=>record&&typeof record.id==="string"&&typeof record.title==="string"&&isCanvas(record.document))){
      parsed.rootId=canvases[parsed.rootId]?parsed.rootId:Object.keys(canvases)[0];parsed.activeId=canvases[parsed.activeId]?parsed.activeId:parsed.rootId;return normalizeWorkspace(parsed);
    }
  }catch(_){}
  return localStorage.getItem("orbit-canvas-v1")?freshWorkspace():createJohnnyDecimalStarterWorkspace();
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
  const record=workspace.canvases[currentCanvasId];if(!record)return;record.document=documentData;record.camera={...camera};const value=$("#canvasTitle")?.value.trim();if(value){if(record.jdCode){const formatted=formatJDCode(record.jdCode),title=(value.startsWith(formatted)?value.slice(formatted.length).replace(/^\s*(?:—|-)\s*/,""):value)||record.jdTitle||"Untitled";record.jdTitle=title;record.title=jdDisplayTitle(record.jdCode,title);const entry=jdEntries()[record.jdCode];if(entry)entry.title=title;}else record.title=value;}
}
function persistWorkspace(){
  saveCurrentCanvasState();workspace.activeId=currentCanvasId;localStorage.setItem(WORKSPACE_KEY,JSON.stringify(workspace));localStorage.setItem("orbit-canvas-v1",JSON.stringify(workspace.canvases[workspace.rootId].document));localStorage.setItem("orbit-title",workspace.canvases[workspace.rootId].title);try{window.orbitLifeStore?.syncCanvasRecord(workspace.canvases[currentCanvasId]);}catch(error){console.warn("Could not update the life database index",error);}
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
  if(node?.type!=="file")return null;const record=Object.values(workspace.canvases).find(item=>item.path===node.file);return record?.id||null;
}
function canvasRecord(id=currentCanvasId){return workspace.canvases[id];}
function canvasDepth(id){
  let depth=0,seen=new Set();while(workspace.canvases[id]?.parentId&&!seen.has(id)){seen.add(id);id=workspace.canvases[id].parentId;depth++;}return depth;
}
function canvasTrail(id=currentCanvasId){
  const trail=[],seen=new Set();while(workspace.canvases[id]&&!seen.has(id)){seen.add(id);trail.unshift(workspace.canvases[id]);id=workspace.canvases[id].parentId;}return trail;
}
function canonicalJDCode(value=""){return String(value).trim().replace(/[–—]/g,"-");}
function formatJDCode(code=""){return canonicalJDCode(code).replace(/^(\d{2})-(\d{2})$/,"$1–$2");}
function jdDisplayTitle(code,title){return `${formatJDCode(code)} — ${String(title).trim()}`;}
function slug(value){return String(value).toLowerCase().normalize("NFKD").replace(/[\u0300-\u036f]/g,"").replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,"").slice(0,54)||"canvas";}
function jdEntries(){return workspace.johnnyDecimal?.entries||{};}
function jdCodeFromNode(node){return node?.type==="text"?canonicalJDCode(node.text.match(/<!--\s*orbit:jd\s+([^\s]+)\s*-->/i)?.[1]||""):"";}
function jdEntryForCanvas(canvasId){return Object.values(jdEntries()).find(entry=>entry.canvasId===canvasId)||null;}
function jdContainerKind(canvasId){if(canvasId===workspace.rootId)return "root";return jdEntryForCanvas(canvasId)?.kind||null;}
function jdChildKind(canvasId){return {root:"area",area:"category",category:"item"}[jdContainerKind(canvasId)]||null;}
function jdSortValue(code=""){const value=canonicalJDCode(code);if(/^\d{2}-\d{2}$/.test(value))return Number(value.slice(0,2))*1000;if(/^\d{2}$/.test(value))return Number(value)*1000+1;if(/^\d{2}\.\d{2}$/.test(value))return Number(value.slice(0,2))*1000+Number(value.slice(3))+2;return 999999;}
function validateJDCode(code,parentCanvasId){
  code=canonicalJDCode(code);const kind=jdChildKind(parentCanvasId),parent=jdEntryForCanvas(parentCanvasId);
  if(!kind)throw new Error("Choose the root index, an area, or a category as the parent.");
  if(jdEntries()[code])throw new Error(`${formatJDCode(code)} is already in use.`);
  if(kind==="area"){const match=code.match(/^(\d{2})-(\d{2})$/),start=match&&Number(match[1]),end=match&&Number(match[2]);if(!match||start%10||end!==start+9)throw new Error("Area IDs must be ranges such as 10-19.");}
  if(kind==="category"){const number=Number(code),range=canonicalJDCode(parent.code).split("-").map(Number);if(!/^\d{2}$/.test(code)||number<range[0]||number>range[1])throw new Error(`Choose a category from ${formatJDCode(parent.code)}.`);}
  if(kind==="item"&&!new RegExp(`^${parent.code.replace(/[.*+?^${}()|[\]\\]/g,"\\$&")}\\.(?:0[1-9]|[1-9]\\d)$`).test(code))throw new Error(`Item IDs must run from ${parent.code}.01 to ${parent.code}.99.`);
  return {code,kind};
}
function suggestJDCode(parentCanvasId){
  const kind=jdChildKind(parentCanvasId),used=new Set(Object.keys(jdEntries()).map(canonicalJDCode));
  if(kind==="area"){for(const start of [10,20,30,40,50,60,70,80,90,0]){const code=`${String(start).padStart(2,"0")}-${String(start+9).padStart(2,"0")}`;if(!used.has(code))return code;}}
  const parent=jdEntryForCanvas(parentCanvasId);if(kind==="category"){const [start,end]=canonicalJDCode(parent.code).split("-").map(Number);for(let number=start;number<=end;number++){const code=String(number).padStart(2,"0");if(!used.has(code))return code;}}
  if(kind==="item")for(let number=1;number<=99;number++){const code=`${parent.code}.${String(number).padStart(2,"0")}`;if(!used.has(code))return code;}
  return "";
}
function starterId(prefix,code){return `${prefix}-${canonicalJDCode(code).replace(/[^0-9]+/g,"-").replace(/-$/g,"")}`;}
function createJohnnyDecimalStarterWorkspace(){
  const rootDocument={nodes:[{id:"jd-starter-guide",type:"text",x:0,y:-210,width:1180,height:150,color:"3",text:"# Alex’s life index — age 30\nA fictional 30-year-old man’s Johnny Decimal system covering administration, health, career, money, home, people, interests, travel, and archives. Replace anything that does not fit your life."}],edges:[]},result=freshWorkspace(rootDocument),root=result.canvases[result.rootId];root.title="Life Index — Alex, age 30";root.document=rootDocument;root.camera=null;result.johnnyDecimal={enabled:true,entries:{}};
  JD_LIFE_STARTER.forEach(([areaCode,areaTitle,categories],areaIndex)=>{
    const areaCanvasId=starterId("jd-canvas",areaCode),areaNodeId=starterId("jd-portal",areaCode),areaPath=`canvases/${slug(`${areaCode}-${areaTitle}`)}.canvas`,areaDocument={nodes:[],edges:[]};rootDocument.nodes.push({id:areaNodeId,type:"file",x:(areaIndex%3)*410,y:Math.floor(areaIndex/3)*290,width:360,height:240,color:"5",file:areaPath});result.canvases[areaCanvasId]={id:areaCanvasId,title:jdDisplayTitle(areaCode,areaTitle),parentId:result.rootId,portalNodeId:areaNodeId,path:areaPath,document:areaDocument,camera:null,jdCode:areaCode,jdTitle:areaTitle,jdKind:"area"};result.johnnyDecimal.entries[areaCode]={code:areaCode,title:areaTitle,kind:"area",parentCanvasId:result.rootId,nodeId:areaNodeId,canvasId:areaCanvasId,itemFormat:"canvas"};
    categories.forEach(([categoryCode,categoryTitle,items],categoryIndex)=>{
      const categoryCanvasId=starterId("jd-canvas",categoryCode),categoryNodeId=starterId("jd-portal",categoryCode),categoryPath=`canvases/${slug(`${categoryCode}-${categoryTitle}`)}.canvas`,categoryDocument={nodes:[],edges:[]};areaDocument.nodes.push({id:categoryNodeId,type:"file",x:categoryIndex*410,y:0,width:360,height:240,color:"5",file:categoryPath});result.canvases[categoryCanvasId]={id:categoryCanvasId,title:jdDisplayTitle(categoryCode,categoryTitle),parentId:areaCanvasId,portalNodeId:categoryNodeId,path:categoryPath,document:categoryDocument,camera:null,jdCode:categoryCode,jdTitle:categoryTitle,jdKind:"category"};result.johnnyDecimal.entries[categoryCode]={code:categoryCode,title:categoryTitle,kind:"category",parentCanvasId:areaCanvasId,nodeId:categoryNodeId,canvasId:categoryCanvasId,itemFormat:"canvas"};
      items.forEach(([itemCode,itemTitle,itemBody],itemIndex)=>{const itemNodeId=starterId("jd-item",itemCode);categoryDocument.nodes.push({id:itemNodeId,type:"text",x:itemIndex*350,y:0,width:310,height:190,color:"3",text:`<!-- orbit:jd ${itemCode} -->\n# ${jdDisplayTitle(itemCode,itemTitle)}\n${itemBody}`});result.johnnyDecimal.entries[itemCode]={code:itemCode,title:itemTitle,kind:"item",parentCanvasId:categoryCanvasId,nodeId:itemNodeId,canvasId:null,itemFormat:"note"};});
    });
  });
  JD_LIFE_STARTER_TASKS.forEach(([categoryCode,title,notes])=>{const entry=result.johnnyDecimal.entries[categoryCode],record=result.canvases[entry.canvasId],taskId=`starter-task-${categoryCode}`,nodeId=`starter-task-node-${categoryCode}`;record.document.nodes.push({id:nodeId,type:"text",x:0,y:240,width:310,height:180,color:"5",text:buildTaskText(taskId,title,notes)});});
  return normalizeWorkspace(result);
}
function resetLifeDatabase(){Promise.resolve(window.orbitLifeReady).then(store=>{if(!store)return;store.importSnapshot({schemaVersion:1});store.syncWorkspaceIndex(workspace);reconcileTaskMarkers(store);renderToday();});}
function loadJohnnyDecimalStarter(){
  if(!confirm("Replace your current local space with the fictional age-30 Johnny Decimal starter? Export your space first if you want a backup."))return;workspace=createJohnnyDecimalStarterWorkspace();currentCanvasId=workspace.rootId;documentData=workspace.canvases[currentCanvasId].document;camera={x:80,y:55,zoom:.78};selected=null;connectSource=null;connectSourceSide=null;$("#johnnyDecimalDialog")?.close();$("#canvasTitle").value=canvasRecord().title;persistWorkspace();resetLifeDatabase();render();fitView();toast("Johnny Decimal starter space loaded");
}
function portalPreview(document){
  const nodes=(document.nodes||[]).slice(0,28);if(!nodes.length)return '<span class="portal-empty">Empty canvas · open to begin</span>';
  const minX=Math.min(...nodes.map(node=>node.x)),minY=Math.min(...nodes.map(node=>node.y)),maxX=Math.max(...nodes.map(node=>node.x+node.width)),maxY=Math.max(...nodes.map(node=>node.y+node.height)),width=Math.max(1,maxX-minX),height=Math.max(1,maxY-minY),scale=Math.min(210/width,82/height);
  return nodes.map(node=>`<i class="${node.type==="group"?"group":""}" style="left:${(node.x-minX)*scale}px;top:${(node.y-minY)*scale}px;width:${Math.max(2,node.width*scale)}px;height:${Math.max(2,node.height*scale)}px;${node.type==="group"?"":`background:${colorValue(node.color)}`}" ></i>`).join("");
}
function orderedCanvasRecords(){
  const records=Object.values(workspace.canvases),result=[],seen=new Set(),compare=(a,b)=>(jdSortValue(a.jdCode)-jdSortValue(b.jdCode))||a.title.localeCompare(b.title),visit=record=>{if(!record||seen.has(record.id))return;seen.add(record.id);result.push(record);records.filter(item=>item.parentId===record.id).sort(compare).forEach(visit);};visit(workspace.canvases[workspace.rootId]);records.sort(compare).forEach(visit);return result;
}
function renderWorkspaceNavigation(){
  const breadcrumbs=$("#canvasBreadcrumbs"),list=$("#canvasList");
  if(breadcrumbs)breadcrumbs.innerHTML=canvasTrail().map((record,index,trail)=>`<button data-canvas-switch="${escapeHTML(record.id)}" ${index===trail.length-1?'aria-current="page"':''}>${escapeHTML(record.title)}</button>${index<trail.length-1?"<span>›</span>":""}`).join("");
  if(list)list.innerHTML=orderedCanvasRecords().map(record=>`<button class="nav-item canvas-list-item ${record.id===currentCanvasId?"active":""}" data-canvas-switch="${escapeHTML(record.id)}" style="--canvas-depth:${canvasDepth(record.id)}"><span>${record.id===workspace.rootId?"◫":record.jdCode?"#":"↳"}</span><b>${escapeHTML(record.title)}</b><em>${(record.document.nodes||[]).length}</em></button>`).join("");
  $("#johnnyDecimalState")?.classList.toggle("active",Boolean(workspace.johnnyDecimal.enabled));
  $$('[data-canvas-switch]').forEach(button=>button.onclick=()=>switchCanvas(button.dataset.canvasSwitch,{direction:"switch"}));
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
  const center=point||canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),id=uid("canvas"),nodeId=uid("node"),siblings=Object.values(workspace.canvases).filter(record=>record.parentId===currentCanvasId).length,title=`New canvas ${siblings+1}`,node={id:nodeId,type:"file",x:Math.round(center.x-180),y:Math.round(center.y-125),width:360,height:250,color:"3",file:`canvases/${id}.canvas`};
  workspace.canvases[id]={id,title,parentId:currentCanvasId,portalNodeId:nodeId,path:node.file,document:{nodes:[],edges:[]},camera:null};documentData.nodes.push(node);selected={kind:"node",id:node.id};shell.classList.add("inspector-open");scheduleSave();render();toast("Sub-canvas created · double-click or zoom into it");return node;
}
function nextNodePosition(document,width,height){
  if(document===documentData){const box=canvas.getBoundingClientRect(),center=canvasPoint(box.left+box.width/2,box.top+box.height/2);return{x:Math.round(center.x-width/2),y:Math.round(center.y-height/2)};}const nodes=document.nodes||[];if(!nodes.length)return{x:0,y:0};return{x:Math.max(...nodes.map(node=>node.x+node.width))+60,y:Math.min(...nodes.map(node=>node.y))};
}
function revealWorkspaceNode(canvasId,nodeId){
  const reveal=()=>{if(currentCanvasId!==canvasId)return;const node=documentData.nodes.find(item=>item.id===nodeId);if(!node)return;selected={kind:"node",id:nodeId};shell.classList.add("inspector-open");render();focusNode(node,1.05);};if(currentCanvasId===canvasId)reveal();else{switchCanvas(canvasId,{direction:"switch",focusNodeId:nodeId});setTimeout(reveal,320);}
}
function createJDEntry({parentCanvasId,code,title,itemFormat="canvas"}){
  const checked=validateJDCode(code,parentCanvasId);code=checked.code;title=String(title).trim();if(!title)throw new Error("Add a short title.");const parent=workspace.canvases[parentCanvasId];if(!parent)throw new Error("The parent canvas no longer exists.");workspace.johnnyDecimal.enabled=true;
  const nodeId=uid("node"),isCanvasEntry=checked.kind!=="item"||itemFormat==="canvas",size=isCanvasEntry?[360,250]:[300,160],position=nextNodePosition(parent.document,...size),displayTitle=jdDisplayTitle(code,title);let canvasId=null,node;
  if(isCanvasEntry){canvasId=uid("canvas");const path=`canvases/${slug(`${canonicalJDCode(code)}-${title}`)}.canvas`;node={id:nodeId,type:"file",...position,width:size[0],height:size[1],color:"5",file:path};workspace.canvases[canvasId]={id:canvasId,title:displayTitle,parentId:parentCanvasId,portalNodeId:nodeId,path,document:{nodes:[],edges:[]},camera:null,jdCode:code,jdTitle:title,jdKind:checked.kind};}
  else node={id:nodeId,type:"text",...position,width:size[0],height:size[1],color:"3",text:`<!-- orbit:jd ${code} -->\n# ${displayTitle}\nAdd the context, outcome, or reference for this item.`};
  parent.document.nodes ||= [];parent.document.nodes.push(node);workspace.johnnyDecimal.entries[code]={code,title,kind:checked.kind,parentCanvasId,nodeId,canvasId,itemFormat:isCanvasEntry?"canvas":"note"};scheduleSave();$("#johnnyDecimalDialog")?.close();revealWorkspaceNode(parentCanvasId,nodeId);toast(`${formatJDCode(code)} added to the index`);return node;
}
function localDateISO(date=new Date()){return `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,"0")}-${String(date.getDate()).padStart(2,"0")}`;}
function taskStore(){return window.orbitLifeStore||null;}
function reconcileTaskMarkers(store=taskStore()){
  if(!store)return;for(const record of Object.values(workspace.canvases))for(const node of record.document.nodes||[]){const id=taskIdFromNode(node);if(!id)continue;const title=taskTitleFromNode(node),existing=store.task(id);if(!existing)store.upsertTask({id,canvasId:record.id,nodeId:node.id,title,status:"inbox"});else if(existing.title!==title||existing.canvasId!==record.id||existing.nodeId!==node.id)store.upsertTask({...existing,canvasId:record.id,nodeId:node.id,title});}
}
async function createTask({title,notes="",canvasId=currentCanvasId,status="inbox",scheduledOn=null,dueOn=null,priority=null}={}){
  title=String(title||"").trim();if(!title)throw new Error("Add a task title.");const store=await window.orbitLifeReady;if(!store)throw new Error("The local life database is unavailable.");const record=workspace.canvases[canvasId];if(!record)throw new Error("Choose an existing canvas.");const id=uid("task"),nodeId=uid("node"),position=nextNodePosition(record.document,310,180),node={id:nodeId,type:"text",...position,width:310,height:180,color:"5",text:buildTaskText(id,title,notes)};record.document.nodes ||= [];record.document.nodes.push(node);store.upsertTask({id,canvasId,nodeId,title,status,scheduledOn:scheduledOn||null,dueOn:dueOn||null,priority:priority===""||priority==null?null:Number(priority)});store.syncCanvasRecord(record);scheduleSave();renderToday();if(canvasId===currentCanvasId&&activeAppView==="canvas"){selected={kind:"node",id:nodeId};shell.classList.add("inspector-open");render();}toast("Task created");return node;
}
function openTaskDialog({today=false}={}){
  const dialog=$("#taskDialog"),select=$("#taskCanvas");select.innerHTML=orderedCanvasRecords().map(record=>`<option value="${escapeHTML(record.id)}">${escapeHTML(record.title)}</option>`).join("");select.value=currentCanvasId;$("#taskTitle").value="";$("#taskNotes").value="";$("#taskStatus").value=today?"scheduled":"inbox";$("#taskScheduledOn").value=today?localDateISO():"";$("#taskDueOn").value="";$("#taskPriority").value="";$("#taskResult").textContent="";dialog.showModal();setTimeout(()=>$("#taskTitle").focus(),60);
}
function taskContext(task){return workspace.canvases[task.canvasId]?.title||"Unknown canvas";}
function taskListHTML(tasks,empty){
  if(!tasks.length)return `<div class="today-empty">${escapeHTML(empty)}</div>`;return tasks.map(task=>`<article class="today-task ${task.status==="done"?"done":""}" data-task-id="${escapeHTML(task.id)}"><button type="button" class="task-check" data-complete-task aria-label="Complete ${escapeHTML(task.title)}">${task.status==="done"?"✓":""}</button><button type="button" class="task-copy" data-open-task><b>${escapeHTML(task.title)}</b><small>${escapeHTML(taskContext(task))}</small></button><div class="task-dates">${task.scheduledOn?`<time datetime="${task.scheduledOn}">Plan ${escapeHTML(task.scheduledOn.slice(5))}</time>`:""}${task.dueOn?`<time class="due" datetime="${task.dueOn}">Due ${escapeHTML(task.dueOn.slice(5))}</time>`:""}</div></article>`).join("");
}
function bindTaskList(root){
  $$('[data-task-id]',root).forEach(row=>{const id=row.dataset.taskId,store=taskStore();$("[data-complete-task]",row).onclick=()=>{store.completeTask(id);renderToday();renderNodes();toast("Task completed");};$("[data-open-task]",row).onclick=()=>{const task=store.task(id);if(!task)return;setAppView("canvas");revealWorkspaceNode(task.canvasId,task.nodeId);};});
}
function renderToday(){
  const root=$("#todayView"),store=taskStore();if(!root||!store)return;const today=localDateISO(),all=store.tasks(),active=task=>!["done","cancelled"].includes(task.status),scheduled=all.filter(task=>active(task)&&task.scheduledOn===today),overdue=all.filter(task=>active(task)&&task.dueOn&&task.dueOn<today&&task.scheduledOn!==today),queue=all.filter(task=>active(task)&&["inbox","next"].includes(task.status)&&task.scheduledOn!==today&&!overdue.includes(task)),completed=all.filter(task=>task.status==="done"&&task.completedAt&&localDateISO(new Date(task.completedAt))===today);$("#todayDate").textContent=new Intl.DateTimeFormat(undefined,{weekday:"long",month:"long",day:"numeric"}).format(new Date());$("#todayPlannedCount").textContent=scheduled.length;$("#todayDueCount").textContent=overdue.length;$("#todayDoneCount").textContent=completed.length;$("#todayScheduled").innerHTML=taskListHTML(scheduled,"Nothing scheduled yet. Choose deliberately rather than carrying everything forward.");$("#todayOverdue").innerHTML=taskListHTML(overdue,"No overdue tasks.");$("#todayQueue").innerHTML=taskListHTML(queue,"The task inbox is clear.");$("#todayCompleted").innerHTML=taskListHTML(completed,"Completed tasks will appear here.");bindTaskList(root);
}
function refreshLifeViews(){reconcileTaskMarkers();renderToday();renderNodes();}
function deleteJDEntriesForCanvas(id){for(const [code,entry] of Object.entries(jdEntries()))if(entry.canvasId===id||entry.parentCanvasId===id)delete workspace.johnnyDecimal.entries[code];}
function deleteCanvasTree(id){for(const child of Object.values(workspace.canvases).filter(record=>record.parentId===id))deleteCanvasTree(child.id);deleteJDEntriesForCanvas(id);delete workspace.canvases[id];}
function jdParentOptions(){return orderedCanvasRecords().filter(record=>record.id===workspace.rootId||["area","category"].includes(jdContainerKind(record.id)));}
function updateJDDialog(){
  const parentId=$("#jdParent").value,kind=jdChildKind(parentId),code=suggestJDCode(parentId),label={area:"Area",category:"Category",item:"Item"}[kind]||"Entry";$("#jdKindLabel").textContent=label;$("#jdCode").value=code;$("#jdCode").placeholder=kind==="area"?"10-19":kind==="category"?"11":"11.01";$("#jdTitle").placeholder=kind==="area"?"Personal":kind==="category"?"Finance":"Monthly budget";$("#jdItemFormatField").hidden=kind!=="item";$("#createJDEntry").disabled=!kind||!code;$("#jdCreateResult").textContent=kind?`Next available ${label.toLowerCase()} ID suggested automatically.`:"This canvas cannot contain another Johnny Decimal level.";
}
function openJohnnyDecimalDialog(){
  const dialog=$("#johnnyDecimalDialog"),parent=$("#jdParent"),options=jdParentOptions();parent.innerHTML=options.map(record=>`<option value="${escapeHTML(record.id)}">${escapeHTML(record.id===workspace.rootId?`Index — ${record.title}`:record.title)}</option>`).join("");const preferred=options.some(record=>record.id===currentCanvasId)?currentCanvasId:options.some(record=>record.id===canvasRecord().parentId)?canvasRecord().parentId:workspace.rootId;parent.value=preferred;$("#jdTitle").value="";$("#jdLookup").value="";$("#jdLookupList").innerHTML=Object.values(jdEntries()).sort((a,b)=>jdSortValue(a.code)-jdSortValue(b.code)).map(entry=>`<option value="${escapeHTML(formatJDCode(entry.code))}">${escapeHTML(entry.title)}</option>`).join("");$("#jdLookupResult").textContent="";updateJDDialog();dialog.showModal();setTimeout(()=>$("#jdTitle").focus(),80);
}
function goToJD(value){
  const code=canonicalJDCode(value),entry=jdEntries()[code],result=$("#jdLookupResult");if(!entry){result.className="settings-test error";result.textContent=`No entry found for ${formatJDCode(code)||"that ID"}.`;return;}result.className="settings-test success";result.textContent=`Opening ${formatJDCode(code)} — ${entry.title}`;setTimeout(()=>{$("#johnnyDecimalDialog").close();if(entry.canvasId)switchCanvas(entry.canvasId,{direction:"switch",fit:!workspace.canvases[entry.canvasId].camera});else revealWorkspaceNode(entry.parentCanvasId,entry.nodeId);},100);
}

const AI_CARD_MARKER="<!-- orbit:ai-card -->",TASK_MARKER_RE=/<!--\s*orbit:task\s+([^\s]+)\s*-->/i;
function taskIdFromNode(node){return node?.type==="text"?node.text.match(TASK_MARKER_RE)?.[1]||null:null;}
function taskTitleFromNode(node){return node?.text.match(/^#\s+(.+)$/m)?.[1]||"Untitled task";}
function buildTaskText(id,title,notes=""){return `<!-- orbit:task ${id} -->\n# ${title.trim()}${notes.trim()?`\n${notes.trim()}`:""}`;}
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
  const selectionKey = selected?.kind === "node" ? selected.id : null;
  const selectionEntering = selectionKey !== null && selectionKey !== renderedSelectionKey;
  nodeLayer.innerHTML = "";
  (documentData.nodes || []).forEach(node => {
    const element = $("#nodeTemplate").content.firstElementChild.cloneNode(true);
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
        content.innerHTML=`<div class="node-kicker">AI OPERATOR</div><h3 class="ai-card-title">${escapeHTML(config.title)}</h3><p class="ai-card-prompt">${escapeHTML(config.prompt)}</p><div class="ai-inputs">${inputs.length?inputs.map(input=>`<span class="ai-input-chip">← ${escapeHTML(nodeTitle(input))}</span>`).join(""):"<span class=\"ai-input-chip\">No inputs connected</span>"}</div><div class="ai-run-row"><span class="ai-run-status">${escapeHTML(runtime.status||"Ready")}</span><button class="ai-run-button" data-ai-run ${runtime.running?"disabled":""}>${runtime.running?"Running…":"Run now"}</button></div>`;
      } else if(taskIdFromNode(node)){const taskId=taskIdFromNode(node),task=taskStore()?.task(taskId),status=task?.status||"inbox";element.classList.add("task-card");element.classList.toggle("task-complete",status==="done");element.dataset.taskId=taskId;content.innerHTML=`<div class="node-kicker">TASK · ${escapeHTML(status.toUpperCase())}</div>${markdownToHTML(node.text)}<div class="task-node-footer"><span>${task?.scheduledOn?`Plan ${escapeHTML(task.scheduledOn)}`:task?.dueOn?`Due ${escapeHTML(task.dueOn)}`:"Not scheduled"}</span><button type="button" data-node-complete-task ${status==="done"?"disabled":""}>${status==="done"?"Completed":"Mark done"}</button></div>`;
      } else {const jdCode=jdCodeFromNode(node);content.innerHTML = `<div class="node-kicker">${jdCode?`ITEM · ${escapeHTML(formatJDCode(jdCode))}`:textMeta(node)}</div>${markdownToHTML(node.text)}`;}
    } else if (node.type === "link") {
      let linkTitle = "Saved link";
      try { linkTitle = new URL(node.url).hostname.replace(/^www\./, ""); } catch (_) {}
      content.innerHTML = `<div class="node-kicker">LINK</div><h3>${escapeHTML(linkTitle)}</h3><p>Open this resource in a new tab.</p><a class="node-link" href="${safeURL(node.url)}" target="_blank" rel="noreferrer">${escapeHTML(node.url)} ↗</a>`;
    } else if (node.type === "file") {
      const subcanvasId=subcanvasIdFromNode(node),subcanvas=subcanvasId&&workspace.canvases[subcanvasId];
      if(subcanvas){
        const children=Object.values(workspace.canvases).filter(record=>record.parentId===subcanvasId).length;element.classList.add("subcanvas-node");element.dataset.subcanvasId=subcanvasId;
        content.innerHTML=`<div class="node-kicker">${subcanvas.jdCode?`${escapeHTML(subcanvas.jdKind.toUpperCase())} · ${escapeHTML(formatJDCode(subcanvas.jdCode))}`:"SUB-CANVAS · ZOOM PORTAL"}</div><h3>${escapeHTML(subcanvas.jdTitle||subcanvas.title)}</h3><p>${subcanvas.document.nodes.length} item${subcanvas.document.nodes.length===1?"":"s"}${children?` · ${children} nested`:""}</p><div class="portal-preview">${portalPreview(subcanvas.document)}</div><div class="portal-actions"><span>Double-click or zoom to 220%</span><button type="button" data-open-subcanvas>Open ↘</button></div>`;
      } else if (/\.html?$/i.test(node.file)) {
        element.classList.add("html-widget");
        content.innerHTML = `<div class="node-kicker">LIVE HTML · SANDBOXED</div><iframe class="widget-frame" src="${safeFileURL(node.file)}" sandbox="allow-scripts" loading="lazy" referrerpolicy="no-referrer" title="${escapeHTML(node.file.split("/").pop())}"></iframe><div class="widget-shield"></div>`;
      } else content.innerHTML = `<div class="node-kicker">FILE</div><div class="file-preview">▧</div><h3>${escapeHTML(node.file.split("/").pop())}</h3><p>${escapeHTML(node.subpath || node.file)}</p>`;
    }
    element.addEventListener("pointerdown", event => nodePointerDown(event, node));
    $$("[data-connection-side]",element).forEach(handle=>{handle.addEventListener("pointerdown",event=>startConnectionDrag(event,node,handle.dataset.connectionSide));handle.addEventListener("keydown",event=>{if(event.key==="Enter"||event.key===" "){event.preventDefault();event.stopPropagation();connectSource=node.id;connectSourceSide=handle.dataset.connectionSide;setTool("connect");toast("Choose a destination node");}});});
    const aiRun=$("[data-ai-run]",element);if(aiRun){aiRun.addEventListener("pointerdown",event=>event.stopPropagation());aiRun.addEventListener("click",event=>{event.stopPropagation();runAICard(node.id,{manual:true});});}
    const taskComplete=$("[data-node-complete-task]",element);if(taskComplete){taskComplete.addEventListener("pointerdown",event=>event.stopPropagation());taskComplete.addEventListener("click",event=>{event.stopPropagation();taskStore()?.completeTask(element.dataset.taskId);renderNodes();renderToday();toast("Task completed");});}
    const portalButton=$("[data-open-subcanvas]",element);if(portalButton){portalButton.addEventListener("pointerdown",event=>event.stopPropagation());portalButton.addEventListener("click",event=>{event.stopPropagation();enterSubcanvas(element.dataset.subcanvasId);});element.addEventListener("dblclick",event=>{if(event.target.closest("button"))return;event.preventDefault();event.stopPropagation();enterSubcanvas(element.dataset.subcanvasId);});}
    element.addEventListener("click", event => {
      const anchor = event.target.closest("a");
      if (anchor) event.stopPropagation();
    });
    nodeLayer.appendChild(element);
  });
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
  if(kind==="subcanvas")return createSubcanvas(point);if(kind==="task"){openTaskDialog();return;}
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
    else if(item.type==="text"&&taskIdFromNode(item)){const task=taskStore()?.task(taskIdFromNode(item))||{status:"inbox"},statuses=["inbox","next","scheduled","waiting","done","cancelled"];contentField=`<label class="field"><span>Markdown</span><textarea data-key="text">${escapeHTML(item.text)}</textarea></label><label class="field"><span>Task status</span><select data-task-key="status">${statuses.map(status=>`<option value="${status}" ${task.status===status?"selected":""}>${status[0].toUpperCase()+status.slice(1)}</option>`).join("")}</select></label><div class="field-row"><label class="field"><span>Scheduled</span><input type="date" data-task-key="scheduledOn" value="${escapeHTML(task.scheduledOn||"")}"></label><label class="field"><span>Due</span><input type="date" data-task-key="dueOn" value="${escapeHTML(task.dueOn||"")}"></label></div><label class="field"><span>Priority</span><select data-task-key="priority"><option value="" ${task.priority==null?"selected":""}>None</option><option value="1" ${task.priority===1?"selected":""}>High</option><option value="2" ${task.priority===2?"selected":""}>Medium</option><option value="3" ${task.priority===3?"selected":""}>Low</option></select></label>`;}
    else if (item.type==="text") contentField=`<label class="field"><span>Markdown</span><textarea data-key="text">${escapeHTML(item.text)}</textarea></label>`;
    if (item.type==="link") contentField=`<label class="field"><span>URL</span><input data-key="url" value="${escapeHTML(item.url)}"></label>`;
    if (item.type==="file") {const subcanvasId=subcanvasIdFromNode(item),subcanvas=subcanvasId&&workspace.canvases[subcanvasId];contentField=subcanvas?`<label class="field"><span>${subcanvas.jdCode?`${escapeHTML(formatJDCode(subcanvas.jdCode))} title`:"Canvas title"}</span><input data-canvas-title="${escapeHTML(subcanvasId)}" value="${escapeHTML(subcanvas.jdTitle||subcanvas.title)}"></label><div class="field-hint">This portal is a standard JSON Canvas file node. Double-click it or zoom in to enter the nested canvas.</div><button type="button" class="button open-subcanvas-inspector" data-open-canvas="${escapeHTML(subcanvasId)}">Open sub-canvas ↘</button>`:`<label class="field"><span>File path</span><input data-key="file" value="${escapeHTML(item.file)}"></label><label class="field"><span>Subpath</span><input data-key="subpath" value="${escapeHTML(item.subpath||"")}"></label>`;}
    if (item.type==="group") contentField=`<label class="field"><span>Label</span><input data-key="label" value="${escapeHTML(item.label||"")}"></label><label class="field"><span>Background path</span><input data-key="background" value="${escapeHTML(item.background||"")}"></label>`;
    panel.innerHTML=`<div class="inspector-head"><h3>${taskIdFromNode(item)?"Task":item.type[0].toUpperCase()+item.type.slice(1)+" node"}</h3><button class="close-inspector">×</button></div><form class="inspector-form">${contentField}<div class="field-row"><label class="field"><span>X</span><input type="number" data-key="x" value="${item.x}"></label><label class="field"><span>Y</span><input type="number" data-key="y" value="${item.y}"></label></div><div class="field-row"><label class="field"><span>Width</span><input type="number" data-key="width" value="${item.width}"></label><label class="field"><span>Height</span><input type="number" data-key="height" value="${item.height}"></label></div><label class="field"><span>Color preset</span><div class="color-list">${colorButtons}</div></label><button type="button" class="danger-btn">Delete node</button></form>`;
  } else {
    panel.innerHTML=`<div class="inspector-head"><h3>Connection</h3><button class="close-inspector">×</button></div><form class="inspector-form"><label class="field"><span>Label</span><input data-key="label" value="${escapeHTML(item.label||"")}"></label><div class="field-row"><label class="field"><span>From side</span><select data-key="fromSide">${sideOptions(item.fromSide)}</select></label><label class="field"><span>To side</span><select data-key="toSide">${sideOptions(item.toSide)}</select></label></div><label class="field"><span>Color preset</span><div class="color-list">${colorButtons}</div></label><button type="button" class="danger-btn">Delete connection</button></form>`;
  }
  $(".close-inspector",panel).onclick=()=>{selected=null;shell.classList.remove("inspector-open");render();};
  $$("[data-key]",panel).forEach(input=>input.addEventListener("input",()=>{
    const before=aiCardSignatures(),key=input.dataset.key;
    if(key==="aiTitle"||key==="aiPrompt"){const config=parseAICard(item);item.text=buildAICardText(key==="aiTitle"?input.value:config.title,key==="aiPrompt"?input.value:config.prompt);}
    else item[key]=input.type==="number"?Math.round(Number(input.value)):input.value;
    if(key==="text"){const code=jdCodeFromNode(item),entry=jdEntries()[code],heading=item.text.match(/^#\s+(.+)$/m)?.[1];if(entry&&heading){const formatted=formatJDCode(code),title=(heading.startsWith(formatted)?heading.slice(formatted.length).replace(/^\s*(?:—|-)\s*/,""):heading).trim();if(title)entry.title=title;}const taskId=taskIdFromNode(item);if(taskId&&heading)taskStore()?.updateTask(taskId,{title:heading});}
    if (input.tagName==="SELECT" && !input.value) delete item[key];
    scheduleSave(); renderNodes(); renderEdges(); renderMinimap(); scheduleChangedAICards(before);
  }));
  $$('[data-task-key]',panel).forEach(input=>input.addEventListener("input",()=>{const id=taskIdFromNode(item),key=input.dataset.taskKey,value=key==="priority"?(input.value?Number(input.value):null):input.value||null;if(!id)return;const patch={[key]:value};if(key==="status")patch.completedAt=value==="done"?new Date().toISOString():null;taskStore()?.updateTask(id,patch);scheduleSave();renderNodes();renderToday();}));
  const canvasTitleField=$("[data-canvas-title]",panel);if(canvasTitleField)canvasTitleField.addEventListener("input",()=>{const record=workspace.canvases[canvasTitleField.dataset.canvasTitle];if(!record)return;const title=canvasTitleField.value||"Untitled";if(record.jdCode){record.jdTitle=title;record.title=jdDisplayTitle(record.jdCode,title);const entry=jdEntries()[record.jdCode];if(entry)entry.title=title;}else record.title=title;scheduleSave();renderNodes();renderWorkspaceNavigation();});
  const openCanvas=$("[data-open-canvas]",panel);if(openCanvas)openCanvas.onclick=()=>enterSubcanvas(openCanvas.dataset.openCanvas);
  $$(".color-choice",panel).forEach(button=>button.onclick=()=>{item.color=button.dataset.color;scheduleSave();render();});
  $(".danger-btn",panel).onclick=deleteSelection;
}
function sideOptions(value) { return ["","top","right","bottom","left"].map(s=>`<option value="${s}" ${value===s?"selected":""}>${s||"Auto"}</option>`).join(""); }
function deleteSelection() {
  if (!selected) return;const before=aiCardSignatures();
  if (selected.kind==="node") {
    const node=documentData.nodes.find(item=>item.id===selected.id),subcanvasId=subcanvasIdFromNode(node),taskId=taskIdFromNode(node),jdPair=Object.entries(jdEntries()).find(([,entry])=>entry.parentCanvasId===currentCanvasId&&entry.nodeId===selected.id);
    if(subcanvasId&&!confirm(`Delete “${workspace.canvases[subcanvasId].title}” and every canvas nested inside it?`))return;
    if(subcanvasId)deleteCanvasTree(subcanvasId);else if(jdPair)delete workspace.johnnyDecimal.entries[jdPair[0]];if(taskId)taskStore()?.deleteTask(taskId);
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

function downloadJSON(data,filename){const blob=new Blob([JSON.stringify(data,null,2)],{type:"application/json"}),anchor=document.createElement("a");anchor.href=URL.createObjectURL(blob);anchor.download=filename;anchor.click();setTimeout(()=>URL.revokeObjectURL(anchor.href),0);}
function exportCanvas() {
  downloadJSON(documentData,slug($("#canvasTitle").value||"life-canvas")+".canvas");toast("Current canvas exported");
}
async function exportWorkspace(){
  persistWorkspace();const store=await window.orbitLifeReady,lifeData=store?.exportSnapshot?.()||null;downloadJSON({format:"orbit-workspace",version:1,exportedAt:new Date().toISOString(),workspace,lifeData},`${slug(workspace.canvases[workspace.rootId].title)}.balaur.json`);toast(`${Object.keys(workspace.canvases).length} canvases and life data exported`);
}
function validWorkspaceBundle(data){
  const candidate=data?.format==="orbit-workspace"?data.workspace:data;if(candidate?.version!==1||!candidate.canvases||typeof candidate.canvases!=="object")return null;const records=Object.values(candidate.canvases);if(!records.length||!records.every(record=>record&&typeof record.id==="string"&&typeof record.title==="string"&&isCanvas(record.document)))return null;candidate.rootId=candidate.canvases[candidate.rootId]?candidate.rootId:records[0].id;candidate.activeId=candidate.canvases[candidate.activeId]?candidate.activeId:candidate.rootId;return normalizeWorkspace(candidate);
}
async function importCanvas(file) {
  try {const parsed=JSON.parse(await file.text()),importedWorkspace=validWorkspaceBundle(parsed);if(importedWorkspace){if(!confirm(`Import this Balaur space with ${Object.keys(importedWorkspace.canvases).length} canvases? Your current local space will be replaced.`))return;workspace=importedWorkspace;currentCanvasId=workspace.activeId;documentData=workspace.canvases[currentCanvasId].document;camera=workspace.canvases[currentCanvasId].camera||{x:80,y:55,zoom:1};selected=null;$("#canvasTitle").value=canvasRecord().title;persistWorkspace();const store=await window.orbitLifeReady;if(store){store.importSnapshot(parsed.lifeData||{schemaVersion:1});store.syncWorkspaceIndex(workspace);reconcileTaskMarkers(store);}render();fitView();toast("Whole workspace and life data imported");return;}if(!isCanvas(parsed))throw new Error("Not a valid JSON Canvas document or Orbit workspace");documentData={nodes:parsed.nodes||[],edges:parsed.edges||[]};selected=null;reconcileTaskMarkers();scheduleSave();render();fitView();toast("Canvas imported");}
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
  const before=aiCardSignatures(),{draft,themes}=validateCanvasOperations(operations);documentData=draft;themes.forEach(applyCanvasTheme);reconcileTaskMarkers();
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
    } else if(/(?:add|create).*(?:3d|webgl|html|widget)/.test(lower)) {
      const center=canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),node={id:uid("node"),type:"file",x:Math.round(center.x-240),y:Math.round(center.y-145),width:480,height:290,color:"5",file:"widgets/focus-orbit.html"};applyCanvasOperations([{type:"node.add",node}]);response="Added a sandboxed WebGL file node. It is still a standard JSON Canvas file node pointing to an HTML file.";
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
  return `You are Balaur, an assistant operating a JSON Canvas 1.0 life-management canvas. Respond with exactly one JSON object and no markdown fences: {"message":"Brief response to the user","operations":[]}.
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

window.orbitCanvas={getDocument:()=>clone(documentData),getWorkspace:()=>clone(workspace),getCurrentCanvas:()=>({id:currentCanvasId,title:canvasRecord().title,trail:canvasTrail().map(record=>({id:record.id,title:record.title}))}),getSummary:canvasSummary,validateOperations:validateCanvasOperations,applyOperations:applyCanvasOperations,runAICard,createSubcanvas,createJDEntry,createTask,goToJD,loadJohnnyDecimalStarter,setView:setAppView,switchCanvas,exportWorkspace};
applyCanvasTheme(localStorage.getItem("orbit-canvas-theme")||"default");updateProviderUI();

$$("[data-add]").forEach(button=>button.onclick=()=>button.dataset.add==="ai-note"?openAINoteDialog():addNode(button.dataset.add));$$('[data-app-view]').forEach(button=>button.onclick=()=>setAppView(button.dataset.appView));
$("#newGroup").onclick=()=>addNode("group");$("#newCanvas").onclick=()=>createSubcanvas();$("#johnnyDecimalState").onclick=openJohnnyDecimalDialog;
$$(".nav-item[data-filter]").forEach(button=>button.onclick=()=>{activeFilter=button.dataset.filter;$$(".nav-item[data-filter]").forEach(b=>b.classList.toggle("active",b===button));renderNodes();renderEdges();});
$$(".tool").forEach(button=>button.onclick=()=>{const tool=button.dataset.tool;if(tool==="note")setTool("note");else setTool(tool);});
$("#zoomIn").onclick=()=>setZoom(camera.zoom*1.2);$("#zoomOut").onclick=()=>setZoom(camera.zoom/1.2);$("#zoomLabel").onclick=()=>setZoom(1);$("#fitView").onclick=fitView;
$("#exportButton").onclick=exportCanvas;$("#exportWorkspaceButton").onclick=exportWorkspace;$("#importButton").onclick=()=>$("#fileInput").click();$("#fileInput").onchange=e=>{if(e.target.files[0])importCanvas(e.target.files[0]);e.target.value="";};
$("#sidebarToggle").onclick=()=>shell.classList.toggle("sidebar-closed");$("#sidebar").addEventListener("click",event=>{if(narrowShell.matches&&event.target.closest("button"))shell.classList.add("sidebar-closed");});
$("#assistantButton").onclick=()=>setAssistantOpen(!$("#aiPanel").classList.contains("open"));$("#closeAssistant").onclick=()=>setAssistantOpen(false);$("#openAISettings").onclick=openAISettings;
$("#aiForm").onsubmit=event=>{event.preventDefault();const input=$("#aiPrompt"),prompt=input.value;input.value="";runAssistant(prompt);};
$("#aiPrompt").onkeydown=event=>{if(event.key==="Enter"&&!event.shiftKey){event.preventDefault();$("#aiForm").requestSubmit();}};
$$(".ai-suggestions button").forEach(button=>button.onclick=()=>runAssistant(button.textContent));
$("#newTodayTask").onclick=()=>openTaskDialog({today:true});$("#closeTaskDialog").onclick=$("#cancelTaskDialog").onclick=()=>$("#taskDialog").close();$("#taskForm").onsubmit=async event=>{event.preventDefault();const result=$("#taskResult"),button=$("#createTaskButton");try{if(!event.currentTarget.reportValidity())return;button.disabled=true;await createTask({title:$("#taskTitle").value,notes:$("#taskNotes").value,canvasId:$("#taskCanvas").value,status:$("#taskStatus").value,scheduledOn:$("#taskScheduledOn").value,dueOn:$("#taskDueOn").value,priority:$("#taskPriority").value});$("#taskDialog").close();}catch(error){result.className="settings-test error";result.textContent=error.message;}finally{button.disabled=false;}};$("#todayQuickAdd").onsubmit=async event=>{event.preventDefault();const input=$("#todayTaskTitle"),title=input.value.trim();if(!title)return;const button=$("button",event.currentTarget);button.disabled=true;try{await createTask({title,status:"scheduled",scheduledOn:localDateISO(),canvasId:currentCanvasId});input.value="";renderToday();}catch(error){toast(error.message);}finally{button.disabled=false;}};
$("#closeJohnnyDecimal").onclick=$("#cancelJohnnyDecimal").onclick=()=>$("#johnnyDecimalDialog").close();$("#loadJDStarter").onclick=loadJohnnyDecimalStarter;$("#jdParent").onchange=updateJDDialog;$("#goToJD").onclick=()=>goToJD($("#jdLookup").value);$("#jdLookup").onkeydown=event=>{if(event.key==="Enter"){event.preventDefault();goToJD(event.currentTarget.value);}};$("#exportJDWorkspace").onclick=exportWorkspace;$("#johnnyDecimalForm").onsubmit=event=>{event.preventDefault();const result=$("#jdCreateResult");try{if(!event.currentTarget.reportValidity())return;createJDEntry({parentCanvasId:$("#jdParent").value,code:$("#jdCode").value,title:$("#jdTitle").value,itemFormat:$("#jdItemFormat").value});}catch(error){result.className="settings-test error";result.textContent=error.message;}};
$("#closeAINote").onclick=$("#cancelAINote").onclick=()=>$("#aiNoteDialog").close();
$("#aiNoteForm").onsubmit=event=>{event.preventDefault();const prompt=$("#aiNotePrompt").value.trim();if(prompt)createAINote(prompt);};
$("#closeAISettings").onclick=$("#cancelAISettings").onclick=()=>$("#aiSettingsDialog").close();
$("#toggleAIKey").onclick=()=>{const input=$("#aiAPIKey"),show=input.type==="password";input.type=show?"text":"password";$("#toggleAIKey").textContent=show?"Hide":"Show";};
$("#aiSettingsForm").onsubmit=event=>{event.preventDefault();if(!event.currentTarget.reportValidity())return;try{const settings=settingsFromForm();if(!settings.model||!settings.apiKey)throw new Error("Model and API key are required");persistAISettings(settings);$("#aiSettingsDialog").close();toast(`Connected to ${settings.model}`);}catch(error){setSettingsResult(error.message,"error");}};
$("#testAIProvider").onclick=async()=>{const form=$("#aiSettingsForm");if(!form.reportValidity())return;const button=$("#testAIProvider");try{const settings=settingsFromForm();button.disabled=true;setSettingsResult("Testing direct browser connection…");setSettingsResult(await testAIProvider(settings),"success");}catch(error){setSettingsResult(error.message,"error");}finally{button.disabled=false;}};
$("#clearAIProvider").onclick=()=>{persistAISettings({...aiSettings,apiKey:"",rememberKey:false});localStorage.removeItem(AI_SECRET_KEY);sessionStorage.removeItem(AI_SECRET_KEY);$("#aiSettingsDialog").close();toast("Using local canvas tools");};
$("#canvasTitle").value=canvasRecord().title;$("#canvasTitle").oninput=()=>{saveCurrentCanvasState();scheduleSave();renderWorkspaceNavigation();};$("#canvasTitle").onblur=()=>{$("#canvasTitle").value=canvasRecord().title;};
$("#resetDemo").onclick=()=>{if(confirm("Reset the whole space to the age-30 Johnny Decimal starter? Every nested canvas and local change will be replaced.")){workspace=createJohnnyDecimalStarterWorkspace();currentCanvasId=workspace.rootId;documentData=workspace.canvases[currentCanvasId].document;camera={x:80,y:55,zoom:.78};selected=null;$("#canvasTitle").value=canvasRecord().title;persistWorkspace();resetLifeDatabase();render();fitView();toast("Johnny Decimal starter restored");}};
$("#minimap").onclick=fitView;

window.addEventListener("keydown",event=>{
  if (["INPUT","TEXTAREA","SELECT"].includes(event.target.tagName)) return;
  if(event.code==="Space"){spaceDown=true;event.preventDefault();}
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==="k"){event.preventDefault();openJohnnyDecimalDialog();setTimeout(()=>$("#jdLookup").focus(),80);return;}
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
window.addEventListener("orbit:life-store-ready",refreshLifeViews);

render();
setTimeout(fitView,50);
