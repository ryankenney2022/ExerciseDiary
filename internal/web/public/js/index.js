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

function secondsToMMSS(sec) {
    sec = parseInt(sec || 0, 10);
    if (!sec || sec <= 0) return "";
    return Math.floor(sec / 60) + ":" + String(sec % 60).padStart(2, "0");
}

// addExercise renders one row in #todayEx based on the input's kind.
// Input is either an Exercise (from clicking the exercise list) or a Set
// (from setFormContent recalling saved sets). Both shapes have .Name.
function addExercise(obj) {
    id = id + 1;
    obj = obj || {};
    const name = obj.Name || "";
    const kind = obj.Kind || kindForName(name);
    const html = (kind === "cardio")
        ? renderCardioRow(id, name, obj)
        : renderStrengthRow(id, name, obj);
    document.getElementById('todayEx').insertAdjacentHTML('beforeend', html);
}

function renderStrengthRow(rowId, name, obj) {
    const safeName = escapeHTMLAttr(name);
    const safeWeight = escapeHTMLAttr(obj.Weight ?? 0);
    const safeReps = escapeHTMLAttr(obj.Reps ?? 0);
    const safeNote = escapeHTMLAttr(obj.Note || "");
    return `<tr id="${rowId}" data-kind="strength">
    <td class="todayex-name">
        <input name="name" type="text" class="form-control todayex-name-input" value="${safeName}">
    </td><td class="todayex-weight">
        <input name="weight" type="number" step="any" min="0" class="form-control" value="${safeWeight}">
    </td><td class="todayex-reps">
        <input name="reps" type="number" min="0" class="form-control" value="${safeReps}">
    </td><td class="todayex-note d-none d-md-table-cell">
        <input name="note" type="text" class="form-control" placeholder="(optional)" value="${safeNote}">
        <input type="hidden" name="duration_mmss" value="">
        <input type="hidden" name="distance_value" value="0">
        <input type="hidden" name="avg_hr" value="0">
        <input type="hidden" name="max_hr" value="0">
        <input type="hidden" name="calories" value="0">
        <input type="hidden" name="equipment" value="">
    </td><td class="todayex-del">
        <button class="btn del-set-button" type="button" title="Delete" onclick="delExercise(${rowId})">
            <i class="bi bi-x-square"></i>
        </button>
    </td></tr>`;
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
        <input name="name" type="text" class="form-control todayex-name-input" value="${safeName}">
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
        <div class="hstack gap-1 justify-content-end">
            <button class="btn del-set-button p-1" type="button" title="Advanced (HR / cal${isBike ? " / bike" : ""})" onclick="toggleCardioAdvanced(${rowId})">
                <i class="bi bi-sliders"></i>
            </button>
            <button class="btn del-set-button p-1" type="button" title="Delete" onclick="delExercise(${rowId})">
                <i class="bi bi-x-square"></i>
            </button>
        </div>
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
}

function setWeightDate() {
    let date = document.getElementById("realDate").value;
    document.getElementById("weightDate").value = date;
}

function delExercise(exID) {
    const row = document.getElementById(exID);
    const adv = document.getElementById('adv-' + exID);
    if (adv) adv.remove();
    if (row) row.remove();
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
