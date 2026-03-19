import { writable, get } from 'svelte/store';
import type { ChatEvent } from '../ws/client';

interface RoomState {
  roomName: string;
  userName: string;
  connected: boolean;
}

export type Message =
  | { type: 'chat'; senderName: string; content: string; timestamp: number }
  | { type: 'system'; text: string; timestamp: number };

const MAX_MESSAGES = 200;

export const roomState = writable<RoomState>({
  roomName: '',
  userName: '',
  connected: false,
});

export const messages = writable<Message[]>([]);
export const users = writable<string[]>([]);
export const typingUsers = writable<string[]>([]);

const typingTimers = new Map<string, ReturnType<typeof setTimeout>>();

export function setRoomJoined(roomName: string, userName: string) {
  roomState.set({ roomName, userName, connected: true });
}

export function setConnected(connected: boolean) {
  roomState.update((s) => ({ ...s, connected }));
}

export function addMessage(msg: ChatEvent) {
  addRawMessage({ type: 'chat', ...msg });
}

export function addSystemMessage(text: string) {
  addRawMessage({ type: 'system', text, timestamp: Date.now() });
}

function addRawMessage(msg: Message) {
  messages.update((msgs) => {
    const updated = [...msgs, msg];
    if (updated.length > MAX_MESSAGES) {
      return updated.slice(updated.length - MAX_MESSAGES);
    }
    return updated;
  });
}

export function updatePresence(userList: string[]) {
  const prev = get(users);
  const joined = userList.filter((u) => !prev.includes(u));
  const left = prev.filter((u) => !userList.includes(u));

  for (const u of joined) {
    addSystemMessage(`${u} joined`);
  }
  for (const u of left) {
    addSystemMessage(`${u} left`);
  }

  users.set(userList);
}

export function updateTyping(userName: string, isTyping: boolean) {
  if (isTyping) {
    const existing = typingTimers.get(userName);
    if (existing) clearTimeout(existing);

    typingUsers.update((current) => {
      if (!current.includes(userName)) return [...current, userName];
      return current;
    });

    const timer = setTimeout(() => {
      typingUsers.update((current) => current.filter((u) => u !== userName));
      typingTimers.delete(userName);
    }, 3000);
    typingTimers.set(userName, timer);
  } else {
    const existing = typingTimers.get(userName);
    if (existing) clearTimeout(existing);
    typingTimers.delete(userName);
    typingUsers.update((current) => current.filter((u) => u !== userName));
  }
}

export function resetRoom() {
  roomState.set({ roomName: '', userName: '', connected: false });
  messages.set([]);
  users.set([]);
  typingUsers.set([]);
  typingTimers.forEach((timer: ReturnType<typeof setTimeout>) => clearTimeout(timer));
  typingTimers.clear();
}

const USER_COLORS = [
  '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7',
  '#ec4899', '#14b8a6', '#f97316', '#6366f1', '#06b6d4',
  '#84cc16', '#e879f9', '#fb923c', '#2dd4bf', '#818cf8',
];

export function userColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return USER_COLORS[Math.abs(hash) % USER_COLORS.length];
}
