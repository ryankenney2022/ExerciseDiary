// plans.js — Workout plan builder. Renders editable item rows, swaps the
// target input between reps and mm:ss based on the chosen exercise's kind/mode,
// and supports drag-reorder. Submission posts parallel arrays:
//   ex_id[], target_sets[], target_reps[], target_seconds[], item_note[]
// Every row always contributes exactly one value to each array (hidden inputs
// keep them aligned), so the server can index them positionally.

(function () {
    let rowSeq = 0;

    function escAttr(s) {
        return String(s == null ? "" : s)
            .replace(/&/g, "&amp;").replace(/"/g, "&quot;")
            .replace(/</g, "&lt;").replace(/>/g, "&gt;");
    }
    function exByName(name) {
        for (const e of (window.allExercises || [])) if (e.Name === name) return e;
        return null;
    }
    function exByID(id) {
        for (const e of (window.allExercises || [])) if (e.ID === id) return e;
        return null;
    }
    function isTimed(ex) {
        return !!ex && (ex.Kind === "cardio" || ex.Mode === "timed");
    }
    function secondsToMMSS(sec) {
        sec = parseInt(sec, 10) || 0;
        if (sec <= 0) return "";
        const m = Math.floor(sec / 60), s = sec % 60;
        return m + ":" + String(s).padStart(2, "0");
    }

    // targetInputHTML returns the target widget plus a hidden counterpart so
    // both target_reps[] and target_seconds[] arrays stay length-aligned.
    function targetInputHTML(timed, prefill) {
        prefill = prefill || {};
        if (timed) {
            return '<input name="target_seconds" class="form-control" placeholder="mm:ss" value="' + secondsToMMSS(prefill.TargetSeconds) + '">' +
                   '<input type="hidden" name="target_reps" value="0">';
        }
        return '<input name="target_reps" type="number" min="0" class="form-control" placeholder="Reps" value="' + (prefill.TargetReps || "") + '">' +
               '<input type="hidden" name="target_seconds" value="">';
    }

    // addItemRow appends one builder row. prefill is optional:
    // {ExerciseID, TargetSets, TargetReps, TargetSeconds, Note}.
    window.addItemRow = function (prefill) {
        prefill = prefill || {};
        rowSeq += 1;
        const rid = "item-" + rowSeq;
        const ex = prefill.ExerciseID ? exByID(prefill.ExerciseID) : null;
        const exName = ex ? ex.Name : "";
        const timed = isTimed(ex);

        const row = document.createElement("div");
        row.className = "plan-row plan-item-row mb-2";
        row.id = rid;
        row.innerHTML =
            '<span class="plan-drag" style="cursor:grab" title="Drag to reorder"><i class="bi bi-grip-vertical"></i></span>' +
            '<input type="hidden" name="ex_id" value="' + (ex ? ex.ID : "") + '">' +
            '<input list="exerciseOptions" class="form-control plan-name plan-ex-name" placeholder="Exercise" value="' + escAttr(exName) + '">' +
            '<input name="target_sets" type="number" min="0" class="form-control plan-sets" placeholder="Sets" value="' + (prefill.TargetSets || "") + '">' +
            '<div class="plan-target">' + targetInputHTML(timed, prefill) + '</div>' +
            '<input name="item_note" class="form-control plan-note" placeholder="Cue / note" value="' + escAttr(prefill.Note) + '">' +
            '<button type="button" class="btn del-set-button" title="Remove"><i class="bi bi-x-lg"></i></button>';
        document.getElementById("itemRows").appendChild(row);

        // Resolve hidden ex_id and swap target widget when the exercise changes.
        const nameInput = row.querySelector(".plan-ex-name");
        nameInput.addEventListener("change", function () {
            const picked = exByName(this.value);
            row.querySelector('input[name="ex_id"]').value = picked ? picked.ID : "";
            row.querySelector(".plan-target").innerHTML = targetInputHTML(isTimed(picked), {});
        });

        // Remove button.
        row.querySelector(".del-set-button").addEventListener("click", function () {
            row.remove();
        });
    };

    document.addEventListener("DOMContentLoaded", function () {
        const itemRows = document.getElementById("itemRows");
        if (!itemRows) return;

        // Prefill source: "save as plan" stash (sessionStorage) wins, else the
        // existing template's items, else a single empty row.
        let items = window.templateItems || [];
        const stash = sessionStorage.getItem("planFromDay");
        if (stash) {
            try { items = JSON.parse(stash); } catch (e) { items = []; }
            sessionStorage.removeItem("planFromDay");
            const nm = sessionStorage.getItem("planFromDayName");
            if (nm) {
                const nameEl = document.getElementById("planName");
                if (nameEl) nameEl.value = nm;
                sessionStorage.removeItem("planFromDayName");
            }
        }

        if (!items.length) {
            addItemRow();
        } else {
            items.forEach(it => addItemRow(it));
        }

        if (window.Sortable) {
            new Sortable(itemRows, { handle: ".plan-drag", animation: 150 });
        }
    });
})();
