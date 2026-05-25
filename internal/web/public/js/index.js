var id = 0;
var today = null;

function escapeHTMLAttr(s) {
    if (s === undefined || s === null) return "";
    return String(s)
        .replaceAll("&", "&amp;")
        .replaceAll('"', "&quot;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;");
}

// Find the kind of an exercise by name. Returns "cardio" or "strength".
function kindForName(name) {
    const all = window.allExercises || [];
    for (const e of all) {
        if (e.Name === name) return (e.Kind === "cardio") ? "cardio" : "strength";
    }
    return "strength";
}

// Find the strength MODE of an exercise by name. Returns "timed" for
// plank-style holds; "reps" for everything else (default).
function modeForName(name) {
    const all = window.allExercises || [];
    for (const e of all) {
        if (e.Name === name) return (e.Mode === "timed") ? "timed" : "reps";
    }
    return "reps";
}

// Find the Group attribute of an exercise by name. Falls back to "Other" so
// rows whose master exercise was deleted still group somewhere sensible.
function groupForName(name) {
    const all = window.allExercises || [];
    for (const e of all) {
        if (e.Name === name) return e.Group || "Other";
    }
    return "Other";
}

function secondsToMMSS(sec) {
    sec = parseInt(sec || 0, 10);
    if (!sec || sec <= 0) return "";
    return Math.floor(sec / 60) + ":" + String(sec % 60).padStart(2, "0");
}

// addExercise renders one row in #todayEx based on the input's kind, then
// places it so that rows of the same Group stay together (and same-Name rows
// stay adjacent within a group). A group header row is rendered above each
// section so the workout reads at a glance ("Push", "Pull", ...).
//
// Input is either an Exercise (from clicking the exercise list) or a Set
// (from setFormContent recalling saved sets). Both shapes have .Name.
function addExercise(obj) {
    id = id + 1;
    obj = obj || {};
    const name = obj.Name || "";
    const kind = obj.Kind || kindForName(name);
    const group = obj.Group || groupForName(name);
    // Mode only meaningful for strength; cardio rows ignore it.
    const mode = (kind === "strength") ? (obj.Mode || modeForName(name)) : "reps";

    let html;
    if (kind === "cardio")        html = renderCardioRow(id, name, obj);
    else if (mode === "timed")    html = renderTimedStrengthRow(id, name, obj);
    else                          html = renderStrengthRow(id, name, obj);

    const tbody = document.getElementById('todayEx');
    const tmp = document.createElement('tbody');
    tmp.innerHTML = html;
    const newRows = Array.from(tmp.children); // 2 TRs per cluster

    // Tag rows with group/name so future inserts can find siblings.
    newRows.forEach(tr => { tr.dataset.group = group; tr.dataset.name = name; });

    insertClusterGrouped(tbody, newRows, group, name);

    // Apply initial completed state from saved data.
    if (obj.Completed === 1 || obj.Completed === "1") {
        const mainTr = newRows.find(r => !r.id.startsWith('hist-') && !r.id.startsWith('adv-'));
        if (mainTr) applyCompletedState(mainTr, true, /*silent*/true);
    }

    // Fire input on the weight field once so plate hint populates without
    // the user having to type.
    const mainTr = newRows.find(r => !r.id.startsWith('hist-') && !r.id.startsWith('adv-'));
    if (mainTr) {
        const w = mainTr.querySelector('input[name="weight"]');
        if (w) w.dispatchEvent(new Event('input', { bubbles: true }));
    }
}

// insertClusterGrouped: append a row cluster (1 or 2 TRs that move together)
// into the today-workout tbody, maintaining the (group, name) ordering rule.
function insertClusterGrouped(tbody, newRows, group, name) {
    let header = tbody.querySelector(`tr.todayex-group-header[data-group="${cssEsc(group)}"]`);
    if (!header) {
        header = document.createElement('tr');
        header.className = 'todayex-group-header';
        header.dataset.group = group;
        const color = colorForGroup(group);
        header.style.setProperty('--group-color', color);
        const escGroup = escapeHTMLAttr(group);
        header.innerHTML = `<td colspan="5" class="todayex-group-title" onclick="toggleGroupCollapse('${escGroup.replaceAll("'","\\'")}')">
            <i class="bi bi-chevron-down todayex-group-chevron"></i>
            <span class="todayex-group-swatch" style="background:${color}"></span>
            <strong>${escGroup}</strong>
        </td>`;
        tbody.appendChild(header);
        // Restore collapsed state from sessionStorage.
        if (sessionStorage.getItem('todayexCollapsed:' + group) === '1') {
            header.classList.add('todayex-group-collapsed');
        }
    }

    // Always insert at the end of this group's section. Same-name rows are
    // NOT auto-stacked any more — the user's drag-drop order wins. (Earlier
    // versions snapped duplicates together, but that fought with manual
    // reorders.) "End of section" = right before the NEXT group header, or
    // at the very end of the tbody.
    let cursor = header.nextSibling;
    while (cursor && !(cursor.nodeType === 1 && cursor.classList && cursor.classList.contains('todayex-group-header'))) {
        cursor = cursor.nextSibling;
    }
    if (cursor) {
        newRows.forEach(tr => tbody.insertBefore(tr, cursor));
    } else {
        newRows.forEach(tr => tbody.appendChild(tr));
    }

    // If the group is currently collapsed, hide the new rows immediately.
    if (header.classList.contains('todayex-group-collapsed')) {
        newRows.forEach(tr => tr.classList.add('d-none'));
    }
}

// insertAfterCluster: insert newRows immediately after the LAST TR of the
// cluster that anchorMainTr belongs to. Cluster siblings:
//   strength: hist-${id} (helper, above)  +  ${id} (main)
//   cardio:   ${id} (main)                +  adv-${id} (helper, below)
function insertAfterCluster(tbody, anchorMainTr, newRows) {
    const anchorId = anchorMainTr.id;
    const adv = document.getElementById('adv-' + anchorId);
    const anchorLast = adv && adv.parentNode === tbody ? adv : anchorMainTr;
    let cursor = anchorLast.nextSibling;
    newRows.forEach(tr => {
        if (cursor) tbody.insertBefore(tr, cursor);
        else tbody.appendChild(tr);
    });
}

// CSS.escape polyfill — older browsers may not have it (Safari pre-12).
function cssEsc(s) {
    if (window.CSS && typeof CSS.escape === "function") return CSS.escape(s);
    return String(s).replace(/[^a-zA-Z0-9_-]/g, c => "\\" + c);
}

// Color-code group headers. Common group names get a fixed accent; anything
// else falls back to a hash-derived hue so multiple custom groups stay
// distinct from each other. Returns an HSL color string usable inline.
const FIXED_GROUP_COLORS = {
    'Push':        '#fd7e14',
    'Pull':        '#20c997',
    'Legs':        '#6610f2',
    'Legs & Posterior Chain': '#6610f2',
    'Shoulders':   '#ffc107',
    'Triceps':     '#d63384',
    'Biceps':      '#0dcaf0',
    'Chest':       '#dc3545',
    'Back':        '#198754',
    'Abs':         '#0d6efd',
    'Cardio (PRT)':'#0d6efd',
    'Cardio':      '#0d6efd',
    'Other':       '#6c757d'
};
function colorForGroup(name) {
    if (FIXED_GROUP_COLORS[name]) return FIXED_GROUP_COLORS[name];
    // hash → hue
    let h = 0;
    for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) & 0xffff;
    return `hsl(${h % 360}, 55%, 55%)`;
}

// toggleGroupCollapse hides/shows all data rows of a group section. The
// chevron in the header rotates and the collapsed state persists in
// sessionStorage so the user's view choice survives row re-renders.
function toggleGroupCollapse(group) {
    const tbody = document.getElementById('todayEx');
    if (!tbody) return;
    const header = tbody.querySelector(`tr.todayex-group-header[data-group="${cssEsc(group)}"]`);
    if (!header) return;
    const willCollapse = !header.classList.contains('todayex-group-collapsed');
    header.classList.toggle('todayex-group-collapsed', willCollapse);
    // Show/hide every TR that belongs to this group (data rows + helper TRs).
    tbody.querySelectorAll(`tr[data-group="${cssEsc(group)}"]:not(.todayex-group-header), tr.todayex-hist-row, tr[id^="adv-"]`)
        .forEach(tr => {
            // We need to filter helper rows that belong to a main row in THIS group.
            if (tr.classList.contains('todayex-hist-row')) {
                const parent = document.getElementById(tr.dataset.parent);
                if (!parent || parent.dataset.group !== group) return;
            } else if (tr.id && tr.id.startsWith('adv-')) {
                const parent = document.getElementById(tr.id.slice(4));
                if (!parent || parent.dataset.group !== group) return;
            } else if (tr.dataset.group !== group) {
                return;
            }
            if (willCollapse) tr.classList.add('d-none');
            else tr.classList.remove('d-none');
        });
    try {
        if (willCollapse) sessionStorage.setItem('todayexCollapsed:' + group, '1');
        else sessionStorage.removeItem('todayexCollapsed:' + group);
    } catch (e) {}
}

function renderStrengthRow(rowId, name, obj) {
    const safeName = escapeHTMLAttr(name);
    const safeWeight = escapeHTMLAttr(obj.Weight ?? 0);
    const safeReps = escapeHTMLAttr(obj.Reps ?? 0);
    const safeNote = escapeHTMLAttr(obj.Note || "");
    const chips = renderHistoryChips(rowId, name);

    return `<tr id="hist-${rowId}" class="todayex-hist-row" data-parent="${rowId}">
        <td colspan="5" class="py-1 px-2 bg-body-tertiary small">
            <div class="d-flex flex-wrap gap-1 align-items-center">${chips}</div>
        </td>
    </tr>
    <tr id="${rowId}" data-kind="strength">
    <td class="todayex-name">
        <div class="input-group">
            <span class="input-group-text todayex-drag-handle" title="Drag to reorder"><i class="bi bi-grip-vertical"></i></span>
            <input name="name" type="text" class="form-control todayex-name-input" value="${safeName}">
        </div>
    </td><td class="todayex-weight">
        <input name="weight" type="number" step="any" min="0" class="form-control todayex-num" value="${safeWeight}">
        <div class="todayex-stepper">
            <button type="button" class="btn btn-sm" tabindex="-1" onclick="bumpNum(${rowId}, 'weight', -2.5, 0)">−</button>
            <button type="button" class="btn btn-sm" tabindex="-1" onclick="bumpNum(${rowId}, 'weight', +2.5, 0)">+</button>
        </div>
        <div class="todayex-plate-hint small text-muted" data-row="${rowId}"></div>
    </td><td class="todayex-reps">
        <input name="reps" type="number" min="0" class="form-control todayex-num" value="${safeReps}">
        <div class="todayex-stepper">
            <button type="button" class="btn btn-sm" tabindex="-1" onclick="bumpNum(${rowId}, 'reps', -1, 0)">−</button>
            <button type="button" class="btn btn-sm" tabindex="-1" onclick="bumpNum(${rowId}, 'reps', +1, 0)">+</button>
        </div>
    </td><td class="todayex-note d-none d-md-table-cell">
        <input name="note" type="text" class="form-control" placeholder="(optional)" value="${safeNote}">
        <input type="hidden" name="duration_mmss" value="">
        <input type="hidden" name="distance_value" value="0">
        <input type="hidden" name="avg_hr" value="0">
        <input type="hidden" name="max_hr" value="0">
        <input type="hidden" name="calories" value="0">
        <input type="hidden" name="equipment" value="">
    </td><td class="todayex-del">
        <div class="vstack gap-1">
            <button class="btn del-set-button p-1 todayex-complete-btn" type="button" title="Mark set complete" onclick="toggleCompleteRow(${rowId})">
                <i class="bi bi-check2-square"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Repeat this set" onclick="repeatRow(${rowId})">
                <i class="bi bi-arrow-repeat"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Delete" onclick="delExercise(${rowId})">
                <i class="bi bi-x-square"></i>
            </button>
        </div>
        <input type="hidden" name="completed" value="0">
    </td></tr>`;
}

// renderTimedStrengthRow renders a strength row whose mode is "timed" — no
// weight or reps, just a duration field driven by Start/Stop buttons (plank,
// wall sit, dead hang). The duration value is stored in DURATION_SECONDS on
// the set (same column cardio uses). Time is also user-editable directly.
function renderTimedStrengthRow(rowId, name, obj) {
    const safeName = escapeHTMLAttr(name);
    const dur = parseInt(obj.DurationSeconds || 0, 10);
    const safeDur = escapeHTMLAttr(secondsToMMSS(dur));
    const safeNote = escapeHTMLAttr(obj.Note || "");
    const chips = renderTimedHistoryChips(rowId, name);

    return `<tr id="hist-${rowId}" class="todayex-hist-row" data-parent="${rowId}">
        <td colspan="5" class="py-1 px-2 bg-body-tertiary small">
            <div class="d-flex flex-wrap gap-1 align-items-center">${chips}</div>
        </td>
    </tr>
    <tr id="${rowId}" data-kind="strength" data-mode="timed">
    <td class="todayex-name">
        <div class="input-group">
            <span class="input-group-text todayex-drag-handle" title="Drag to reorder"><i class="bi bi-grip-vertical"></i></span>
            <input name="name" type="text" class="form-control todayex-name-input" value="${safeName}">
        </div>
    </td><td class="todayex-weight">
        <div class="btn-group" role="group">
            <button type="button" class="btn btn-sm btn-success" onclick="timedStrengthStart(${rowId})"><i class="bi bi-play-fill"></i> Start</button>
            <button type="button" class="btn btn-sm btn-secondary" onclick="timedStrengthStop(${rowId})"><i class="bi bi-stop-fill"></i> Stop</button>
        </div>
        <input type="hidden" name="weight" value="0">
    </td><td class="todayex-reps">
        <input name="duration_mmss" type="text" pattern="^\\d+:[0-5]\\d$" class="form-control todayex-timed-mmss" placeholder="mm:ss" value="${safeDur}">
        <input type="hidden" name="reps" value="0">
    </td><td class="todayex-note d-none d-md-table-cell">
        <input name="note" type="text" class="form-control" placeholder="(optional)" value="${safeNote}">
        <input type="hidden" name="distance_value" value="0">
        <input type="hidden" name="avg_hr" value="0">
        <input type="hidden" name="max_hr" value="0">
        <input type="hidden" name="calories" value="0">
        <input type="hidden" name="equipment" value="">
    </td><td class="todayex-del">
        <div class="vstack gap-1">
            <button class="btn del-set-button p-1 todayex-complete-btn" type="button" title="Mark set complete" onclick="toggleCompleteRow(${rowId})">
                <i class="bi bi-check2-square"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Repeat this set" onclick="repeatRow(${rowId})">
                <i class="bi bi-arrow-repeat"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Delete" onclick="delExercise(${rowId})">
                <i class="bi bi-x-square"></i>
            </button>
        </div>
        <input type="hidden" name="completed" value="0">
    </td></tr>`;
}

// Per-row timed-strength timer state, keyed by rowId.
const timedTimers = {};

function timedStrengthStart(rowId) {
    const row = document.getElementById(rowId);
    if (!row) return;
    const completed = row.dataset.completed === "1";
    if (completed) {
        if (!confirm("This set is already marked complete. Restart the timer and clear the recorded time?")) return;
        applyCompletedState(row, false);
    }
    // Cancel any prior interval for this row (silent reset before complete).
    if (timedTimers[rowId]) clearInterval(timedTimers[rowId].id);
    const mmss = row.querySelector('input[name="duration_mmss"]');
    if (mmss) mmss.value = "";
    const startedAt = Date.now();
    const tick = () => {
        const elapsed = Math.floor((Date.now() - startedAt) / 1000);
        if (mmss) mmss.value = secondsToMMSS(elapsed);
    };
    timedTimers[rowId] = { id: setInterval(tick, 250), startedAt };
    if (mmss) mmss.value = "0:00";
}

function timedStrengthStop(rowId) {
    const t = timedTimers[rowId];
    if (!t) return;
    clearInterval(t.id);
    delete timedTimers[rowId];
    const row = document.getElementById(rowId);
    if (!row) return;
    const elapsed = Math.floor((Date.now() - t.startedAt) / 1000);
    const mmss = row.querySelector('input[name="duration_mmss"]');
    if (mmss) mmss.value = secondsToMMSS(elapsed);
}

// renderTimedHistoryChips returns inline HTML chips of recent durations for a
// timed-strength exercise (e.g. plank 1:30, 1:45). Click to fill.
function renderTimedHistoryChips(rowId, name) {
    const all = window.allSets || [];
    const out = [];
    const seen = new Set();
    for (let i = all.length - 1; i >= 0 && out.length < 5; i--) {
        const s = all[i];
        if (s.Name !== name) continue;
        if (s.Completed !== undefined && s.Completed !== 1 && s.Completed !== "1") continue;
        const d = parseInt(s.DurationSeconds || 0, 10);
        if (!d) continue;
        const key = String(d);
        if (seen.has(key)) continue;
        seen.add(key);
        out.push(d);
    }
    if (!out.length) {
        return `<span class="text-muted">No prior holds for <em>${escapeHTMLAttr(name)}</em></span>`;
    }
    const chips = out.map(d => {
        const label = secondsToMMSS(d);
        return `<button type="button" class="btn btn-sm btn-outline-secondary py-0 px-2" onclick="fillTimedRow(${rowId}, ${d})">${label}</button>`;
    }).join("");
    return `<span class="text-muted me-1">Recent:</span>${chips}`;
}

function fillTimedRow(rowId, seconds) {
    const row = document.getElementById(rowId);
    if (!row) return;
    const mmss = row.querySelector('input[name="duration_mmss"]');
    if (mmss) mmss.value = secondsToMMSS(seconds);
}

// applyCompletedState (de)applies the .todayex-completed class, flips the
// hidden completed field, and switches the check icon. When silent=false and
// we just transitioned to completed for a strength row, fires the rest timer.
function applyCompletedState(mainTr, completed, silent) {
    if (!mainTr) return;
    mainTr.dataset.completed = completed ? "1" : "0";
    mainTr.classList.toggle("todayex-completed", completed);
    const hist = document.getElementById('hist-' + mainTr.id);
    if (hist) hist.classList.toggle("todayex-completed", completed);
    const hidden = mainTr.querySelector('input[name="completed"]');
    if (hidden) hidden.value = completed ? "1" : "0";
    const btn = mainTr.querySelector('.todayex-complete-btn i');
    if (btn) btn.className = completed ? "bi bi-check-square-fill" : "bi bi-check2-square";
    if (completed && !silent && mainTr.dataset.kind === "strength") {
        if (typeof window.onStrengthSetAdded === "function") {
            window.onStrengthSetAdded();
        }
    }
}

function toggleCompleteRow(rowId) {
    const row = document.getElementById(rowId);
    if (!row) return;
    // Stop any running timed-strength interval before flipping state.
    if (timedTimers[rowId]) {
        timedStrengthStop(rowId);
    }
    applyCompletedState(row, row.dataset.completed !== "1");
}

// renderHistoryChips returns the inline HTML for the most-recent N sets of an
// exercise (this user, any past date, newest first). Each chip auto-fills the
// weight/reps inputs in the parent row when tapped.
function renderHistoryChips(rowId, name) {
    const past = lastNStrengthSetsFor(name, 5);
    if (!past.length) {
        return `<span class="text-muted">No prior sets for <em>${escapeHTMLAttr(name)}</em></span>`;
    }
    const today = document.getElementById("formDate") ? document.getElementById("formDate").value : "";
    const chips = past.map(s => {
        const w = String(s.Weight ?? 0);
        const r = String(s.Reps ?? 0);
        const label = `${w}×${r}`;
        const dateLabel = (s.Date === today) ? "earlier today" : s.Date;
        return `<button type="button" class="btn btn-sm btn-outline-secondary py-0 px-2"
                    title="${dateLabel}"
                    onclick="fillRow(${rowId}, ${w}, ${r})">${label}</button>`;
    }).join("");
    return `<span class="text-muted me-1">Recent:</span>${chips}`;
}

// lastNStrengthSetsFor returns up to n recent sets (newest first, de-duplicated
// by weight×reps combo) for the given exercise name. Cardio-kind sets are
// skipped so chips stay numeric.
function lastNStrengthSetsFor(name, n) {
    const all = window.allSets || [];
    if (kindForName(name) === "cardio") return [];
    const out = [];
    const seen = new Set();
    for (let i = all.length - 1; i >= 0 && out.length < n; i--) {
        const s = all[i];
        if (s.Name !== name) continue;
        // Only suggest completed past sets — planned-but-skipped rows would
        // mislead with weights the user never actually lifted.
        if (s.Completed !== undefined && s.Completed !== 1 && s.Completed !== "1") continue;
        const key = `${s.Weight}|${s.Reps}`;
        if (seen.has(key)) continue;
        seen.add(key);
        out.push(s);
    }
    return out;
}

function fillRow(rowId, weight, reps) {
    const row = document.getElementById(rowId);
    if (!row) return;
    const w = row.querySelector('input[name="weight"]');
    const r = row.querySelector('input[name="reps"]');
    if (w) { w.value = weight; w.dispatchEvent(new Event("input", { bubbles: true })); }
    if (r) { r.value = reps; r.dispatchEvent(new Event("input", { bubbles: true })); }
}

function bumpNum(rowId, fieldName, delta, min) {
    const row = document.getElementById(rowId);
    if (!row) return;
    const el = row.querySelector('input[name="' + fieldName + '"]');
    if (!el) return;
    const cur = parseFloat(el.value || "0") || 0;
    let next = cur + delta;
    if (typeof min === "number" && next < min) next = min;
    // Round to avoid 137.49999 from float math when stepping by 2.5.
    next = Math.round(next * 100) / 100;
    el.value = next;
    el.dispatchEvent(new Event("input", { bubbles: true }));
}

// Plate-hint refresh hook. Called whenever a strength row's weight input
// changes. Looks for window.suggestPlates(weight) provided by the plate
// calculator (task 6) and renders its HTML into the row's hint slot.
// No-op until that module is loaded.
function refreshPlateHint(row) {
    const hint = row.querySelector('.todayex-plate-hint');
    if (!hint) return;
    if (typeof window.suggestPlates !== "function") { hint.innerHTML = ""; return; }
    const w = parseFloat(row.querySelector('input[name="weight"]').value || "0") || 0;
    hint.innerHTML = window.suggestPlates(w);
}

// Delegate input events on the today-form so newly-added rows pick up plate
// hint refresh without per-row wiring.
document.addEventListener("input", function(ev) {
    const t = ev.target;
    if (!t || t.name !== "weight") return;
    const row = t.closest('tr[data-kind="strength"]');
    if (row) refreshPlateHint(row);
});

function repeatRow(rowId) {
    const row = document.getElementById(rowId);
    if (!row) return;
    const name = row.querySelector('input[name="name"]').value;
    const weight = row.querySelector('input[name="weight"]').value;
    const reps = row.querySelector('input[name="reps"]').value;
    const note = row.querySelector('input[name="note"]').value;
    addExercise({ Name: name, Weight: weight, Reps: reps, Note: note, Kind: "strength" });
}

function renderCardioRow(rowId, name, obj) {
    const safeName = escapeHTMLAttr(name);
    const safeDist = escapeHTMLAttr(obj.DistanceValue ?? 0);
    const durSec = parseInt(obj.DurationSeconds || 0, 10);
    const safeDurMMSS = escapeHTMLAttr(secondsToMMSS(durSec));
    const safeNote = escapeHTMLAttr(obj.Note || "");
    const safeAvg = escapeHTMLAttr(obj.AvgHR ?? 0);
    const safeMax = escapeHTMLAttr(obj.MaxHR ?? 0);
    const safeCals = escapeHTMLAttr(obj.Calories ?? 0);
    const safeEquip = escapeHTMLAttr(obj.Equipment || "");
    const isBike = /stationary bike/i.test(name);
    const unit = window.distanceUnit || "mi";
    const showAdvanced = (safeAvg !== "0" || safeMax !== "0" || safeCals !== "0" || (isBike && safeEquip !== ""));

    const bikeEquipFragment = isBike
        ? `<div class="input-group input-group-sm" style="width:auto">
             <span class="input-group-text">Bike</span>
             <input name="equipment" type="text" class="form-control" list="bikeModelsList" autocomplete="off" placeholder="model" value="${safeEquip}" style="width:14em">
           </div>`
        : `<input type="hidden" name="equipment" value="${safeEquip}">`;

    return `<tr id="${rowId}" data-kind="cardio">
    <td class="todayex-name">
        <div class="input-group">
            <span class="input-group-text todayex-drag-handle" title="Drag to reorder"><i class="bi bi-grip-vertical"></i></span>
            <input name="name" type="text" class="form-control todayex-name-input" value="${safeName}">
        </div>
    </td><td class="todayex-weight">
        <div class="input-group">
            <input name="distance_value" type="number" step="any" min="0" class="form-control" placeholder="dist" value="${safeDist}">
            <span class="input-group-text">${unit}</span>
        </div>
        <input type="hidden" name="weight" value="0">
    </td><td class="todayex-reps">
        <input name="duration_mmss" type="text" pattern="^\\d+:[0-5]\\d$" class="form-control" placeholder="mm:ss" value="${safeDurMMSS}">
        <input type="hidden" name="reps" value="0">
    </td><td class="todayex-note d-none d-md-table-cell">
        <input name="note" type="text" class="form-control" placeholder="(optional)" value="${safeNote}">
    </td><td class="todayex-del">
        <div class="vstack gap-1 align-items-end">
            <button class="btn del-set-button p-1 todayex-complete-btn" type="button" title="Mark complete" onclick="toggleCompleteRow(${rowId})">
                <i class="bi bi-check2-square"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Advanced (HR / cal${isBike ? " / bike" : ""})" onclick="toggleCardioAdvanced(${rowId})">
                <i class="bi bi-sliders"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Delete" onclick="delExercise(${rowId})">
                <i class="bi bi-x-square"></i>
            </button>
        </div>
        <input type="hidden" name="completed" value="0">
    </td></tr>
    <tr id="adv-${rowId}" class="cardio-advanced ${showAdvanced ? "" : "d-none"}" data-parent="${rowId}">
        <td colspan="5">
            <div class="d-flex flex-wrap gap-2 small bg-body-tertiary p-2 rounded">
                <div class="input-group input-group-sm" style="width:auto">
                    <span class="input-group-text">Avg HR</span>
                    <input name="avg_hr" type="number" min="0" max="250" class="form-control" placeholder="bpm" value="${safeAvg}" style="width:5em">
                </div>
                <div class="input-group input-group-sm" style="width:auto">
                    <span class="input-group-text">Max HR</span>
                    <input name="max_hr" type="number" min="0" max="250" class="form-control" placeholder="bpm" value="${safeMax}" style="width:5em">
                </div>
                <div class="input-group input-group-sm" style="width:auto">
                    <span class="input-group-text">Cal</span>
                    <input name="calories" type="number" min="0" class="form-control" placeholder="kcal" value="${safeCals}" style="width:6em">
                </div>
                ${bikeEquipFragment}
            </div>
        </td>
    </tr>`;
}

function toggleCardioAdvanced(rowId) {
    const adv = document.getElementById('adv-' + rowId);
    if (!adv) return;
    adv.classList.toggle('d-none');
}

function setFormContent(sets, date) {
    window.sessionStorage.setItem("today", date);
    document.getElementById('todayEx').innerHTML = "";
    document.getElementById("formDate").value = date;
    document.getElementById("realDate").value = date;

    if (sets) {
        for (let i = 0; i < sets.length; i++) {
            if (sets[i].Date == date) {
                // Saved sets don't carry Kind; look it up from the exercise list.
                const s = Object.assign({}, sets[i], { Kind: kindForName(sets[i].Name) });
                addExercise(s);
            }
        }
    }
}

function setFormDate(sets) {
    today = document.getElementById("realDate").value;
    if (!today) {
        today = window.sessionStorage.getItem("today");
        if (!today) {
            today = new Date().toJSON().slice(0, 10);
        }
    }
    setFormContent(sets, today);
    initSortable();
}

// initSortable wires SortableJS to the today-workout tbody once, allowing
// the user to drag and drop strength/cardio/timed rows within their group
// (cross-group drags are rejected via onMove). Touch-friendly: a 150ms hold
// on the grip handle starts the drag so accidental list-scrolls don't fire.
let sortableInstance = null;
function initSortable() {
    if (sortableInstance) return; // single-init
    if (typeof Sortable === "undefined") return; // CDN not loaded
    const tbody = document.getElementById('todayEx');
    if (!tbody) return;
    sortableInstance = Sortable.create(tbody, {
        handle: '.todayex-drag-handle',
        // hist-${id} (above main strength row) and adv-${id} (below cardio
        // main row) are helper TRs and group headers are anchors; none
        // should be draggable themselves.
        filter: '.todayex-group-header, .todayex-hist-row, [id^="adv-"]',
        draggable: 'tr[data-kind]',
        delay: 150,
        delayOnTouchOnly: true,
        animation: 150,
        ghostClass: 'todayex-drag-ghost',
        chosenClass: 'todayex-drag-chosen',
        // Reject drops outside the source row's own group.
        onMove(evt) {
            const dragGroup = evt.dragged && evt.dragged.dataset.group;
            const related = evt.related;
            if (!related || !dragGroup) return true;
            let relGroup = related.dataset.group;
            if (!relGroup) {
                // Helper rows (hist-${id}/adv-${id}) don't carry data-group
                // themselves — resolve via their parent main row.
                if (related.classList.contains('todayex-hist-row')) {
                    const p = document.getElementById(related.dataset.parent);
                    relGroup = p && p.dataset.group;
                } else if (related.id && related.id.startsWith('adv-')) {
                    const p = document.getElementById(related.id.slice(4));
                    relGroup = p && p.dataset.group;
                }
            }
            if (relGroup && relGroup !== dragGroup) return false;
            return true;
        },
        // After a drop, re-pair helper TRs with their main row so hist-${id}
        // stays directly above its strength row and adv-${id} stays below
        // its cardio row.
        onEnd() { restoreClusterAdjacency(); }
    });
}

function restoreClusterAdjacency() {
    const tbody = document.getElementById('todayEx');
    if (!tbody) return;
    tbody.querySelectorAll('tr[data-kind]').forEach(main => {
        const hist = document.getElementById('hist-' + main.id);
        if (hist && main.previousElementSibling !== hist) {
            tbody.insertBefore(hist, main);
        }
        const adv = document.getElementById('adv-' + main.id);
        if (adv && main.nextElementSibling !== adv) {
            if (main.nextSibling) tbody.insertBefore(adv, main.nextSibling);
            else tbody.appendChild(adv);
        }
    });
}

function setWeightDate() {
    let date = document.getElementById("realDate").value;
    document.getElementById("weightDate").value = date;
}

function delExercise(exID) {
    const row = document.getElementById(exID);
    const adv = document.getElementById('adv-' + exID);
    const hist = document.getElementById('hist-' + exID);
    const group = row && row.dataset && row.dataset.group;
    if (adv) adv.remove();
    if (hist) hist.remove();
    if (row) row.remove();

    // Remove the group header if no rows for that group remain.
    if (group) {
        const tbody = document.getElementById('todayEx');
        const remaining = tbody.querySelectorAll(
            `tr[data-group="${cssEsc(group)}"]:not(.todayex-group-header)`
        );
        if (!remaining.length) {
            const header = tbody.querySelector(
                `tr.todayex-group-header[data-group="${cssEsc(group)}"]`
            );
            if (header) header.remove();
        }
    }
}

function moveDayLeftRight(where, sets) {
    const dateStr = document.getElementById("realDate").value;
    const year  = dateStr.substring(0, 4);
    const month = dateStr.substring(5, 7);
    const day   = dateStr.substring(8, 10);
    const date  = new Date(year, month - 1, day);
    date.setDate(date.getDate() + parseInt(where));
    const left = date.toLocaleDateString('en-CA');
    setFormContent(sets, left);
}

function addAllGroup(exs, gr) {
    if (!exs) return;
    for (let i = 0; i < exs.length; i++) {
        if (exs[i].Group == gr) {
            addExercise(exs[i]);
        }
    }
}

// filterAccordion narrows the home-page exercise accordion as the user
// types. Empty query restores the default collapsed state; a non-empty
// query auto-expands groups that contain at least one match and hides
// groups with no matches so the user sees a focused result list.
function filterAccordion(q) {
    q = (q || "").trim().toLowerCase();
    const acc = document.getElementById('exAccordion');
    if (!acc) return;
    const items = acc.querySelectorAll('.accordion-item');
    items.forEach(item => {
        const rows = item.querySelectorAll('.accordion-body .hstack');
        let matches = 0;
        rows.forEach(row => {
            const nameEl = row.querySelector('.exercise-button .text-truncate');
            const name = nameEl ? nameEl.textContent.trim().toLowerCase() : '';
            const match = !q || name.includes(q);
            row.classList.toggle('d-none', !match);
            if (match) matches++;
        });
        item.classList.toggle('d-none', !!q && matches === 0);
        const collapse = item.querySelector('.accordion-collapse');
        const btn = item.querySelector('.accordion-button');
        if (!collapse || !btn) return;
        if (q && matches > 0) {
            collapse.classList.add('show');
            btn.classList.remove('collapsed');
            btn.setAttribute('aria-expanded', 'true');
        } else if (!q) {
            collapse.classList.remove('show');
            btn.classList.add('collapsed');
            btn.setAttribute('aria-expanded', 'false');
        }
    });
}

// renderWeightPanel populates the last-weight number, BMI badge (if enabled
// and height is set), and goal-progress block (if a goal weight is set).
// Called once from the inline init script with the current user's weight
// history. Safe with an empty array.
function renderWeightPanel(weight) {
    const cfg = window.bmiCfg || { enabled: false, heightInches: 0, goalWeight: 0, unit: "lb" };
    const last = (weight && weight.length) ? weight[weight.length - 1] : null;
    const lastVal = last ? parseFloat(last.Weight) : 0;

    const numEl = document.getElementById('weightLastNumber');
    if (numEl) {
        numEl.textContent = lastVal ? (Math.round(lastVal * 10) / 10) : '—';
    }

    const bmiBlock = document.getElementById('bmiBlock');
    const bmiVal = document.getElementById('bmiValue');
    const bmiBadge = document.getElementById('bmiBadge');
    if (bmiBlock && bmiVal && bmiBadge) {
        if (!cfg.enabled || !cfg.heightInches || !lastVal) {
            bmiBlock.style.display = 'none';
        } else {
            bmiBlock.style.display = '';
            let bmi;
            if (cfg.unit === 'kg') {
                // metric: kg / m^2
                const meters = cfg.heightInches * 0.0254;
                bmi = lastVal / (meters * meters);
            } else {
                // imperial: lb / in^2 * 703
                bmi = (lastVal / (cfg.heightInches * cfg.heightInches)) * 703;
            }
            bmi = Math.round(bmi * 10) / 10;
            bmiVal.textContent = bmi.toFixed(1);
            const cat = bmiCategory(bmi);
            bmiBadge.textContent = cat.label;
            bmiBadge.className = 'badge bg-' + cat.color;
        }
    }

    const goalBlock = document.getElementById('goalBlock');
    const goalTarget = document.getElementById('goalTarget');
    const goalDelta = document.getElementById('goalDelta');
    const goalBar = document.getElementById('goalBar');
    if (goalBlock && goalTarget && goalDelta && goalBar) {
        if (!cfg.goalWeight || !lastVal || !weight || !weight.length) {
            goalBlock.classList.add('d-none');
        } else {
            goalBlock.classList.remove('d-none');
            goalTarget.textContent = cfg.goalWeight + ' ' + cfg.unit;
            const delta = Math.round((lastVal - cfg.goalWeight) * 10) / 10;
            if (Math.abs(delta) < 0.05) {
                goalDelta.innerHTML = '<span class="text-success">at goal</span>';
            } else if (delta > 0) {
                goalDelta.innerHTML = '<span class="text-warning">' + delta + ' ' + cfg.unit + ' to go (cut)</span>';
            } else {
                goalDelta.innerHTML = '<span class="text-warning">' + Math.abs(delta) + ' ' + cfg.unit + ' to go (gain)</span>';
            }
            // Progress: assume the starting weight is the FIRST entry. Pct
            // toward goal = (start - current) / (start - goal), clamped.
            const start = parseFloat(weight[0].Weight) || lastVal;
            const denom = (start - cfg.goalWeight);
            let pct = 0;
            if (denom !== 0) {
                pct = Math.max(0, Math.min(100, ((start - lastVal) / denom) * 100));
            }
            goalBar.style.width = pct + '%';
            goalBar.className = 'progress-bar bg-' + (pct >= 100 ? 'success' : 'primary');
        }
    }
}

function bmiCategory(bmi) {
    if (bmi < 18.5)  return { label: 'Underweight', color: 'info' };
    if (bmi < 25.0)  return { label: 'Healthy',     color: 'success' };
    if (bmi < 30.0)  return { label: 'Overweight',  color: 'warning' };
    return { label: 'Obesity', color: 'danger' };
}

// Weight card visibility (persisted in localStorage)
function hideWeightCard() {
    document.getElementById('weightCardCol').classList.add('d-none');
    document.getElementById('weightShowCol').classList.remove('d-none');
    try { localStorage.setItem('hideWeightCard', '1'); } catch (e) {}
}
function showWeightCard() {
    document.getElementById('weightCardCol').classList.remove('d-none');
    document.getElementById('weightShowCol').classList.add('d-none');
    try { localStorage.removeItem('hideWeightCard'); } catch (e) {}
}
(function() {
    try {
        if (localStorage.getItem('hideWeightCard') === '1') {
            document.addEventListener('DOMContentLoaded', hideWeightCard);
        }
    } catch (e) {}
})();

// PRT standards card visibility (persisted in localStorage)
function hidePrtStandards() {
    const card = document.getElementById('prtStandardsCard');
    const btn = document.getElementById('prtStandardsShowBtn');
    if (card) card.classList.add('d-none');
    if (btn) btn.classList.remove('d-none');
    try { localStorage.setItem('hidePrtStandards', '1'); } catch (e) {}
}
function showPrtStandards() {
    const card = document.getElementById('prtStandardsCard');
    const btn = document.getElementById('prtStandardsShowBtn');
    if (card) card.classList.remove('d-none');
    if (btn) btn.classList.add('d-none');
    try { localStorage.removeItem('hidePrtStandards'); } catch (e) {}
}
(function() {
    try {
        if (localStorage.getItem('hidePrtStandards') === '1') {
            document.addEventListener('DOMContentLoaded', hidePrtStandards);
        }
    } catch (e) {}
})();
