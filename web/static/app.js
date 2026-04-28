const App = (() => {
    const storageKey = "numbers.profileId";
    let profileId = localStorage.getItem(storageKey) || "";

    function getCookie(name) {
        const rows = document.cookie ? document.cookie.split(";") : [];
        for (const row of rows) {
            const [rawKey, ...rest] = row.trim().split("=");
            if (rawKey === name) return decodeURIComponent(rest.join("="));
        }

        return "";
    }

    function looksValidProfileId(value) {
        return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
            value || ""
        );
    }

    function applyProfile(id) {
        profileId = id;
        localStorage.setItem(storageKey, id);
        document.cookie = `numbers_profile_id=${encodeURIComponent(id)}; path=/; max-age=31536000; samesite=lax`;

        if (window.htmx) {
            htmx.config.headers = htmx.config.headers || {};
            htmx.config.headers["X-Profile-ID"] = id;
        }
    }

    function clearProfile() {
        profileId = "";
        localStorage.removeItem(storageKey);
        document.cookie = "numbers_profile_id=; path=/; max-age=0; samesite=lax";
    }

    if (!profileId) {
        profileId = getCookie("numbers_profile_id") || "";
    }

    function formatMMSS(totalSeconds) {
        const safe = Math.max(0, Number(totalSeconds) || 0);
        
        const m = Math.floor(safe / 60)
            .toString()
            .padStart(2, "0");

        const s = Math.floor(safe % 60)
            .toString()
            .padStart(2, "0");

        return `${m}:${s}`;
    }

    function tickCountdown() {
        const timer = document.getElementById("next-roll-timer");
        if (!timer) return;

        // If we're waiting for a state update after hitting zero, don't decrement further
        if (timer.dataset.waitingForState === "true") {
            return;
        }

        const raw = Number(timer.dataset.nextRollSeconds || "0");
        const next = Math.max(0, raw - 1);

        timer.dataset.nextRollSeconds = String(next);
        timer.textContent = formatMMSS(next);

        // If we hit zero, refresh the roll state immediately to get the new timer
        if (next === 0) {
            timer.dataset.waitingForState = "true";
            requestPanel("/roll/state", "#roll-controls");
        }
    }

    async function requestPanel(url, targetSelector) {
        const target = document.querySelector(targetSelector);
        if (!target) return;

        try {
            const response = await fetch(url, {
                method: "GET",
                headers: profileId ? { "X-Profile-ID": profileId } : {},
                credentials: "same-origin",
            });

            target.innerHTML = await response.text();
            if (window.htmx) {
                htmx.process(target);
            }
            // If we just updated the roll controls, reset the waiting flag
            if (targetSelector === "#roll-controls") {
                const timer = document.getElementById("next-roll-timer");
                if (timer) {
                    timer.dataset.waitingForState = "false";
                }
            }
        } catch (err) {
            target.innerHTML = '<div class="panel"><p class="meta">Unable to load panel right now.</p></div>';
            // If we failed to update the roll controls, reset the waiting flag so the timer can continue
            if (targetSelector === "#roll-controls") {
                const timer = document.getElementById("next-roll-timer");
                if (timer) {
                    timer.dataset.waitingForState = "false";
                }
            }
        }
    }

    async function refreshAllPanels() {
        await Promise.all([
            requestPanel("/roll/state", "#roll-controls"),
            requestPanel("/profile/view", "#profile-mini"),
            requestPanel("/history", "#history-panel"),
            requestPanel("/specs/unlocked", "#unlocked-specs-panel"),
            requestPanel("/leaderboard", "#leaderboard-panel"),
            requestPanel("/leaderboard/total-value", "#total-value-leaderboard-panel"),
        ]);
    }

    async function ensureProfile() {
        if (looksValidProfileId(profileId)) {
            applyProfile(profileId);
            return profileId;
        }

        clearProfile();

        const response = await fetch("/profile/init", { method: "POST" });
        if (!response.ok) throw new Error("profile init failed");

        const payload = await response.json();
        if (!looksValidProfileId(payload.profile_id)) {
            throw new Error("invalid profile id response");
        }

        applyProfile(payload.profile_id);
        return profileId;
    }

    function switchView(viewName) {
        document.querySelectorAll(".nav-btn").forEach((btn) => {
            btn.classList.toggle("active", btn.dataset.view === viewName);
        });

        document.querySelectorAll(".view-section").forEach((sec) => {
            sec.classList.toggle("active", sec.id === "view-" + viewName);
        });

        if (viewName === "history") {
            requestPanel("/history", "#history-panel");
            requestPanel("/specs/unlocked", "#unlocked-specs-panel");
        } else if (viewName === "leaderboard") {
            requestPanel("/leaderboard", "#leaderboard-panel");
            requestPanel("/leaderboard/total-value", "#total-value-leaderboard-panel");
        }
    }

    async function loadUsername() {
        try {
            const response = await fetch("/profile/view", {
                headers: profileId ? { "X-Profile-ID": profileId } : {},
                credentials: "same-origin",
            });
            const html = await response.text();

            const parser = new DOMParser();
            const doc = parser.parseFromString(html, "text/html");
            const input = doc.querySelector('input[name="username"]');
            if (input) {
                document.getElementById("header-username").value = input.value;
            }
        } catch (err) {}
    }

    async function saveUsername() {
        const username = document.getElementById("header-username").value.trim();
        if (!username) return;

        try {
            const response = await fetch("/profile/username", {
                method: "POST",
                headers: {
                "Content-Type": "application/x-www-form-urlencoded",
                ...(profileId ? { "X-Profile-ID": profileId } : {}),
                },
                credentials: "same-origin",
                body: new URLSearchParams({ username }),
            });

            if (response.ok) {
                const btn = document.querySelector(".username-save");
                const original = btn.textContent;
                btn.textContent = "Saved!";
                setTimeout(() => (btn.textContent = original), 1200);
                requestPanel("/profile/view", "#profile-mini");
            }
        } catch (err) {}
    }

    function bindEvents() {
        document.getElementById("header-username")?.addEventListener("keydown", (e) => {
            if (e.key === "Enter") saveUsername();
        });

        document.body.addEventListener("htmx:configRequest", (event) => {
            if (profileId) {
                event.detail.headers["X-Profile-ID"] = profileId;
            }
        });

        document.body.addEventListener("refresh-roll-state", () =>
            requestPanel("/roll/state", "#roll-controls")
        );

        document.body.addEventListener("refresh-history", () => {
            if (document.getElementById("view-history").classList.contains("active")) {
                requestPanel("/history", "#history-panel");
            }
        });

        document.body.addEventListener("refresh-unlocked-specs", () => {
            if (document.getElementById("view-history").classList.contains("active")) {
                requestPanel("/specs/unlocked", "#unlocked-specs-panel");
            }
        });

        document.body.addEventListener("refresh-leaderboard", () => {
            if (document.getElementById("view-leaderboard").classList.contains("active")) {
                requestPanel("/leaderboard", "#leaderboard-panel");
                requestPanel("/leaderboard/total-value", "#total-value-leaderboard-panel");
            }
        });
    }

    async function init() {
        try {
            await ensureProfile();
            await loadUsername();
            await refreshAllPanels();
            setInterval(tickCountdown, 1000);
            setInterval(() => requestPanel("/roll/state", "#roll-controls"), 15000);
            setInterval(() => {
                if (document.getElementById("view-history").classList.contains("active")) {
                    requestPanel("/history", "#history-panel");
                    requestPanel("/specs/unlocked", "#unlocked-specs-panel");
                }
            }, 60000);
            setInterval(() => {
                if (document.getElementById("view-leaderboard").classList.contains("active")) {
                    requestPanel("/leaderboard", "#leaderboard-panel");
                    requestPanel("/leaderboard/total-value", "#total-value-leaderboard-panel");
                }
            }, 30000);
        } catch (err) {
            document.getElementById("roll-result").innerHTML = '<p class="meta">Unable to initialize profile.</p>';
        }
    }

    bindEvents();

    return {
        init,
        switchView,
        saveUsername,
    };
})();

document.addEventListener("DOMContentLoaded", App.init);