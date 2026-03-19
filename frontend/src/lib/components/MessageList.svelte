<script lang="ts">
  import { messages, userColor } from '../stores/room';
  import { afterUpdate } from 'svelte';

  let container: HTMLDivElement;
  let shouldAutoScroll = true;

  function handleScroll() {
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    shouldAutoScroll = scrollHeight - scrollTop - clientHeight < 50;
  }

  afterUpdate(() => {
    if (shouldAutoScroll && container) {
      container.scrollTop = container.scrollHeight;
    }
  });
</script>

<div class="message-list" bind:this={container} onscroll={handleScroll}>
  {#each $messages as msg}
    {#if msg.type === 'system'}
      <div class="message system"><em>{msg.text}</em></div>
    {:else}
      <div class="message">
        <span class="sender" style="color: {userColor(msg.senderName)}">{msg.senderName}</span>
        <span class="content">{msg.content}</span>
      </div>
    {/if}
  {/each}
</div>
