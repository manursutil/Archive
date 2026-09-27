<script>
  import Preview from "./Preview.svelte";

  let items = $state([]);
  let query = $state("");
  let error = $state("");
  let selected = $state(null);

  async function load() {
    const q = query.trim();
    const res = await fetch(q ? `/search?q=${encodeURIComponent(q)}` : "/items");

    if (!res.ok) {
      error = await res.text();
      return;
    }

    error = "";
    items = (await res.json()) ?? [];
  }

  function toggle(item) {
    selected = selected?.id === item.id ? null : item;
  }

  function removed() {
    selected = null;
    load();
  }

  load();
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && (selected = null)} />

<form
  class="search"
  role="search"
  onsubmit={(e) => {
    e.preventDefault();
    load();
  }}
>
  <input
    type="search"
    placeholder="search..."
    aria-label="Search"
    bind:value={query}
    oninput={() => query.trim() || load()}
  />
  <button>search</button>
</form>

{#if error}
  <p class="error">{error}</p>
{/if}

<div class="layout" class:open={selected}>
  <ul>
    {#each items as item (item.id)}
      <li>
        <button
          class="card"
          class:selected={selected?.id === item.id}
          aria-pressed={selected?.id === item.id}
          onclick={() => toggle(item)}
        >
          <h2>{item.title || item.source}</h2>
          <p class="label">{item.savedAt?.slice(0, 10) ?? ""} {item.source}</p>
          <p class="excerpt">{item.excerpt ?? item.snippet}</p>
        </button>
      </li>
    {:else}
      <li class="label">{query.trim() ? "no matches" : "nothing saved yet"}</li>
    {/each}
  </ul>

  {#if selected}
    <Preview item={selected} onclose={() => (selected = null)} ondelete={removed} />
  {/if}
</div>

<style>
  .search {
    display: flex;
    gap: 11px;
    margin-bottom: 22px;
  }

  .search input {
    flex: 1;
  }

  .search > * {
    font-size: 16.5px;
  }

  .error {
    margin-bottom: 22px;
  }

  .layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-rows: 100%;
    gap: 22px;
  }

  .layout.open {
    grid-template-columns: minmax(0, 1fr) minmax(0, 2fr);
  }

  ul {
    list-style: none;
    padding: 0;
    display: grid;
    align-content: start;
    gap: 11px;
    overflow: auto;
  }

  .card {
    display: block;
    width: 100%;
    text-align: left;
    padding: 22px;
  }

  .card.selected {
    color: var(--carbon);
    background: var(--foam);
    border-color: var(--foam);
  }

  .card:not(.selected):is(:hover, :focus-visible) {
    --b: 2px;
    --w: 11px;
    --_g: #0000 90deg, var(--accent) 0;
    --_p: var(--w) var(--w) border-box no-repeat;
    color: inherit;
    border-color: transparent;
    outline: none;
    background:
      conic-gradient(from 90deg at top var(--b) left var(--b), var(--_g)) 0 0 / var(--_p),
      conic-gradient(from 180deg at top var(--b) right var(--b), var(--_g)) 100% 0 / var(--_p),
      conic-gradient(from 0deg at bottom var(--b) left var(--b), var(--_g)) 0 100% / var(--_p),
      conic-gradient(from -90deg at bottom var(--b) right var(--b), var(--_g)) 100% 100% / var(--_p);
  }

  .selected .label {
    color: var(--clay);
  }

  h2 {
    font-size: 16.5px;
    font-weight: normal;
    margin-bottom: 6px;
  }

  .label {
    margin-bottom: 11px;
    overflow-wrap: anywhere;
  }

  .excerpt {
    display: -webkit-box;
    -webkit-line-clamp: 6;
    line-clamp: 6;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .open .excerpt {
    -webkit-line-clamp: 4;
    line-clamp: 4;
  }

  @media (max-width: 767px) {
    .layout.open {
      grid-template-columns: 1fr;
    }

    .open ul {
      display: none;
    }
  }
</style>
