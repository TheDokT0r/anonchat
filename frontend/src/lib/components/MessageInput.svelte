<script lang="ts">
  let { onsend, ontyping }: { onsend: (content: string) => void; ontyping: (typing: boolean) => void } = $props();
  let content = $state('');
  let typingTimeout: ReturnType<typeof setTimeout> | null = null;

  function handleSubmit(e: Event) {
    e.preventDefault();
    const trimmed = content.trim();
    if (!trimmed || trimmed.length > 2000) return;
    onsend(trimmed);
    content = '';
    ontyping(false);
    if (typingTimeout) clearTimeout(typingTimeout);
  }

  function handleInput() {
    ontyping(true);
    if (typingTimeout) clearTimeout(typingTimeout);
    typingTimeout = setTimeout(() => {
      ontyping(false);
    }, 2000);
  }
</script>

<form class="message-input" onsubmit={handleSubmit}>
  <input
    type="text"
    bind:value={content}
    oninput={handleInput}
    placeholder="Type a message..."
    maxlength="2000"
  />
  <button type="submit">Send</button>
</form>
