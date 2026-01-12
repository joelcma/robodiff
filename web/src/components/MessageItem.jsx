import { formatTime } from "../utils/timeFormatter";
import { splitTextByJsonAssignments } from "../utils/jsonPrettify";

export default function MessageItem({ message }) {
  const segments = splitTextByJsonAssignments(message.text);

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
      <span className="message-text">
        {segments.map((seg, i) => {
          if (seg.type === "json") {
            return (
              <span key={i} className="argument-json-block">
                <span className="argument-key">{seg.key}=</span>
                <pre className="argument-json">{seg.pretty}</pre>
              </span>
            );
          }
          return <span key={i}>{seg.value}</span>;
        })}
      </span>
    </div>
  );
}
