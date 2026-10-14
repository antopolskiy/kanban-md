(() => {
  const body = document.body;
  const csrf = body.dataset.csrf || "";

  const selectedCard = () => document.querySelector("[data-card].selected");

  const selectCard = (card) => {
    for (const el of document.querySelectorAll("[data-card]")) {
      el.classList.remove("selected");
    }
    card.classList.add("selected");
    const url = new URL(location.href);
    url.searchParams.set("sel", card.dataset.id);
    history.replaceState(null, "", url);
  };

  const post = async (path, fields) => {
    const params = new URLSearchParams({
      csrf,
      return: `${location.pathname}${location.search}`,
      ...fields,
    });
    const res = await fetch(path, { method: "POST", body: params });
    if (res.ok) {
      location.href = res.url;
      return;
    }
    const doc = new DOMParser().parseFromString(await res.text(), "text/html");
    const alert = doc.querySelector("[role=alert]");
    window.alert(alert?.textContent ?? "Could not update the task");
  };

  const cycle = (name, values) => {
    const current = { dir: body.dataset.dir, sort: body.dataset.sort, level: body.dataset.level }[name];
    const index = Math.max(values.indexOf(current), 0);
    const next = values[(index + 1) % values.length];
    const url = new URL(location.href);
    const defaults = { level: "all", sort: "priority", dir: "desc" };
    if (next === defaults[name]) url.searchParams.delete(name);
    else url.searchParams.set(name, next);
    location.href = `${url.pathname}${url.search}`;
  };

  const typing = (target) => target && /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName);

  for (const card of document.querySelectorAll("[data-card]")) {
    card.addEventListener("click", (event) => {
      if (event.target.closest("a, button")) return;
      selectCard(card);
    });
    card.addEventListener("dragstart", (event) => {
      event.dataTransfer.setData("text/plain", card.dataset.id);
      event.dataTransfer.effectAllowed = "move";
    });
  }

  for (const column of document.querySelectorAll("[data-column]")) {
    column.addEventListener("dragover", (event) => {
      event.preventDefault();
    });
    column.addEventListener("drop", (event) => {
      event.preventDefault();
      const id = event.dataTransfer.getData("text/plain");
      if (!id) return;
      post(`/tasks/${id}/move`, { status: column.dataset.column });
    });
  }

  let dirty = false;
  for (const el of document.querySelectorAll("input, textarea, select")) {
    el.addEventListener("input", () => {
      dirty = true;
    });
  }

  if (window.EventSource) {
    const source = new EventSource("/events");
    source.addEventListener("reload", () => {
      if (!dirty) location.reload();
    });
  }

  document.addEventListener("keydown", (event) => {
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    if (event.key === "/" && !typing(event.target)) {
      event.preventDefault();
      document.querySelector("[name=q]")?.focus();
      return;
    }
    if (typing(event.target)) return;
    const card = selectedCard();
    const id = card?.dataset.id;
    if (event.key === "?") {
      location.href = "/help";
    } else if (event.key === "c" && body.dataset.create) {
      location.href = body.dataset.create;
    } else if (event.key === "e" && id) {
      location.href = `/tasks/${id}/edit`;
    } else if (event.key === "d" && id) {
      location.href = `/tasks/${id}/archive`;
    } else if (event.key === "Enter" && card) {
      const link = card.querySelector("h3 a");
      if (link) location.href = link.href;
    } else if (event.key === "n" && id) {
      post(`/tasks/${id}/move`, { status: "next" });
    } else if (event.key === "p" && id) {
      post(`/tasks/${id}/move`, { status: "prev" });
    } else if ((event.key === "+" || event.key === "=") && id) {
      post(`/tasks/${id}/priority`, { direction: "up" });
    } else if (event.key === "-" && id) {
      post(`/tasks/${id}/priority`, { direction: "down" });
    } else if (event.key === "S" || (event.key === "s" && event.shiftKey)) {
      cycle("dir", ["desc", "asc"]);
    } else if (event.key === "s") {
      cycle("sort", ["priority", "created", "updated", "title"]);
    } else if (event.key === "v") {
      cycle("level", (body.dataset.levels || "all").split(","));
    } else {
      return;
    }
    event.preventDefault();
  });
})();
