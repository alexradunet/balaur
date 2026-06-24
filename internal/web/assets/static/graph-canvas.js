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
          // Draw each node as its per-type glyph (emoji from /ui/graph.json) over a
          // faint type-colored disc, with a short title beneath. Replaces the
          // default circle so node types are distinguishable at a glance.
          .nodeCanvasObject(function (n, ctx, scale) {
            ctx.beginPath();
            ctx.arc(n.x, n.y, 6, 0, 2 * Math.PI);
            ctx.fillStyle = n.color || "#888";
            ctx.globalAlpha = 0.25;
            ctx.fill();
            ctx.globalAlpha = 1;
            var size = 12 / scale;
            ctx.font = size + "px sans-serif";
            ctx.textAlign = "center";
            ctx.textBaseline = "middle";
            ctx.fillText(n.icon || "•", n.x, n.y);
            var label = n.title || "";
            if (label.length > 18) label = label.slice(0, 17) + "…";
            var lsize = 4 / scale;
            ctx.font = lsize + "px sans-serif";
            ctx.fillStyle = "#666";
            ctx.fillText(label, n.x, n.y + size * 0.8 + lsize);
          })
          // Keep clicks hittable over the glyph (nodeCanvasObject overrides the
          // default pointer region).
          .nodePointerAreaPaint(function (n, color, ctx) {
            ctx.fillStyle = color;
            ctx.beginPath();
            ctx.arc(n.x, n.y, 9, 0, 2 * Math.PI);
            ctx.fill();
          })
          .onNodeClick(function (n) {
            window.dispatchEvent(new CustomEvent("graphopen", { detail: { id: n.id } }));
          })
          .onNodeRightClick(function (n) {
            // Grow the graph: merge that node's 1-hop neighborhood in place.
            fetchGraph(n.id, 1).then(function (d) {
              var cur = G.graphData();
              var ids = new Set(cur.nodes.map(function (x) { return x.id; }));
              var keys = new Set(cur.links.map(linkKey));
              (d.nodes || []).forEach(function (x) {
                if (!ids.has(x.id)) { ids.add(x.id); cur.nodes.push(x); }
              });
              (d.links || []).forEach(function (x) {
                if (!keys.has(linkKey(x))) { keys.add(linkKey(x)); cur.links.push(x); }
              });
              G.graphData(cur);
            });
          });
        // Keep the canvas sized to its container — the panel is narrow and
        // resizable, and force-graph otherwise captures a wrong (full-window)
        // size at init, scattering nodes off-screen. Fires immediately with the
        // current size, then on every resize.
        new ResizeObserver(function () {
          if (box.clientWidth) G.width(box.clientWidth).height(box.clientHeight);
        }).observe(box);
        // An empty/absent data-focus means "the whole graph" — fetchGraph then
        // omits the id and /ui/graph.json returns every active node.
        fetchGraph(box.dataset.focus || "", 2)
          .then(function (d) {
            if (d && d.nodes && d.nodes.length) {
              G.graphData({ nodes: d.nodes, links: d.links || [] }); // links null → force-graph throws
              hideFallback(box); // live canvas has data → drop the SVG fallback
              setTimeout(function () { G.zoomToFit(400, 24); }, 600); // frame the whole graph
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

  // Belt-and-suspenders: on reload the panel can restore a graph card via an
  // in-place morph the observer doesn't always see. init() is idempotent, so a
  // few delayed re-scans safely catch it.
  [150, 600, 1500].forEach(function (ms) { setTimeout(function () { scan(document); }, ms); });
})();
