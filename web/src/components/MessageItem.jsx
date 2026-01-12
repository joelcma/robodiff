import { formatTime } from "../utils/timeFormatter";

export default function MessageItem({ message }) {
  return (
    <div
      className={`message-item message-${
        message.level?.toLowerCase() || "info"
      }`}
    >
      <span className="message-level">{message.level}</span>
      {message.timestamp && (
        <span className="message-timestamp">
          {formatTime(message.timestamp)}
        </span>
      )}
      <span className="message-text">{message.text}</span>
    </div>
  );
}
