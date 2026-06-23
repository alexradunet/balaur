// graph-canvas.js — boots the interactive force-graph inside the graph card.
//
// Loaded once in the page shell (deferred), like basm.js. The card body is
// morphed into the panel via Datastar SSE, and <script> tags in morphed HTML do
// NOT execute — so instead of a per-card script we observe the DOM for a
// #graphbox and initialize it when one appears. The heavy vendored lib
// (force-graph.min.js) is fetched lazily the first time a graph is shown, so
// pages without a graph never pay for it.
//
// Node click → dispatches a `graphopen` window event carrying the node id; a
// hidden element rendered in the card listens via
//   data-on:graphopen__window="@get('/ui/show/note?id=' + evt.detail.id)"
// and lets Datastar morph the panel to that node — reusing the exact open path
// the static SVG fallback uses. (Verify this bridge in /verify against the
// embedded Datastar version; fall back to mirroring the SVG anchor if needed.)
(function () {
  var LIB = "/static/vendor/force-graph.min.js";
  var libLoading = null;

  function ensureLib() {
    if (window.ForceGraph) return Promise.resolve();
    if (libLoading) return libLoading;
    libLoading = new Promise(function (resolve, reject) {
      var s = document.createElement("script");
      s.src = LIB;
      s.onload = resolve;
      s.onerror = reject;
      document.head.appendChild(s);
    });
    return libLoading;
  }

  function fetchGraph(id, depth) {
    return fetch("/ui/graph.json?id=" + encodeURIComponent(id) + "&depth=" + depth).then(function (r) {
      return r.json();
    });
  }

  // force-graph swaps string endpoints for node objects after layout, so a link
  // key must read either shape.
  function linkKey(l) {
    var s = (l.source && l.source.id) || l.source;
    var t = (l.target && l.target.id) || l.target;
    return s + "->" + t;
  }

  function hideFallback(box) {
    var card = box.closest(".ucard-graph") || document;
    var fb = card.querySelector(".graph-fallback");
    if (fb) fb.style.display = "none";
  }

  function init(box) {
    if (box.dataset.graphReady) return;
    box.dataset.graphReady = "1";
    ensureLib()
      .then(function () {
        box.style.display = ""; // reveal before init so force-graph can size it
        var G = window
          .ForceGraph()(box)
          .nodeLabel("title")
          .nodeAutoColorBy("type")
          .onNodeClick(function (n) {
            window.dispatchEvent(new CustomEvent("graphopen", { detail: { id: n.id } }));
          })
          .onNodeRightClick(function (n) {
            // Grow the graph: merge that node's 1-hop neighborhood in place.
            fetchGraph(n.id, 1).then(function (d) {
              var cur = G.graphData();
              var ids = new Set(cur.nodes.map(function (x) { return x.id; }));
              var keys = new Set(cur.links.map(linkKey));
              d.nodes.forEach(function (x) {
                if (!ids.has(x.id)) { ids.add(x.id); cur.nodes.push(x); }
              });
              d.links.forEach(function (x) {
                if (!keys.has(linkKey(x))) { keys.add(linkKey(x)); cur.links.push(x); }
              });
              G.graphData(cur);
            });
          });
        fetchGraph(box.dataset.focus, 2)
          .then(function (d) {
            if (d && d.nodes && d.nodes.length) {
              G.graphData(d);
              hideFallback(box); // live canvas has data → drop the SVG fallback
            } else {
              box.style.display = "none"; // nothing to show → keep the SVG
            }
          })
          .catch(function () { box.style.display = "none"; });
      })
      .catch(function () { box.dataset.graphReady = ""; });
  }

  function scan(root) {
    var box = root && root.querySelector ? root.querySelector("#graphbox") : null;
    if (!box) box = document.getElementById("graphbox");
    if (box) init(box);
  }

  if (document.readyState !== "loading") scan(document);
  else document.addEventListener("DOMContentLoaded", function () { scan(document); });

  new MutationObserver(function (muts) {
    for (var i = 0; i < muts.length; i++) {
      for (var j = 0; j < muts[i].addedNodes.length; j++) {
        var n = muts[i].addedNodes[j];
        if (n.nodeType !== 1) continue;
        if (n.id === "graphbox") init(n);
        else scan(n);
      }
    }
  }).observe(document.body, { childList: true, subtree: true });
})();
