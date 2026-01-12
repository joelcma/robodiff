export function formatTime(timeStr) {
  if (!timeStr) return "";
  // Robot Framework timestamp format: 20241231 12:34:56.789
  // Convert to readable format
  const match = timeStr.match(
    /(\d{4})(\d{2})(\d{2}) (\d{2}):(\d{2}):(\d{2})\.(\d{3})/
  );
  if (!match) return timeStr;
  const [, year, month, day, hour, min, sec, ms] = match;
  return `${hour}:${min}:${sec}.${ms}`;
}
