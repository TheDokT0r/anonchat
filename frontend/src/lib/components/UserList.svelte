<script lang="ts">
  import { users, roomState, userColor } from '../stores/room';
</script>

<aside class="user-list">
  <h3>Users ({$users.length})</h3>
  <ul>
    {#each $users as user}
      <li style="color: {userColor(user)}">
        {user}{user === $roomState.userName ? ' (you)' : ''}
      </li>
    {/each}
  </ul>
</aside>
