<script lang="ts">
  import { roomState } from '../stores/room';
  import MessageList from './MessageList.svelte';
  import MessageInput from './MessageInput.svelte';
  import TypingIndicator from './TypingIndicator.svelte';
  import UserList from './UserList.svelte';

  let { onsend, ontyping, onleave }: {
    onsend: (content: string) => void;
    ontyping: (typing: boolean) => void;
    onleave: () => void;
  } = $props();
</script>

<div class="chat-room">
  <header>
    <h2>#{$roomState.roomName}</h2>
    <span class="user-name">You are <strong>{$roomState.userName}</strong></span>
    <button onclick={onleave}>Leave</button>
  </header>
  <div class="chat-body">
    <div class="chat-main">
      <MessageList />
      <TypingIndicator />
      <MessageInput {onsend} {ontyping} />
    </div>
    <UserList />
  </div>
</div>
