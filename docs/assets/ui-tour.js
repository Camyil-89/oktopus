function initMockTabs() {
  document.querySelectorAll(".ui-mock-tabs[data-mock-tabs]").forEach((bar) => {
    const root = bar.closest(".ui-mock-content") || bar.parentElement;
    if (!root) return;
    bar.querySelectorAll("[data-mock-tab]").forEach((tab) => {
      tab.addEventListener("click", (e) => {
        e.stopPropagation();
        const id = tab.getAttribute("data-mock-tab");
        if (!id) return;
        bar.querySelectorAll("[data-mock-tab]").forEach((t) => {
          t.classList.toggle("is-active", t === tab);
        });
        root.querySelectorAll("[data-mock-tab-panel]").forEach((panel) => {
          panel.classList.toggle("is-active", panel.getAttribute("data-mock-tab-panel") === id);
        });
        document.dispatchEvent(new CustomEvent("ui-mock-tab-change", { detail: { tab: id } }));
      });
    });
  });
}

function initUiTour() {
  const hintTitle = document.getElementById("ui-tour-title");
  const hintBody = document.getElementById("ui-tour-body");
  const defaultTitle = hintTitle?.dataset.defaultTitle || "";
  const defaultBody = hintBody?.dataset.defaultBody || "";

  const wrap = document.querySelector(".ui-mock-wrap");
  if (!wrap || !hintTitle || !hintBody) return;

  function allZones() {
    return wrap.querySelectorAll(".tour-zone[data-tour-title]");
  }

  function setDefault() {
    hintTitle.textContent = defaultTitle;
    hintBody.textContent = defaultBody;
    allZones().forEach((z) => z.classList.remove("is-tour-active"));
  }

  function bindZone(zone) {
    if (zone.dataset.tourBound === "1") return;
    zone.dataset.tourBound = "1";
    zone.addEventListener("mouseenter", () => {
      if (!zone.offsetParent && zone.closest("[data-mock-tab-panel]")) return;
      allZones().forEach((z) => z.classList.remove("is-tour-active"));
      zone.classList.add("is-tour-active");
      hintTitle.textContent = zone.dataset.tourTitle || "";
      hintBody.textContent = zone.dataset.tourBody || "";
    });
    zone.addEventListener("focus", () => {
      zone.dispatchEvent(new Event("mouseenter"));
    });
  }

  allZones().forEach(bindZone);

  wrap.addEventListener("mouseleave", (e) => {
    if (!wrap.contains(e.relatedTarget)) setDefault();
  });

  document.addEventListener("ui-mock-tab-change", () => {
    setDefault();
    allZones().forEach(bindZone);
  });
}

function initUiDocs() {
  initMockTabs();
  initUiTour();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initUiDocs);
} else {
  initUiDocs();
}
