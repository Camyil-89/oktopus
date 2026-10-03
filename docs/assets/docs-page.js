const main = document.getElementById("main");
const article = document.querySelector(".doc-page");
const copiedLabel = document.body.dataset.copied || "Copied";

function buildToc(root) {
  const list = document.getElementById("toc-list");
  if (!list || !root) return;
  list.innerHTML = "";
  const heads = root.querySelectorAll("h2, h3");

  heads.forEach((h, i) => {
    if (!h.id) h.id = "section-h" + i;
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
  if (!article) return;
  const heads = [...article.querySelectorAll("h2, h3")];
  if (!heads.length) return;

  let activeId = heads[0].id;
  for (const h of heads) {
    if (h.getBoundingClientRect().top < 140) activeId = h.id;
  }
  document.querySelectorAll(".toc-link").forEach((a) => {
    a.classList.toggle("is-active", a.dataset.target === activeId);
  });
}

if (main) {
  let ticking = false;
  main.addEventListener("scroll", () => {
    if (ticking) return;
    ticking = true;
    requestAnimationFrame(() => {
      spy();
      ticking = false;
    });
  });
}

const navSearch = document.getElementById("nav-search");
if (navSearch) {
  navSearch.addEventListener("input", (e) => {
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
}

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

function copyAiPrompt(btn) {
  const pre = document.getElementById("ai-prompt-pre");
  if (!pre) return;
  const text = pre.innerText;
  const copyLabel = document.body.dataset.copy || "Copy";
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

window.copyAiPrompt = copyAiPrompt;

if (article) buildToc(article);
