<script>
  let { item, onclose, ondelete } = $props();

  let text = $state("");
  let confirming = $state(false);
  let isWeb = $derived(/^https?:\/\//.test(item.source));

  $effect(() => {
    const id = item.id;
    let stale = false;

    confirming = false;
    text = "";

    if (!item.hasHtml) {
      fetch(`/items/${id}/text`)
        .then((res) => (res.ok ? res.text() : "could not load text"))
        .then((t) => stale || (text = t));
    }

    return () => (stale = true);
  });

  function open() {
    if (window.openExternal) window.openExternal(item.source);
    else window.open(item.source, "_blank", "noopener");
  }

  async function remove() {
    if (!confirming) {
      confirming = true;
      return;
    }

    await fetch(`/items/${item.id}`, { method: "DELETE" });
    ondelete();
  }
</script>

<article>
  <header>
    <div>
      <h2>{item.title || item.source}</h2>
      <p class="label">{item.source}</p>
    </div>
    <button class="close" aria-label="Close preview" onclick={onclose}>×</button>
  </header>

  {#if item.hasHtml}
    <!-- the server also sends sandbox -->
    <iframe sandbox="" src="/items/{item.id}/html" title={item.title || item.source}></iframe>
  {:else}
    <pre>{text}</pre>
  {/if}

  <footer>
    <button onclick={remove} onblur={() => (confirming = false)}>
      {confirming ? "really delete?" : "delete"}
    </button>
    {#if isWeb}
      <button class="primary" onclick={open}>open ↗</button>
    {/if}
  </footer>
</article>

<style>
  article {
    --notch: 44px;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 22px;
    padding: 22px;
    color: var(--carbon);
    background: var(--panel);
    clip-path: polygon(0 0, calc(100% - var(--notch)) 0, 100% var(--notch), 100% 100%, 0 100%);
  }

  header {
    display: flex;
    justify-content: space-between;
    gap: 11px;
    padding-right: var(--notch);
  }

  h2 {
    font-size: 22px;
    font-weight: normal;
    line-height: 1.2;
    margin-bottom: 6px;
  }

  .label {
    color: var(--clay);
    overflow-wrap: anywhere;
  }

  button {
    border-color: var(--carbon);
  }

  button:hover {
    color: var(--carbon);
    background: var(--foam);
  }

  .close {
    align-self: flex-start;
    border: none;
    font-size: 22px;
    line-height: 1;
    padding: 0 6px;
  }

  iframe,
  pre {
    flex: 1;
    min-height: 0;
    width: 100%;
    border: 1px solid var(--carbon);
    background: #fff;
  }

  pre {
    overflow: auto;
    padding: 22px;
    font: inherit;
    white-space: pre-wrap;
    background: var(--enamel);
  }

  footer {
    display: flex;
    justify-content: space-between;
  }

  .primary {
    margin-left: auto;
    background: var(--amber);
    border-color: var(--amber);
  }
</style>
