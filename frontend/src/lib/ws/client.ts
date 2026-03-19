export interface RoomJoinedEvent {
  roomName: string;
  assignedName: string;
  users: string[];
}

export interface ChatEvent {
  senderName: string;
  content: string;
  timestamp: number;
}

export interface TypingEvent {
  userName: string;
  isTyping: boolean;
}

export interface ChatClientCallbacks {
  onRoomJoined: (event: RoomJoinedEvent) => void;
  onChat: (event: ChatEvent) => void;
  onPresence: (event: { users: string[] }) => void;
  onTyping: (event: TypingEvent) => void;
  onError: (event: { message: string }) => void;
  onConnectionChange: (connected: boolean) => void;
}

export class ChatClient {
  private ws: WebSocket | null = null;
  private url: string;
  private callbacks: ChatClientCallbacks;
  private currentRoom: string | null = null;
  private reconnectDelay = 1000;
  private maxReconnectDelay = 30000;
  private shouldReconnect = false;

  constructor(url: string, callbacks: ChatClientCallbacks) {
    this.url = url;
    this.callbacks = callbacks;
  }

  join(roomName: string) {
    this.currentRoom = roomName;
    this.shouldReconnect = true;
    this.connect();
  }

  leave() {
    this.shouldReconnect = false;
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ leave: {} }));
      this.ws.close();
    }
    this.currentRoom = null;
  }

  sendChat(content: string) {
    this.send({ chat: { content } });
  }

  sendTyping(isTyping: boolean) {
    this.send({ typing: { isTyping } });
  }

  disconnect() {
    this.shouldReconnect = false;
    this.ws?.close();
    this.ws = null;
  }

  private connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      this.callbacks.onConnectionChange(true);
      if (this.currentRoom) {
        this.send({ join: { roomName: this.currentRoom } });
      }
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        this.handleMessage(msg);
      } catch {
        // Ignore malformed messages
      }
    };

    this.ws.onclose = () => {
      this.callbacks.onConnectionChange(false);
      if (this.shouldReconnect) {
        setTimeout(() => this.connect(), this.reconnectDelay);
        this.reconnectDelay = Math.min(
          this.reconnectDelay * 2,
          this.maxReconnectDelay
        );
      }
    };

    this.ws.onerror = () => {
      // onclose will fire after onerror
    };
  }

  private handleMessage(msg: Record<string, unknown>) {
    if (msg.roomJoined) {
      this.callbacks.onRoomJoined(msg.roomJoined as RoomJoinedEvent);
    } else if (msg.chat) {
      this.callbacks.onChat(msg.chat as ChatEvent);
    } else if (msg.presence) {
      this.callbacks.onPresence(msg.presence as { users: string[] });
    } else if (msg.typing) {
      this.callbacks.onTyping(msg.typing as TypingEvent);
    } else if (msg.error) {
      this.callbacks.onError(msg.error as { message: string });
    }
  }

  private send(data: unknown) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }
}
