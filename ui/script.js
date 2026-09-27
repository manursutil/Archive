const out = document.getElementById("out")

function render(rows, detail) {
  out.replaceChildren(...rows.map(row => {
    const li = document.createElement("li");
    li.textContent = row.source + " ";

    const meta = document.createElement("span");
    meta.className = "meta";
    meta.textContent = detail(row) + " ";

    const del = document.createElement("button");
    del.textContent = "Delete";
    del.onclick = async () => {
      await fetch(`/items/${row.id}`, { method: "DELETE" });
      load();
    };

    li.append(meta, del);
    return li;
  }));
}

async function load() {
  const res = await fetch("/items");
  render(await res.json() ?? [], row => row.savedAt);
}

document.getElementById("find").oninput = async (e) => {
  const q = e.target.value.trim();
  if (!q) return load();

  const res = await fetch(`/search?q=${encodeURIComponent(q)}`);
  if (!res.ok) return;

  render(await res.json() ?? [], row => row.snippet);
};

document.getElementById("find").onsubmit = (e) => e.preventDefault();

document.getElementById("add").onsubmit = async (e) => {
  e.preventDefault();

  const res = await fetch("/items", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ source: e.target.source.value }),
  });

  if (!res.ok) return alert(await res.text());

  e.target.reset();
  
  load();
};

load();