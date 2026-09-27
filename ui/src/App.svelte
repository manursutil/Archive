<script>
  import Library from "./Library.svelte";
  import Add from "./Add.svelte";

  let route = $state(location.hash);
</script>

<svelte:window onhashchange={() => (route = location.hash)} />

<header>
  <h1><a href="#/"><span>Archive</span></a></h1>
  <nav>
    {#if route === "#add"}
      <a href="#/">← library</a>
    {:else}
      <a href="#add">+ add</a>
    {/if}
  </nav>
</header>

<main>
  {#if route === "#add"}
    <Add />
  {:else}
    <Library />
  {/if}
</main>

<style>
  header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    padding: 44px 22px 22px;
  }

  h1 {
    font-size: 44px;
    line-height: 1;
  }

  h1 a {
    text-decoration: none;
  }

  h1 span {
    color: var(--soot);
    background: var(--aluminum);
  }

  h1 a:hover span {
    background: var(--foam);
  }

  nav {
    font-size: 16.5px;
  }

  main {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 0 22px 22px;
  }
</style>
