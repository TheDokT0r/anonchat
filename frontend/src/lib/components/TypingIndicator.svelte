<script lang="ts">
  import { typingUsers, roomState } from '../stores/room';
  import { derived } from 'svelte/store';

  const text = derived([typingUsers, roomState], ([$typingUsers, $roomState]) => {
    const filtered = $typingUsers.filter((u) => u !== $roomState.userName);
    if (filtered.length === 0) return '';
    if (filtered.length === 1) return `${filtered[0]} is typing...`;
    if (filtered.length === 2) return `${filtered[0]} and ${filtered[1]} are typing...`;
    return `${filtered.length} people are typing...`;
  });
</script>

{#if $text}
  <div class="typing-indicator">{$text}</div>
{/if}
