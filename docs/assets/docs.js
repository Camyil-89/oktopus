const ORDER = [...document.querySelectorAll(".nav-item:not(.disabled)")].map((n) => n.dataset.doc);
const main = document.getElementById("main");
let current = "intro";

const copyLabel = document.body.dataset.copy || "Copy";
const copiedLabel = document.body.dataset.copied || "Copied";

function go(id) {
  if (!ORDER.includes(id)) return;
  current = id;

  document.querySelectorAll(".doc").forEach((d) => d.classList.toggle("active", d.id === "doc-" + id));
  document.querySelectorAll(".nav-item").forEach((n) => n.classList.toggle("active", n.dataset.doc === id));

  const active = document.getElementById("doc-" + id);
  document.getElementById("crumb-title").textContent = active.dataset.title;
  document.getElementById("crumb-section").textContent = active.dataset.section;

  main.scrollTop = 0;
  buildToc(active);
  updatePager(id);
  if (history.replaceState) history.replaceState(null, "", "#" + id);
}

document.querySelectorAll(".nav-item:not(.disabled)").forEach((item) => {
  item.addEventListener("click", () => go(item.dataset.doc));
});

function buildToc(doc) {
  const list = document.getElementById("toc-list");
  list.innerHTML = "";
  const heads = doc.querySelectorAll("h2, h3");

  heads.forEach((h, i) => {
    if (!h.id) h.id = doc.id + "-h" + i;
    const li = document.createElement("li");
    const a = document.createElement("a");
    a.href = "#" + h.id;
    a.textContent = h.textContent;
    a.className = "toc-link" + (h.tagName === "H3" ? " toc-sub" : "");
    a.dataset.target = h.id;
    a.addEventListener("click", (e) => {
      e.preventDefault();
      h.scrollIntoView({ behavior: "smooth", block: "start" });
    });
    li.appendChild(a);
    list.appendChild(li);
  });

  spy();
}

function spy() {
  const doc = document.getElementById("doc-" + current);
  if (!doc) return;
  const heads = [...doc.querySelectorAll("h2, h3")];
  if (!heads.length) return;

  let activeId = heads[0].id;
  for (const h of heads) {
    if (h.getBoundingClientRect().top < 140) activeId = h.id;
  }
  document.querySelectorAll(".toc-link").forEach((a) => {
    a.classList.toggle("is-active", a.dataset.target === activeId);
  });
}

let ticking = false;
main.addEventListener("scroll", () => {
  if (ticking) return;
  ticking = true;
  requestAnimationFrame(() => {
    spy();
    ticking = false;
  });
});

function updatePager(id) {
  const i = ORDER.indexOf(id);
  const prev = ORDER[i - 1];
  const next = ORDER[i + 1];

  const prevBtn = document.getElementById("pager-prev");
  const nextBtn = document.getElementById("pager-next");

  if (prev) {
    prevBtn.style.visibility = "visible";
    document.getElementById("pager-prev-title").textContent =
      document.getElementById("doc-" + prev).dataset.title;
    prevBtn.onclick = () => go(prev);
  } else {
    prevBtn.style.visibility = "hidden";
  }

  if (next) {
    nextBtn.style.visibility = "visible";
    document.getElementById("pager-next-title").textContent =
      document.getElementById("doc-" + next).dataset.title;
    nextBtn.onclick = () => go(next);
  } else {
    nextBtn.style.visibility = "hidden";
  }
}

document.getElementById("nav-search").addEventListener("input", (e) => {
  const q = e.target.value.trim().toLowerCase();
  document.querySelectorAll("#nav .nav-item").forEach((n) => {
    const t = n.textContent.toLowerCase();
    n.style.display = !q || t.includes(q) ? "" : "none";
  });
  document.querySelectorAll("#nav > div").forEach((g) => {
    const visible = [...g.querySelectorAll(".nav-item")].some((n) => n.style.display !== "none");
    g.style.display = visible ? "" : "none";
  });
});

function copyCode(btn) {
  const code = btn.closest(".code").querySelector("pre");
  const text = code.innerText;
  navigator.clipboard.writeText(text).then(() => {
    const old = btn.textContent;
    btn.textContent = copiedLabel;
    btn.style.color = "#5eead4";
    btn.style.borderColor = "rgba(94,234,212,.3)";
    setTimeout(() => {
      btn.textContent = old;
      btn.style.color = "";
      btn.style.borderColor = "";
    }, 1400);
  });
}

window.copyCode = copyCode;

const hash = location.hash.replace("#", "");
go(ORDER.includes(hash) ? hash : "intro");
