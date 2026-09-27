<script>
  let source = $state("");
  let status = $state("");
  let error = $state(false);
  let saving = $state(false);

  const canPick = "pickFile" in window;

  async function pick() {
    try {
      source = (await window.pickFile()) || source;
    } catch (err) {
      error = true;
      status = `file picker unavailable: ${err}`;
    }
  }

  async function save(e) {
    e.preventDefault();
    saving = true;
    status = "saving...";

    const res = await fetch("/items", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ source: source.trim() }),
    });

    saving = false;
    error = !res.ok;
    status = res.ok ? `saved ${source.trim()}` : await res.text();
    if (res.ok) source = "";
  }
</script>

<section>
  <h2 class="label">Add to archive</h2>

  <form onsubmit={save}>
    <input
      placeholder="https://... or /path/to/file.md"
      aria-label="URL or file path"
      required
      disabled={saving}
      bind:value={source}
    />
    {#if canPick}
      <button type="button" disabled={saving} onclick={pick}>choose file…</button>
    {/if}
    <button disabled={saving}>save</button>
  </form>

  <p class="status" class:error aria-live="polite">{status}</p>

  <p class="callout">
    Web pages are fetched and saved with their HTML.<br />
    .txt, .md, .pdf, .docx and .odt files are read from disk. Use an absolute path{canPick ? " or choose a file" : ""}.
  </p>
</section>

<style>
  section {
    max-width: 88ch;
  }

  h2 {
    font-size: 11px;
    font-weight: normal;
    margin-bottom: 11px;
  }

  form {
    display: flex;
    gap: 11px;
  }

  input {
    flex: 1;
  }

  form > * {
    font-size: 16.5px;
    white-space: nowrap;
  }

  .status {
    min-height: 1.4em;
    margin: 11px 0 22px;
  }

  .callout {
    color: var(--mud);
    border-left: 7px solid var(--mud);
    padding: 8px 0 8px 12px;
  }
</style>
