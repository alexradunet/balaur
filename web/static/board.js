// board.js — drag-to-move and corner-resize for board slots. Vanilla JS, no
// framework. Safe on pages without a board.
//
// Swap-safe: the pointer listeners are delegated from `document`, and the grid
// + board id are resolved per interaction. This survives Datastar/htmx swaps
// that replace #board-grid (switching boards patches #main), where a one-time
// IIFE that captured the grid at load would go dead.
//
// Grid model: 12 columns, row unit = 10px. Layout persists on pointer up via
// POST /ui/boards/{id}/layout.

(function () {
  'use strict';

  function gridGeometry(grid) {
    var style = getComputedStyle(grid);
    var gap = parseFloat(style.columnGap) || 16;
    var totalWidth = grid.getBoundingClientRect().width;
    return { colW: (totalWidth - 11 * gap) / 12, rowH: 10, gap: gap };
  }

  function pxToCol(px, colW) { return Math.max(0, Math.min(11, Math.round(px / colW))); }
  function pxToRow(px, rowH) { return Math.max(0, Math.round(px / rowH)); }

  function applySlotStyle(slot) {
    var x = parseInt(slot.dataset.x, 10) || 0;
    var y = parseInt(slot.dataset.y, 10) || 0;
    var w = parseInt(slot.dataset.w, 10) || 4;
    var h = parseInt(slot.dataset.h, 10) || 16;
    slot.style.gridColumn = (x + 1) + ' / span ' + w;
    slot.style.gridRow = (y + 1) + ' / span ' + h;
  }

  // pinAllSlots: convert a legacy flow board to explicit positions on first drag.
  function pinAllSlots(grid) {
    if (grid.classList.contains('board-grid-free')) return;
    var geo = gridGeometry(grid);
    var gridRect = grid.getBoundingClientRect();
    grid.querySelectorAll('.board-slot').forEach(function (slot) {
      var slotRect = slot.getBoundingClientRect();
      var x = pxToCol(slotRect.left - gridRect.left, geo.colW + geo.gap);
      var y = pxToRow(slotRect.top - gridRect.top, geo.rowH);
      var w = Math.max(1, Math.round(slot.offsetWidth / (geo.colW + geo.gap)));
      var h = Math.max(6, Math.round(slot.offsetHeight / geo.rowH));
      if (x + w > 12) w = 12 - x;
      slot.dataset.x = x; slot.dataset.y = y; slot.dataset.w = w; slot.dataset.h = h;
      applySlotStyle(slot);
    });
    grid.classList.add('board-grid-free');
    packAndApply(grid);
  }

  function serializeLayout(grid) {
    var list = [];
    grid.querySelectorAll('.board-slot').forEach(function (slot) {
      list.push({
        idx: parseInt(slot.dataset.idx, 10),
        x: parseInt(slot.dataset.x, 10) || 0,
        y: parseInt(slot.dataset.y, 10) || 0,
        w: parseInt(slot.dataset.w, 10) || 4,
        h: parseInt(slot.dataset.h, 10) || 16,
      });
    });
    return list;
  }

  // packLayout: compact slots upward (sort by y,x then settle each as high as
  // it fits). Pure: mutates and returns the items.
  function packLayout(items) {
    var placed = [];
    items.slice().sort(function (a, b) { return a.y !== b.y ? a.y - b.y : a.x - b.x; })
      .forEach(function (it) {
        var y = 0;
        while (placed.some(function (p) {
          return it.x < p.x + p.w && p.x < it.x + it.w && y < p.y + p.h && p.y < y + it.h;
        })) { y++; }
        it.y = y;
        placed.push(it);
      });
    return items;
  }

  function packAndApply(grid) {
    var items = [];
    grid.querySelectorAll('.board-slot').forEach(function (slot) {
      items.push({
        el: slot,
        x: parseInt(slot.dataset.x, 10) || 0,
        y: parseInt(slot.dataset.y, 10) || 0,
        w: parseInt(slot.dataset.w, 10) || 4,
        h: parseInt(slot.dataset.h, 10) || 16,
      });
    });
    packLayout(items);
    items.forEach(function (it) { it.el.dataset.y = it.y; applySlotStyle(it.el); });
  }

  function persistLayout(grid) {
    var boardId = grid.dataset.boardId;
    if (!boardId) return;
    fetch('/ui/boards/' + boardId + '/layout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ layout: JSON.stringify(serializeLayout(grid)) }),
    }).then(function (res) { if (!res.ok) resyncBoard(); })
      .catch(function () { resyncBoard(); });
  }

  // resyncBoard re-pulls the current board into #main via the active Datastar
  // tab, so a rejected layout save (e.g. a grid another tab already changed)
  // refreshes the canvas WITHOUT a full-page reload that would tear down the
  // dock and kill its live chat stream. Falls back to reload only if no tab.
  function resyncBoard() {
    var tab = document.querySelector('.board-tabs .k-tab-active');
    if (tab) tab.click();
    else location.reload();
  }

  // ── Delegated drag (move) ──────────────────────────────────────────────
  document.addEventListener('pointerdown', function (e) {
    var grip = e.target.closest('.board-slot-grip');
    if (!grip) return;
    var slot = grip.closest('.board-slot');
    var grid = grip.closest('.board-grid');
    if (!slot || !grid) return;

    e.preventDefault();
    grip.setPointerCapture(e.pointerId);
    pinAllSlots(grid);

    var geo = gridGeometry(grid);
    var gridRect = grid.getBoundingClientRect();
    var slotRect = slot.getBoundingClientRect();
    var offsetX = e.clientX - slotRect.left;
    var offsetY = e.clientY - slotRect.top;
    var w = parseInt(slot.dataset.w, 10) || 4;
    slot.classList.add('dragging');

    function onMove(ev) {
      var x = pxToCol(ev.clientX - gridRect.left - offsetX, geo.colW + geo.gap);
      var y = pxToRow(ev.clientY - gridRect.top - offsetY, geo.rowH);
      if (x + w > 12) x = 12 - w;
      if (y < 0) y = 0;
      slot.dataset.x = x; slot.dataset.y = y;
      applySlotStyle(slot);
    }
    function onUp() {
      slot.classList.remove('dragging');
      document.removeEventListener('pointermove', onMove);
      document.removeEventListener('pointerup', onUp);
      document.removeEventListener('pointercancel', onUp);
      packAndApply(grid);
      persistLayout(grid);
    }
    // Listen on document, not the grip: a mid-drag #main swap can detach the
    // grip, but document is stable, so pointerup still fires and persists.
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
    document.addEventListener('pointercancel', onUp);
  });

  // ── Delegated resize ───────────────────────────────────────────────────
  document.addEventListener('pointerdown', function (e) {
    var handle = e.target.closest('.board-slot-resize');
    if (!handle) return;
    var slot = handle.closest('.board-slot');
    var grid = handle.closest('.board-grid');
    if (!slot || !grid) return;

    e.preventDefault();
    handle.setPointerCapture(e.pointerId);
    pinAllSlots(grid);

    var geo = gridGeometry(grid);
    var gridRect = grid.getBoundingClientRect();
    var x = parseInt(slot.dataset.x, 10) || 0;
    var slotRect = slot.getBoundingClientRect();
    var originX = slotRect.left - gridRect.left;
    var originY = slotRect.top - gridRect.top;
    slot.classList.add('dragging');

    function onMove(ev) {
      var w = Math.max(1, Math.round((ev.clientX - gridRect.left - originX) / (geo.colW + geo.gap)));
      var h = Math.max(6, Math.round((ev.clientY - gridRect.top - originY) / geo.rowH));
      if (x + w > 12) w = 12 - x;
      slot.dataset.w = w; slot.dataset.h = h;
      applySlotStyle(slot);
    }
    function onUp() {
      slot.classList.remove('dragging');
      document.removeEventListener('pointermove', onMove);
      document.removeEventListener('pointerup', onUp);
      document.removeEventListener('pointercancel', onUp);
      packAndApply(grid);
      persistLayout(grid);
    }
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
    document.addEventListener('pointercancel', onUp);
  });
}());
