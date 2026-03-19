<script lang="ts">
  import { ChatClient } from '$lib/ws/client';
  import { roomState, addMessage, setRoomJoined, setConnected, updatePresence, updateTyping, resetRoom, addSystemMessage, users } from '$lib/stores/room';
  import RoomEntry from '$lib/components/RoomEntry.svelte';
  import ChatRoom from '$lib/components/ChatRoom.svelte';

  const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws`;

  let client: ChatClient | null = null;

  function handleJoin(roomName: string) {
    client = new ChatClient(wsUrl, {
      onRoomJoined: (e) => {
        setRoomJoined(e.roomName, e.assignedName);
        // Seed initial user list without triggering join/leave messages
        users.set(e.users);
        addSystemMessage(`You joined as ${e.assignedName}`);
      },
      onChat: (e) => addMessage(e),
      onPresence: (e) => updatePresence(e.users),
      onTyping: (e) => updateTyping(e.userName, e.isTyping),
      onError: (e) => console.error('Server error:', e.message),
      onConnectionChange: (connected) => setConnected(connected),
    });
    client.join(roomName);
  }

  function handleSend(content: string) {
    addMessage({
      senderName: $roomState.userName,
      content: content,
      timestamp: Date.now(),
    });
    client?.sendChat(content);
  }

  function handleTyping(typing: boolean) {
    client?.sendTyping(typing);
  }

  function handleLeave() {
    client?.leave();
    client?.disconnect();
    client = null;
    resetRoom();
  }
</script>

<main>
  {#if $roomState.roomName}
    <ChatRoom
      onsend={handleSend}
      ontyping={handleTyping}
      onleave={handleLeave}
    />
  {:else}
    <RoomEntry onjoin={handleJoin} />
  {/if}
</main>
