// rest-timer.js — between-sets cool-down timer.
//
// Activated when window.restTimer.enabled is true (set from the user profile).
// index.js calls window.onStrengthSetAdded() right after every strength row is
// added; this module restarts a floating countdown chip pinned to the bottom
// of the today-workout column.
//
// When the timer hits zero:
//   - Chip background flashes green
//   - Short WebAudio "ding" plays (silent if audio is blocked / unprimed)
//   - navigator.vibrate(200) on supported phones
//
// The chip stays visible after firing until the next add, so the user can
// glance at it for confirmation. Clicking the chip cancels the countdown.

(function() {
    let chipEl = null;
    let intervalId = null;
    let remaining = 0;

    function ensureChip() {
        if (chipEl) return chipEl;
        chipEl = document.createElement('div');
        chipEl.id = 'restTimerChip';
        chipEl.className = 'rest-timer-chip d-none';
        chipEl.title = 'Tap to cancel rest timer';
        chipEl.addEventListener('click', cancel);
        document.body.appendChild(chipEl);
        return chipEl;
    }

    function fmt(sec) {
        sec = Math.max(0, Math.floor(sec));
        const m = Math.floor(sec / 60);
        const s = sec % 60;
        return m + ':' + String(s).padStart(2, '0');
    }

    function render() {
        if (!chipEl) return;
        chipEl.innerHTML = '<i class="bi bi-stopwatch"></i> <span class="rest-timer-text">' + fmt(remaining) + '</span>';
    }

    function cancel() {
        if (intervalId) { clearInterval(intervalId); intervalId = null; }
        if (chipEl) chipEl.classList.add('d-none');
    }

    function start(seconds) {
        ensureChip();
        if (intervalId) clearInterval(intervalId);
        remaining = seconds;
        chipEl.classList.remove('d-none');
        chipEl.classList.remove('rest-timer-done');
        render();

        const startedAt = Date.now();
        const total = seconds;
        intervalId = setInterval(() => {
            const elapsed = (Date.now() - startedAt) / 1000;
            remaining = Math.max(0, Math.ceil(total - elapsed));
            render();
            if (remaining <= 0) {
                clearInterval(intervalId);
                intervalId = null;
                fire();
            }
        }, 250);
    }

    function fire() {
        if (chipEl) chipEl.classList.add('rest-timer-done');
        try { ding(); } catch (e) {}
        try { if (navigator.vibrate) navigator.vibrate(200); } catch (e) {}
    }

    // ding: minimal WebAudio beep. Browsers require a prior user gesture before
    // audio works; the strength-row click that schedules the timer is itself a
    // gesture, so a freshly-armed AudioContext usually plays. Errors swallowed.
    let audioCtx = null;
    function ding() {
        if (!window.AudioContext && !window.webkitAudioContext) return;
        if (!audioCtx) {
            const Ctx = window.AudioContext || window.webkitAudioContext;
            audioCtx = new Ctx();
        }
        // Resume if the page suspended the context.
        if (audioCtx.state === "suspended") audioCtx.resume();

        const now = audioCtx.currentTime;
        const osc = audioCtx.createOscillator();
        const gain = audioCtx.createGain();
        osc.type = "sine";
        osc.frequency.setValueAtTime(880, now);
        osc.frequency.exponentialRampToValueAtTime(660, now + 0.18);
        gain.gain.setValueAtTime(0.0001, now);
        gain.gain.exponentialRampToValueAtTime(0.25, now + 0.02);
        gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.35);
        osc.connect(gain).connect(audioCtx.destination);
        osc.start(now);
        osc.stop(now + 0.4);
    }

    // Public hook called by index.js after every strength row add.
    window.onStrengthSetAdded = function() {
        const cfg = window.restTimer || {};
        if (!cfg.enabled) return;
        const seconds = parseInt(cfg.seconds, 10) || 90;
        start(seconds);
    };
})();
