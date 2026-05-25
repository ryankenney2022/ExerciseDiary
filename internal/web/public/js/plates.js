// plates.js — client side of the plate calculator.
//
// Provides two functions on the window:
//   - suggestPlates(targetWeight): returns HTML for an inline plate hint
//     (used by index.js refreshPlateHint under each strength row's weight
//     stepper). Returns "" if no suggestion is meaningful.
//   - formatSuggestion(json): renders the JSON returned by /equipment/suggest
//     as a readable HTML block (used by the equipment page's "Try it" panel).
//
// suggestPlates uses a tiny in-memory cache keyed by target weight to avoid
// re-fetching on every keystroke when the user is bumping the stepper.

(function() {
    const cache = new Map();
    let inflight = null;
    let scheduledKey = null;

    // formatSuggestion: human-readable summary of a /equipment/suggest response.
    window.formatSuggestion = function(j) {
        if (!j) return "";
        const target = parseFloat(j.target || "0") || 0;
        const bar = parseFloat(j.bar || "0") || 0;
        const remainder = parseFloat(j.remainder || "0") || 0;
        const achievable = parseFloat(j.achievable || "0") || 0;
        const unit = j.unit || "lb";
        const side = j.side || [];

        if (target <= bar) {
            return `<div>Bar (<strong>${bar} ${unit}</strong>) already covers the target.</div>`;
        }
        if (!side.length) {
            return `<div class="text-warning">No plates fit — bar is ${bar} ${unit}; you need ${(target - bar)/2} ${unit} per side but have nothing that small.</div>`;
        }

        const perSide = side.map(p => `${p.n} × ${p.w}`).join(" + ");
        const totalPerSide = side.reduce((a, p) => a + parseFloat(p.w) * p.n, 0);
        let html = `<div class="mb-1">Per side: <strong>${perSide}</strong> = ${roundDisplay(totalPerSide)} ${unit}</div>`;
        html += `<div class="text-muted">Bar ${bar} + 2 × ${roundDisplay(totalPerSide)} = <strong>${roundDisplay(achievable)} ${unit}</strong></div>`;
        if (remainder > 0) {
            html += `<div class="text-warning small mt-1">Short by ${roundDisplay(remainder * 2)} ${unit} total — closest your inventory can hit.</div>`;
        }
        return html;
    };

    // Inline strength-row hint: small one-line plate spec.
    window.suggestPlates = function(targetWeight) {
        const t = parseFloat(targetWeight || "0") || 0;
        if (t <= 0) return "";
        const key = String(t);

        if (cache.has(key)) {
            return cache.get(key);
        }

        // First call returns "" while we kick off the fetch; the input-event
        // handler will fire again when the response arrives and updates the
        // hint. We also debounce concurrent fetches per row.
        if (scheduledKey !== key) {
            scheduledKey = key;
            fetchAndCache(t);
        }
        return cache.get(key) || "";
    };

    async function fetchAndCache(target) {
        if (inflight) return; // serialize to avoid swamping the server
        inflight = (async () => {
            try {
                const r = await fetch('/equipment/suggest?weight=' + encodeURIComponent(String(target)));
                if (!r.ok) return;
                const j = await r.json();
                cache.set(String(target), renderInline(j));
                // Trigger refresh on all visible plate-hint slots so newly
                // arrived values show up without the user touching the input.
                document.querySelectorAll('tr[data-kind="strength"]').forEach(row => {
                    const w = row.querySelector('input[name="weight"]');
                    const hint = row.querySelector('.todayex-plate-hint');
                    if (!w || !hint) return;
                    const wv = parseFloat(w.value || "0") || 0;
                    if (String(wv) === String(target)) {
                        hint.innerHTML = cache.get(String(target));
                    }
                });
            } catch (e) {
                // swallow — hint stays empty
            } finally {
                inflight = null;
            }
        })();
    }

    function renderInline(j) {
        const target = parseFloat(j.target || "0") || 0;
        const bar = parseFloat(j.bar || "0") || 0;
        const remainder = parseFloat(j.remainder || "0") || 0;
        const side = j.side || [];
        const unit = j.unit || "lb";

        if (target <= bar) return ""; // no plates needed
        if (!side.length) {
            return `<span class="text-warning">no plate fits (need ${roundDisplay((target-bar)/2)} ${unit}/side)</span>`;
        }
        const spec = side.map(p => p.n > 1 ? `${p.n}×${p.w}` : `${p.w}`).join(" + ");
        let s = `<span class="text-muted">per side:</span> ${spec}`;
        if (remainder > 0) {
            s += ` <span class="text-warning">(${roundDisplay(remainder * 2)} short)</span>`;
        }
        return s;
    }

    function roundDisplay(n) {
        // Trim trailing zeros: 5.0 -> "5", 2.50 -> "2.5"
        return String(Math.round(n * 100) / 100);
    }
})();
