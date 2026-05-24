var offset = 0;

function setToday() {
    let today = new Date().toJSON().slice(0, 10);
    document.getElementById("todayDate").value = today;
};

// Pre-fill the Add form with an existing row's values; switch button label
// to "Update". Hidden `id` field gets the row id so the server updates
// instead of inserting.
function editWeight(id, date, weight) {
    document.getElementById("weightId").value = id;
    document.getElementById("todayDate").value = date;
    document.getElementById("weightValue").value = weight;
    document.getElementById("weightSubmit").innerText = "Update";
    document.getElementById("weightCancel").classList.remove("d-none");
    document.getElementById("todayDate").scrollIntoView({behavior: "smooth", block: "center"});
};

function cancelEditWeight() {
    document.getElementById("weightId").value = "";
    document.getElementById("weightValue").value = "";
    document.getElementById("weightSubmit").innerText = "Add";
    document.getElementById("weightCancel").classList.add("d-none");
    setToday();
};

function addWeightRow(i, date, weight, id) {
    // Escape values for use in inline JS args
    const safeDate = String(date).replace(/'/g, "\\'");
    const safeWeight = String(weight).replace(/'/g, "\\'");

    const html_code = `<tr>
        <td style="opacity: 45%;">${i}.</td>
        <td>${date}</td>
        <td>${weight}</td>
        <td class="text-nowrap">
            <button type="button" class="btn del-set-button" title="Edit"
                    onclick="editWeight(${id}, '${safeDate}', '${safeWeight}')">
                <i class="bi bi-pencil-square"></i>
            </button>
            <a href="/weight/?del=${id}" onclick="return confirm('Delete this weight entry?');">
                <button type="button" class="btn del-set-button" title="Delete">
                    <i class="bi bi-x-square"></i>
                </button>
            </a>
        </td>
    </tr>`;
    document.getElementById('weightList').insertAdjacentHTML('beforeend', html_code);
};

function setWeights(weights, wcolor, off, step) {
    // Render newest first.
    const sorted = (weights || []).slice().sort((a, b) => (a.Date < b.Date ? 1 : -1));
    const arrayLength = sorted.length;

    offset = Math.max(0, offset + off);
    const start = Math.min(offset * step, Math.max(0, arrayLength - step));
    const end = Math.min(start + step, arrayLength);

    document.getElementById('weightList').innerHTML = "";

    let dates = [], ws = [];
    for (let i = start; i < end; i++) {
        // Display index counts from the newest item being #1
        addWeightRow(i + 1, sorted[i].Date, sorted[i].Weight, sorted[i].ID);
        dates.push(sorted[i].Date);
        ws.push(sorted[i].Weight);
    }

    // Chart goes oldest-left to newest-right regardless of list direction.
    weightChart('weight-chart', dates.slice().reverse(), ws.slice().reverse(), wcolor, true);
};
