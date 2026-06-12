// board.js — drag-to-move and corner-resize for board slots.
// Loaded only by boards.html (deferred). Vanilla JS, no framework.
//
// Grid model: 12 columns, row unit = 10px. Layout is persisted to the server
// on pointer up via POST /ui/boards/{id}/layout.
//
// Flow → free migration (pinAllSlots):
//   A legacy "flow" board has no explicit positions stored on any card.
//   The first drag triggers pinAllSlots(), which reads each slot's current
//   rendered offsetLeft/offsetTop against the grid's column/row size and
//   assigns explicit data-x/y/w/h attributes, switching the container to
//   board-grid-free. From that point on every slot is positioned explicitly.

(function () {
  'use strict';

  var grid = document.getElementById('board-grid');
  if (!grid) return;

  var boardId = grid.dataset.boardId;
  if (!boardId) return;

  // ── Geometry helpers ─────────────────────────────────────────────────────

  function gridGeometry() {
    var style = getComputedStyle(grid);
    var gap = parseFloat(style.columnGap) || 16;
    var totalWidth = grid.getBoundingClientRect().width;
    var colW = (totalWidth - 11 * gap) / 12;
    var rowH = 10; // row unit = 10px; grid-auto-rows is 10px
    return { colW: colW, rowH: rowH, gap: gap };
  }

  function pxToCol(px, colW) {
    // Returns 0-based column index from a pixel offset.
    return Math.max(0, Math.min(11, Math.round(px / colW)));
  }

  function pxToRow(px, rowH) {
    // Returns 0-based row index from a pixel offset.
    return Math.max(0, Math.round(px / rowH));
  }

  // ── Flow → free migration ────────────────────────────────────────────────

  // pinAllSlots reads each slot's current rendered position (offsetLeft/
  // offsetTop relative to the grid) and assigns explicit data-x/y/w/h,
  // then switches the container to free-layout mode.
  //
  // Design note: getComputedStyle grid-column/row resolution is unreliable
  // for auto-flow tracks, so we use offsetLeft/offsetTop which are always
  // resolved layout values — reliable across browsers.
  function pinAllSlots() {
    if (grid.classList.contains('board-grid-free')) return; // already free

    var geo = gridGeometry();
    var gridRect = grid.getBoundingClientRect();

    var slots = grid.querySelectorAll('.board-slot');
    slots.forEach(function (slot) {
      var slotRect = slot.getBoundingClientRect();
      var x = pxToCol(slotRect.left - gridRect.left, geo.colW + geo.gap);
      var y = pxToRow(slotRect.top - gridRect.top, geo.rowH);
      var w = Math.max(1, Math.round(slot.offsetWidth / (geo.colW + geo.gap)));
      var h = Math.max(6, Math.round(slot.offsetHeight / geo.rowH));

      // Clamp x+w to 12.
      if (x + w > 12) w = 12 - x;

      slot.dataset.x = x;
      slot.dataset.y = y;
      slot.dataset.w = w;
      slot.dataset.h = h;

      applySlotStyle(slot);
    });

    grid.classList.add('board-grid-free');
  }

  // applySlotStyle writes the inline style from dataset.x/y/w/h.
  function applySlotStyle(slot) {
    var x = parseInt(slot.dataset.x, 10) || 0;
    var y = parseInt(slot.dataset.y, 10) || 0;
    var w = parseInt(slot.dataset.w, 10) || 4;
    var h = parseInt(slot.dataset.h, 10) || 16;
    slot.style.gridColumn = (x + 1) + ' / span ' + w;
    slot.style.gridRow = (y + 1) + ' / span ' + h;
  }

  // ── Persistence ──────────────────────────────────────────────────────────

  // serializeLayout collects all slots' current data-x/y/w/h into the JSON
  // array expected by POST /ui/boards/{id}/layout.
  function serializeLayout() {
    var slots = grid.querySelectorAll('.board-slot');
    var list = [];
    slots.forEach(function (slot) {
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

  function persistLayout() {
    var layout = serializeLayout();
    var body = new URLSearchParams({ layout: JSON.stringify(layout) });
    fetch('/ui/boards/' + boardId + '/layout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body,
    }).then(function (res) {
      if (!res.ok) {
        // Server state wins on error.
        location.reload();
      }
    }).catch(function () {
      location.reload();
    });
  }

  // ── Drag (move) ──────────────────────────────────────────────────────────

  grid.addEventListener('pointerdown', function (e) {
    var grip = e.target.closest('.board-slot-grip');
    if (!grip) return;

    var slot = grip.closest('.board-slot');
    if (!slot) return;

    e.preventDefault();
    grip.setPointerCapture(e.pointerId);

    // Pin all slots on the first drag of a flow-layout board.
    pinAllSlots();

    var geo = gridGeometry();
    var gridRect = grid.getBoundingClientRect();

    // Starting pointer offset within the slot (for smooth drag).
    var slotRect = slot.getBoundingClientRect();
    var offsetX = e.clientX - slotRect.left;
    var offsetY = e.clientY - slotRect.top;

    var origX = parseInt(slot.dataset.x, 10) || 0;
    var origY = parseInt(slot.dataset.y, 10) || 0;
    var w = parseInt(slot.dataset.w, 10) || 4;
    var h = parseInt(slot.dataset.h, 10) || 16;

    slot.classList.add('dragging');

    function onMove(ev) {
      var pointerX = ev.clientX - gridRect.left - offsetX;
      var pointerY = ev.clientY - gridRect.top - offsetY;

      var x = pxToCol(pointerX, geo.colW + geo.gap);
      var y = pxToRow(pointerY, geo.rowH);

      // Clamp so x+w ≤ 12.
      if (x + w > 12) x = 12 - w;
      if (y < 0) y = 0;

      slot.dataset.x = x;
      slot.dataset.y = y;
      applySlotStyle(slot);
    }

    function onUp() {
      slot.classList.remove('dragging');
      grip.removeEventListener('pointermove', onMove);
      grip.removeEventListener('pointerup', onUp);
      grip.removeEventListener('pointercancel', onUp);
      persistLayout();
    }

    grip.addEventListener('pointermove', onMove);
    grip.addEventListener('pointerup', onUp);
    grip.addEventListener('pointercancel', onUp);
  });

  // ── Resize ───────────────────────────────────────────────────────────────

  grid.addEventListener('pointerdown', function (e) {
    var handle = e.target.closest('.board-slot-resize');
    if (!handle) return;

    var slot = handle.closest('.board-slot');
    if (!slot) return;

    e.preventDefault();
    handle.setPointerCapture(e.pointerId);

    // Pin all slots on the first resize of a flow-layout board.
    pinAllSlots();

    var geo = gridGeometry();
    var gridRect = grid.getBoundingClientRect();

    var x = parseInt(slot.dataset.x, 10) || 0;
    var y = parseInt(slot.dataset.y, 10) || 0;

    // Track the slot's top-left in px for calculating new span.
    var slotRect = slot.getBoundingClientRect();
    var originX = slotRect.left - gridRect.left;
    var originY = slotRect.top - gridRect.top;

    slot.classList.add('dragging');

    function onMove(ev) {
      var pointerX = ev.clientX - gridRect.left;
      var pointerY = ev.clientY - gridRect.top;

      var w = Math.max(1, Math.round((pointerX - originX) / (geo.colW + geo.gap)));
      var h = Math.max(6, Math.round((pointerY - originY) / geo.rowH));

      // Clamp w so x+w ≤ 12.
      if (x + w > 12) w = 12 - x;

      slot.dataset.w = w;
      slot.dataset.h = h;
      applySlotStyle(slot);
    }

    function onUp() {
      slot.classList.remove('dragging');
      handle.removeEventListener('pointermove', onMove);
      handle.removeEventListener('pointerup', onUp);
      handle.removeEventListener('pointercancel', onUp);
      persistLayout();
    }

    handle.addEventListener('pointermove', onMove);
    handle.addEventListener('pointerup', onUp);
    handle.addEventListener('pointercancel', onUp);
  });

}());
