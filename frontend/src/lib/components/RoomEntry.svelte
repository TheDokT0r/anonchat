<script lang="ts">
  let { onjoin }: { onjoin: (roomName: string) => void } = $props();
  let roomName = $state('');
  let error = $state('');

  function validate(name: string): string | null {
    const normalized = name.trim().toLowerCase();
    if (!normalized) return 'Room name is required';
    if (normalized.length > 50) return 'Room name must be 50 characters or less';
    if (!/^[a-z0-9-]+$/.test(normalized)) return 'Only letters, numbers, and hyphens allowed';
    return null;
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    const validationError = validate(roomName);
    if (validationError) {
      error = validationError;
      return;
    }
    error = '';
    onjoin(roomName.trim().toLowerCase());
  }
</script>

<div class="room-entry">
  <h1>anonchat</h1>
  <p class="subtitle">Anonymous, ephemeral chat rooms</p>
  <form onsubmit={handleSubmit}>
    <input
      type="text"
      bind:value={roomName}
      placeholder="Enter a room name"
      maxlength="50"
      autofocus
    />
    {#if error}
      <p class="error">{error}</p>
    {/if}
    <button type="submit">Join Room</button>
  </form>
</div>
