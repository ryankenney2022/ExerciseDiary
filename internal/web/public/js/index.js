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

function addExercise(name, weight, reps, note) {
    id = id + 1;
    const safeName = escapeHTMLAttr(name);
    const safeWeight = escapeHTMLAttr(weight);
    const safeReps = escapeHTMLAttr(reps);
    const safeNote = escapeHTMLAttr(note);

    const html_to_insert = `<tr id="${id}">
    <td>
        <input name="name" type="text" class="form-control" value="${safeName}">
    </td><td>
        <input name="weight" type="number" step="any" min="0" class="form-control" value="${safeWeight}">
    </td><td>
        <input name="reps" type="number" min="0" class="form-control" value="${safeReps}">
    </td><td>
        <input name="note" type="text" class="form-control" placeholder="(optional)" value="${safeNote}">
    </td><td>
        <button class="btn del-set-button" type="button" title="Delete" onclick="delExercise(${id})">
            <i class="bi bi-x-square"></i>
        </button>
    </td></tr>`;

    document.getElementById('todayEx').insertAdjacentHTML('beforeend', html_to_insert);
};

function setFormContent(sets, date) {
    window.sessionStorage.setItem("today", date);
    document.getElementById('todayEx').innerHTML = "";
    document.getElementById("formDate").value = date;
    document.getElementById("realDate").value = date;

    if (sets) {
        let len = sets.length;
        for (let i = 0 ; i < len; i++) {
            if (sets[i].Date == date) {
                addExercise(sets[i].Name, sets[i].Weight, sets[i].Reps, sets[i].Note);
            }
        }
    }
};

function setFormDate(sets) {
    today = document.getElementById("realDate").value;
    if (!today) {
        today = window.sessionStorage.getItem("today");

        if (!today) {
            today = new Date().toJSON().slice(0, 10);
        }
    }

    setFormContent(sets, today);
};

function setWeightDate() {
    let date = document.getElementById("realDate").value;
    document.getElementById("weightDate").value = date;
};

function delExercise(exID) {

    document.getElementById(exID).remove();
};

function moveDayLeftRight(where, sets) {
    dateStr = document.getElementById("realDate").value;

    let year  = dateStr.substring(0,4);
    let month = dateStr.substring(5,7);
    let day   = dateStr.substring(8,10);
    var date  = new Date(year, month-1, day);

    date.setDate(date.getDate() + parseInt(where));
    let left = date.toLocaleDateString('en-CA');

    setFormContent(sets, left);
};

function addAllGroup(exs, gr) {
    if (exs) {
        let len = exs.length;
        for (let i = 0 ; i < len; i++) {
            if (exs[i].Group == gr) {
                addExercise(exs[i].Name, exs[i].Weight, exs[i].Reps, "");
            }
        }
    }
}
